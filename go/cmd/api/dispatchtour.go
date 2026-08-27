package main

import (
	"net/http"
	"strconv"

	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/dispatch"
)

// The tour half of kørsel (PRD 009, task 111): one car's run, from A to B with as many stops
// as it takes.
//
// Every transition is its own route because the driver's screen is another app in another repo
// and these are what it will call — `POST …/stop/:stopId/visited` most of all. Folding them
// into a PATCH on the tour would mean building them a second time later.

// createDispatchTourHandler opens a tour.
//
// @Summary     Create a tour
// @Description Opens a run for one dispatch unit. The unit is a dispatchable subsection — not a car and not a person — so the tour survives a vehicle being swapped mid-night, and the unit is recorded at planning time rather than followed, so a car moved to another subsection later does not silently change who owned this tour. Departure and notes are optional; stops are set by their own call.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       body body object{sectionSlug=string,departureUts=int,notes=string} true "The tour"
// @Success     201 {object} map[string]interface{} "envelope with \"tourId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour [post]
func (app *application) createDispatchTourHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SectionSlug  types.Slug `json:"sectionSlug"`
		DepartureUts *int64     `json:"departureUts"`
		Notes        string     `json:"notes"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	id, err := app.commands.Dispatch.CreateTour(r.Context(), app.dispatchActor(r), app.YearSlug(r), dispatch.CreateTourCommand{
		SectionSlug:  input.SectionSlug,
		DepartureUts: input.DepartureUts,
		Notes:        input.Notes,
	})
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"tourId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// patchDispatchTourHandler edits departure, unit or notes.
//
// @Summary     Edit a tour
// @Description Changes the tour's unit, planned departure or notes. Partial: an absent field is left alone, an explicit null clears the departure. Changing the departure deliberately does *not* re-derive the stops' planned times — that would throw away overrides a dispatcher typed; ask for the stops again if you want them re-derived. Resubmitting unchanged values publishes nothing and answers 200.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Tour id"
// @Param       body body object{sectionSlug=string,departureUts=int,notes=string} true "The fields to change"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id} [patch]
func (app *application) patchDispatchTourHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TourID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		SectionSlug  *types.Slug `json:"sectionSlug"`
		DepartureUts optionalUts `json:"departureUts"`
		Notes        *string     `json:"notes"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Dispatch.PatchTour(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, dispatch.PatchTourCommand{
		SectionSlug:  input.SectionSlug,
		DepartureUts: input.DepartureUts.patch(),
		Notes:        input.Notes,
	})
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"tourId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// setDispatchTourStopsHandler sets the whole ordered stop list.
//
// @Summary     Set a tour's stops
// @Description Replaces the tour's ordered stops in one call, each with the tasks actioned there. The whole list rather than per-stop add/remove/move endpoints, because a reorder is one operator intent and three calls would make a half-applied reorder representable — the shape `/api/sections/sorted` already uses. The array order is the order. A stop with no id is new and gets one; send an existing id to keep a stop's identity through a reorder. Planned times are derived from the departure plus a per-leg allowance, anchored on the actual time of any visited stop, and any stop may carry its own plannedUts as an override, from which the following stops re-derive. Refused: reordering or dropping a visited stop, and ordering a task's unload before its load. Returned as warnings rather than refusals: exceeding the vehicle's seats — seats fold down, and a system that refuses the real world gets worked around.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Tour id"
// @Param       body body setTourStopsRequest true "The ordered stops"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\" and a \"warnings\" array"
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id}/stops [put]
func (app *application) setDispatchTourStopsHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TourID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input setTourStopsRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	stops := make([]dispatch.StopInput, 0, len(input.Stops))
	for _, s := range input.Stops {
		tasks := make([]dispatch.StopTask, 0, len(s.Tasks))
		for _, t := range s.Tasks {
			tasks = append(tasks, dispatch.StopTask{TaskID: t.TaskID, Role: t.Role})
		}
		stops = append(stops, dispatch.StopInput{
			StopID:     s.StopID,
			Place:      s.Place.domain(),
			PlannedUts: s.PlannedUts,
			Tasks:      tasks,
		})
	}
	warnings, err := app.commands.Dispatch.SetStops(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, stops)
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	// The seat check lives here, not in the domain: seat counts are on the vehicle, and the
	// dispatch entity deliberately knows nothing about vehicles — a unit is a subsection of the
	// organisation tree (PRD 009 §8). Appended to whatever the domain warned about.
	warnings = append(warnings, app.seatWarnings(r, id)...)
	if warnings == nil {
		warnings = []dispatch.Warning{}
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"tourId": id, "warnings": warnings}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

type setTourStopsRequest struct {
	Stops []struct {
		StopID     dispatch.StopID `json:"stopId"`
		Place      dispatchPlace   `json:"place"`
		PlannedUts *int64          `json:"plannedUts"`
		Tasks      []struct {
			TaskID dispatch.TaskID `json:"taskId"`
			Role   dispatch.Role   `json:"role"`
		} `json:"tasks"`
	} `json:"stops"`
}

// seatWarnings reports a tour carrying more people than the car has seats.
//
// Counted across the tour's *unvisited* pickup stops, since scouts already dropped at HQ have
// left the car. A warning and never a refusal (PRD 009 §11, answer 8): seats get folded down, a
// member sits with a leader, and the desk knows things the platform does not. Silence when the
// unit has no vehicle — that is a readiness problem, and task 116 is where the board says so.
func (app *application) seatWarnings(r *http.Request, id dispatch.TourID) []dispatch.Warning {
	year := app.YearSlug(r)
	tour, err := app.models.Dispatch.GetTour(r.Context(), year, id)
	if err != nil || tour == nil {
		return nil
	}
	seats := 0
	vehicles, err := app.models.Vehicle.GetAll(r.Context(), vehicle.Filter{YearSlug: year, SectionSlug: tour.SectionSlug})
	if err != nil {
		return nil
	}
	for _, v := range vehicles {
		seats += int(v.SeatCount)
	}
	if seats == 0 {
		return nil
	}

	ids := []dispatch.TaskID{}
	for _, s := range tour.Stops {
		if s.Visited() {
			continue
		}
		for _, st := range s.Tasks {
			if st.Role == dispatch.RoleUnload {
				continue
			}
			ids = append(ids, st.TaskID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	people := 0
	for _, taskID := range ids {
		task, err := app.models.Dispatch.GetTask(r.Context(), year, taskID)
		if err != nil || task == nil || task.Kind != dispatch.KindPickup {
			continue
		}
		// A pickup that names nobody still carries somebody: the operator wrote down a scout
		// without linking the member record, which is common at 3am. Counted as one, because
		// counting it as zero is how a full car looks empty.
		if n := len(task.MemberIDs); n > 0 {
			people += n
		} else {
			people++
		}
	}
	if people <= seats {
		return nil
	}
	return []dispatch.Warning{{
		Code:    "seats",
		Message: pluralSeats(people, seats),
	}}
}

func pluralSeats(people, seats int) string {
	return "Turen skal hente " + strconv.Itoa(people) + " personer, men bilen har " + strconv.Itoa(seats) + " pladser"
}

// startDispatchTourHandler records that the car has set off.
//
// @Summary     A tour has set off
// @Description Marks the tour underway, and its tasks with it. Idempotent. Marking a stop visited also does this implicitly: a tour whose stops are being ticked off while it still says "planned" would have the desk looking for a car it thinks has not left.
// @Tags        dispatch
// @Produce     json
// @Param       id path string true "Tour id"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id}/underway [post]
func (app *application) startDispatchTourHandler(w http.ResponseWriter, r *http.Request) {
	app.tourTransition(w, r, func(id dispatch.TourID) error {
		return app.commands.Dispatch.StartTour(r.Context(), app.dispatchActor(r), app.YearSlug(r), id)
	})
}

// visitDispatchTourStopHandler records a stop reached.
//
// @Summary     A stop has been reached
// @Description Marks one stop visited and progresses the tasks actioned there: a task is completed by its unload or by its single action, never by its load — a scout collected at Post 2B is aboard, not delivered, and completing the task there would take them off the board while they are still in a car. Visiting a stop also starts the tour if it was still planned. Visiting an already visited stop is harmless and answers 200. This is the endpoint a driver app will call most.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Tour id"
// @Param       stopId path string true "Stop id"
// @Param       body body object{atUts=int} false "When, if not now"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\" and \"stopId\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id}/stop/{stopId}/visited [post]
func (app *application) visitDispatchTourStopHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TourID(app.ReadNamedParam(r, "id"))
	stopID := dispatch.StopID(app.ReadNamedParam(r, "stopId"))
	if id == "" || stopID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		AtUts int64 `json:"atUts"`
	}
	// An empty body is normal here — "reached, now" — so a decode failure on no content must
	// not be an error. ReadJSON is only consulted when there is something to read.
	if r.ContentLength > 0 {
		if err := app.ReadJSON(w, r, &input); err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	err := app.commands.Dispatch.VisitStop(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, stopID, input.AtUts)
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"tourId": id, "stopId": stopID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// completeDispatchTourHandler closes a tour.
//
// @Summary     Complete a tour
// @Description Closes a tour whose stops have all been visited. Refused while any stop is unvisited: the alternative is a tour marked done with a task silently stranded on it, and the desk would never see the job it dropped — remove the stop, or cancel the tour. Completing an already completed tour answers 200.
// @Tags        dispatch
// @Produce     json
// @Param       id path string true "Tour id"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id}/completed [post]
func (app *application) completeDispatchTourHandler(w http.ResponseWriter, r *http.Request) {
	app.tourTransition(w, r, func(id dispatch.TourID) error {
		return app.commands.Dispatch.CompleteTour(r.Context(), app.dispatchActor(r), app.YearSlug(r), id)
	})
}

// cancelDispatchTourHandler abandons a tour.
//
// @Summary     Cancel a tour
// @Description Abandons a tour, with a required reason, and returns its unvisited work to the queue — each task keeping its original waiting clock, because the scout has been waiting since the call and not since the re-plan. This is also how a broken-down car is modelled: no special case, the tour is cancelled and its stops go back. Tasks already unloaded at a visited stop stay done, and a task that has since been put on another tour is left alone.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Tour id"
// @Param       body body object{reason=string} true "Why"
// @Success     200 {object} map[string]interface{} "envelope with \"tourId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/tour/{id}/cancelled [post]
func (app *application) cancelDispatchTourHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TourID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Dispatch.CancelTour(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, input.Reason)
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"tourId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// tourTransition is the shape the bodyless transitions share.
func (app *application) tourTransition(w http.ResponseWriter, r *http.Request, do func(dispatch.TourID) error) {
	id := dispatch.TourID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if err := do(id); err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"tourId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// tourTransition is the shape the bodyless transitions share.
