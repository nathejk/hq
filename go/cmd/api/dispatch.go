package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/dispatch"
)

// Kørsel — the dispatch desk (PRD 009). What needs moving, which car is doing it, and when.
//
// # Why the transitions are their own routes
//
// The drivers will get a screen, and it will be another app in another repo (PRD 009 §4, §8) —
// as the post scanner already is. That makes these endpoints an integration surface rather
// than a private back end for one Vue view: `POST …/pickedup` is exactly what a driver app
// would call, so it stays a first-class route rather than a field on a PATCH that would have
// to be built again later.
//
// # One board, one payload
//
// GET /api/dispatch answers a whole screen: the queue, the tours with their stops, and the
// units. One round trip because the board renders all three together and the operator moves
// between them constantly — and one cache key in the SPA, so a task dropped into a tour
// invalidates the whole board rather than leaving two panes disagreeing about where it is.

// dispatchActor is who the dispatch desk records as having acted.
//
// The fourth of these little conversions (sos, spejderstatus, spejdernote, dispatch), and
// deliberately still not a shared types.Actor: every one of these packages is written to be
// liftable to shared-go independently and none may import another, so a shared struct is a
// cross-repo change, not a local tidy-up. Worth doing when somebody is in shared-go anyway.
//
// Empty in practice until HQ has login. That matters less here than elsewhere, because the
// fact the desk actually needs — *which unit* took the job — is an explicit choice on the
// tour, not something inferred from who is typing (PRD 009 §8).
func (app *application) dispatchActor(r *http.Request) dispatch.Actor {
	user := app.actor(r)
	return dispatch.Actor{UserID: user.UserID, Name: user.Name}
}

// dispatchUnit is one dispatchable subsection, with the car and the people in it.
//
// Assembled here rather than stored, because it already exists in the organisation tree: a
// subsection holding a vehicle and a crew member *is* a dispatch unit (PRD 009 §8), and the
// only new fact is the `dispatchable` flag. Nothing in the dispatch entity knows about
// vehicles or crew, which is why this join lives in the handler.
type dispatchUnit struct {
	SectionSlug types.Slug `json:"sectionSlug"`
	Label       string     `json:"label"`

	// Vehicles is a list because more than one car in a dispatch unit is a configuration
	// mistake that is flagged rather than forbidden (PRD 009 §6): the desk can still work,
	// and the Organisation page is where it gets fixed. A single field would have to choose
	// one silently.
	Vehicles []vehicle.Vehicle `json:"vehicles"`

	// People are the crew members in the subsection. The vehicle's own driverUserId names
	// the driver; anybody else here is a co-driver.
	People []crewmember.CrewMember `json:"people"`
}

// showDispatchBoardHandler serves the whole board.
//
// @Summary     The dispatch board
// @Description Everything the kørsel screen renders: the tasks (queue first, oldest first), the tours with their ordered stops and the tasks at each, and the dispatchable units with their vehicles and crew. One payload because the board is one screen and a dispatcher moves between its panes constantly; one cache key in the SPA, so a task dragged into a tour cannot leave two panes disagreeing about where it is. Every collection is an array, never null. Year comes from the X-YearSlug header, or the current year.
// @Tags        dispatch
// @Produce     json
// @Success     200 {object} map[string]interface{} "envelope with \"tasks\", \"tours\" and \"units\" arrays"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch [get]
func (app *application) showDispatchBoardHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)

	// Every state, in one query. The queue, the tours' tasks and the night's finished work
	// are the same list seen three ways, and this is fewer than a few hundred rows for a
	// whole race — a filtered query per pane would be three round trips to render one screen.
	tasks, err := app.models.Dispatch.Tasks(r.Context(), dispatch.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	tours, err := app.models.Dispatch.Tours(r.Context(), dispatch.TourFilter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	units, err := app.dispatchUnits(r, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	// Never null, always an array. The queries already promise this, and the handler promises
	// it again: a null collection has broken rendering in this repo three times, and the cost
	// of the belt as well as the braces is two lines.
	if tasks == nil {
		tasks = []*dispatch.Task{}
	}
	if tours == nil {
		tours = []*dispatch.Tour{}
	}

	envelope := jsonapi.Envelope{
		"tasks": tasks,
		"tours": tours,
		"units": units,
		// The vocabularies the dialog offers, sent from here for the reason the korps list
		// is: a client that invents its own would store values nothing can filter on.
		"kinds":      []dispatch.Kind{dispatch.KindPickup, dispatch.KindTransport, dispatch.KindCollection, dispatch.KindDelivery},
		"priorities": []dispatch.Priority{dispatch.PriorityGreen, dispatch.PriorityYellow, dispatch.PriorityRed},
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// dispatchUnits builds the units from the organisation tree.
func (app *application) dispatchUnits(r *http.Request, year types.YearSlug) ([]dispatchUnit, error) {
	slugs, err := app.models.Dispatch.DispatchableSections(r.Context(), year)
	if err != nil {
		return nil, err
	}
	units := []dispatchUnit{}
	if len(slugs) == 0 {
		// No unit configured is a normal state, not an error: the queue still accepts tasks,
		// and the board says so rather than showing an empty strip with no explanation.
		return units, nil
	}
	sections, err := app.models.Section.GetAll(r.Context(), section.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	labels := map[types.Slug]string{}
	for _, s := range sections {
		labels[s.Slug] = s.Label
	}
	vehicles, err := app.models.Vehicle.GetAll(r.Context(), vehicle.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	people, err := app.models.CrewMember.GetAll(r.Context(), crewmember.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	// One pass over each collection rather than a query per unit: fewer than ten units, and
	// a query per unit would be a dozen round trips to draw one strip.
	for _, slug := range slugs {
		unit := dispatchUnit{
			SectionSlug: slug,
			// Fall back to the slug rather than showing an empty row: a unit whose section
			// has been renamed away is exactly the drift PRD 009 §8 warns about, and it must
			// be visible, not invisible.
			Label:    labels[slug],
			Vehicles: []vehicle.Vehicle{},
			People:   []crewmember.CrewMember{},
		}
		if unit.Label == "" {
			unit.Label = string(slug)
		}
		for _, v := range vehicles {
			if v.SectionSlug == slug {
				unit.Vehicles = append(unit.Vehicles, v)
			}
		}
		for _, p := range people {
			if p.SectionSlug == slug {
				unit.People = append(unit.People, p)
			}
		}
		units = append(units, unit)
	}
	return units, nil
}

// dispatchPlace is a place as the client sends it.
type dispatchPlace struct {
	Kind  dispatch.PlaceKind `json:"kind"`
	RefID string             `json:"refId"`
	Label string             `json:"label"`
}

func (p dispatchPlace) domain() dispatch.Place {
	return dispatch.Place{Kind: p.Kind, RefID: p.RefID, Label: p.Label}
}

// optionalUts is a nullable time that can tell "absent" from "explicitly null".
//
// It exists because `**int64` cannot: encoding/json decodes `"deadlineUts": null` by leaving the
// outer pointer nil, exactly as it leaves an absent field, so the two requests are
// indistinguishable — and "clear the deadline" would silently become "leave it alone". A type
// implementing Unmarshaler is called even for a JSON null, which is what makes the distinction
// recoverable. Found by the test that asserts it, not by reading the docs.
type optionalUts struct {
	Set   bool
	Value *int64
}

func (o *optionalUts) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v int64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// patch returns the double pointer the command takes: nil for absent, otherwise a pointer to
// the new value — which may itself be nil, meaning cleared.
func (o optionalUts) patch() **int64 {
	if !o.Set {
		return nil
	}
	v := o.Value
	return &v
}

// createDispatchTaskHandler opens a task.
//
// @Summary     Create a dispatch task
// @Description Writes down something that needs moving: a scout collected from a roadside, maps between two posts, materials out of a closed Start, dinner to the loks. Only the kind and a description are required — almost everything else is optional on purpose, because the board is only as good as the desk's discipline and the written path has to be the fastest path. Places are a type plus a label and may be free text ("på Slangerupvej ved skovbrynet"), which is the normal case rather than a fallback. Priority is the nødtelefon's own vocabulary (green/yellow/red), so a pickup created from a red case can arrive red. createdUts may be backdated: a patrol that rang twenty minutes ago has been waiting twenty minutes.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       body body createDispatchTaskRequest true "The task"
// @Success     201 {object} map[string]interface{} "envelope with \"taskId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/task [post]
func (app *application) createDispatchTaskHandler(w http.ResponseWriter, r *http.Request) {
	var input createDispatchTaskRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	id, err := app.commands.Dispatch.CreateTask(r.Context(), app.dispatchActor(r), app.YearSlug(r), dispatch.CreateTaskCommand{
		Kind:         input.Kind,
		Priority:     input.Priority,
		Description:  input.Description,
		SpaceNeeds:   input.SpaceNeeds,
		Pickup:       input.Pickup.domain(),
		Dropoff:      input.Dropoff.domain(),
		CreatedUts:   input.CreatedUts,
		NotBeforeUts: input.NotBeforeUts,
		DeadlineUts:  input.DeadlineUts,
		SosID:        input.SosID,
		TeamID:       input.TeamID,
		MemberIDs:    input.MemberIDs,
	})
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"taskId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

type createDispatchTaskRequest struct {
	Kind        dispatch.Kind     `json:"kind"`
	Priority    dispatch.Priority `json:"priority"`
	Description string            `json:"description"`
	SpaceNeeds  string            `json:"spaceNeeds"`
	Pickup      dispatchPlace     `json:"pickup"`
	Dropoff     dispatchPlace     `json:"dropoff"`

	CreatedUts   int64  `json:"createdUts"`
	NotBeforeUts *int64 `json:"notBeforeUts"`
	DeadlineUts  *int64 `json:"deadlineUts"`

	SosID     types.SosID      `json:"sosId"`
	TeamID    types.TeamID     `json:"teamId"`
	MemberIDs []types.MemberID `json:"memberIds"`
}

// showDispatchTaskHandler serves one task with its timeline.
//
// @Summary     One dispatch task
// @Description The task, the stops it occupies on tours (which is where the answer to "when?" comes from — a planned time made by a human who knows the roads), and its full timeline. A dispatch desk is a log first: every state change is on the timeline with a timestamp, so a shift handover loses nothing.
// @Tags        dispatch
// @Produce     json
// @Param       id path string true "Task id"
// @Success     200 {object} map[string]interface{} "envelope with \"task\""
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/task/{id} [get]
func (app *application) showDispatchTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TaskID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	task, err := app.models.Dispatch.GetTask(r.Context(), app.YearSlug(r), id)
	if err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
			return
		}
		app.ServerErrorResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"task": task}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// patchDispatchTaskHandler edits a task.
//
// @Summary     Edit a dispatch task
// @Description Changes times, description, priority, space needs or either place. A partial update: an absent field is left alone and an explicit null clears it, which is why notBeforeUts and deadlineUts are distinguishable as "not mentioned" and "cleared". Resubmitting unchanged values changes nothing, publishes nothing and answers 200 — so a client relying only on the live signal to confirm a save must also refresh. A finished or cancelled task cannot be edited: the timeline is the record of what the desk promised.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Task id"
// @Param       body body map[string]interface{} true "The fields to change"
// @Success     200 {object} map[string]interface{} "envelope with \"taskId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/task/{id} [patch]
func (app *application) patchDispatchTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TaskID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	// Pointers to pointers for the times in the command, and a small Unmarshaler here: an
	// absent field leaves the time alone while an explicit null clears it, and encoding/json
	// cannot tell those apart on its own.
	var input struct {
		Kind        *dispatch.Kind     `json:"kind"`
		Priority    *dispatch.Priority `json:"priority"`
		Description *string            `json:"description"`
		SpaceNeeds  *string            `json:"spaceNeeds"`
		Pickup      *dispatchPlace     `json:"pickup"`
		Dropoff     *dispatchPlace     `json:"dropoff"`

		NotBeforeUts optionalUts `json:"notBeforeUts"`
		DeadlineUts  optionalUts `json:"deadlineUts"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	cmd := dispatch.PatchTaskCommand{
		Kind:         input.Kind,
		Priority:     input.Priority,
		Description:  input.Description,
		SpaceNeeds:   input.SpaceNeeds,
		NotBeforeUts: input.NotBeforeUts.patch(),
		DeadlineUts:  input.DeadlineUts.patch(),
	}
	if input.Pickup != nil {
		place := input.Pickup.domain()
		cmd.Pickup = &place
	}
	if input.Dropoff != nil {
		place := input.Dropoff.domain()
		cmd.Dropoff = &place
	}
	if err := app.commands.Dispatch.PatchTask(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, cmd); err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"taskId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// dispatchTaskPickedUpHandler records people aboard.
//
// @Summary     People aboard
// @Description Records that the people a pickup is for are in the car. A distinct moment from completion — the car still has to get to HQ — and the moment custody changes, which is why it is recorded at all. The unit is a section slug rather than a vehicle id: the unit is who took them, and it survives a car being swapped mid-night. Only a pickup can record this, and pressing it twice is harmless. Until the drivers have their own screen the dispatcher records it on the driver's behalf; the member transitions to transit hang off this endpoint (task 118).
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Task id"
// @Param       body body object{sectionSlug=string,atUts=int} false "The unit, and when"
// @Success     200 {object} map[string]interface{} "envelope with \"taskId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/task/{id}/pickedup [post]
func (app *application) dispatchTaskPickedUpHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TaskID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		SectionSlug types.Slug `json:"sectionSlug"`
		AtUts       int64      `json:"atUts"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Dispatch.MarkPickedUp(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, input.SectionSlug, input.AtUts)
	if err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"taskId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// cancelDispatchTaskHandler withdraws a task.
//
// @Summary     Cancel a dispatch task
// @Description Withdraws a task, with a required reason — a cancelled task with no explanation is the one thing a shift handover cannot recover from. Cancelling an already cancelled task answers 200, because two operators pressing the same button is a race the desk should not have to think about; cancelling a completed one is refused, because somebody has misread the board. The task leaves any tour it was in.
// @Tags        dispatch
// @Accept      json
// @Produce     json
// @Param       id path string true "Task id"
// @Param       body body object{reason=string} true "Why"
// @Success     200 {object} map[string]interface{} "envelope with \"taskId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/dispatch/task/{id}/cancelled [post]
func (app *application) cancelDispatchTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := dispatch.TaskID(app.ReadNamedParam(r, "id"))
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
	if err := app.commands.Dispatch.CancelTask(r.Context(), app.dispatchActor(r), app.YearSlug(r), id, input.Reason); err != nil {
		app.dispatchCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"taskId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// dispatchCommandError maps the dispatch commands' refusals onto responses.
//
// Danish, and phrased as what is wrong with the request rather than as the rule it broke —
// the operator reading it is tired and is not going to read the source.
func (app *application) dispatchCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, dispatch.ErrEmptyDescription):
		app.FailedValidationResponse(w, r, map[string]string{"description": "opgaven skal have en beskrivelse"})
	case errors.Is(err, dispatch.ErrInvalidKind):
		app.FailedValidationResponse(w, r, map[string]string{"kind": "ukendt opgavetype"})
	case errors.Is(err, dispatch.ErrInvalidPriority):
		app.FailedValidationResponse(w, r, map[string]string{"priority": "ukendt prioritet"})
	case errors.Is(err, dispatch.ErrReasonRequired):
		app.FailedValidationResponse(w, r, map[string]string{"reason": "angiv en årsag"})
	case errors.Is(err, dispatch.ErrNotPickup):
		app.FailedValidationResponse(w, r, map[string]string{"kind": "kun en hentning kan have folk med i bilen"})
	case errors.Is(err, dispatch.ErrTaskFinished):
		app.FailedValidationResponse(w, r, map[string]string{"state": "opgaven er afsluttet"})
	case errors.Is(err, tables.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}
