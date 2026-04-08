package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/year"
)

func (app *application) listYearHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Year = app.ReadString(qs, "year", fmt.Sprintf("%d", time.Now().Year()))
	input.Filters.Page = app.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize = app.ReadInt(qs, "page_size", 1000, v)
	input.Filters.Sort = app.ReadString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id"}

	if input.Filters.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}
	years, err := app.models.Year.GetAll(r.Context(), year.Filter{})
	//status, metadata, err := app.models.GetCheckgroupsStatus(input.Filters)
	//checkgroups, metadata, err := app.models.Checkgroups.GetAll(input.Filters) //, startedTeamIDs, discontinuedTeamIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"years": years,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) yearHandler(w http.ResponseWriter, r *http.Request) {
	cgID := types.CheckgroupID(app.ReadNamedParam(r, "id"))
	cg, err := app.models.Checkgroup.GetByID(r.Context(), cgID)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	cps, _ := app.models.Checkpoint.GetAll(r.Context(), checkpoint.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	scanners, _ := app.models.Checkpersonnel.GetAll(r.Context(), checkpersonnel.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	users, _ := app.models.Personnel.GetAll(r.Context(), personnel.Filter{YearSlug: cg.YearSlug, UserTypes: []string{"friend"}})

	envelope := jsonapi.Envelope{
		"checkgroup":  cg,
		"checkpoints": cps,
		"scanners":    scanners,
		"users":       users,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) createYearHandler(w http.ResponseWriter, r *http.Request) {
	slug := types.YearSlug(app.ReadNamedParam(r, "slug"))
	if err := app.commands.Year.Create(r.Context(), slug); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"created": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateYearHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Headline        *string     `json:"headline"`
		Description     *string     `json:"description"`
		CityDeparture   *string     `json:"cityDeparture"`
		CityDestination *string     `json:"cityDestination"`
		DateStart       *types.Date `json:"dateStart"`
		DateEnd         *types.Date `json:"dateEnd"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	slug := types.YearSlug(app.ReadNamedParam(r, "slug"))
	cmd := year.UpdateCommand{
		Headline:        input.Headline,
		Description:     input.Description,
		CityDeparture:   input.CityDeparture,
		CityDestination: input.CityDestination,
		DateStart:       input.DateStart,
		DateEnd:         input.DateEnd,
	}
	err := app.commands.Year.Update(r.Context(), slug, cmd)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"updated": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) deleteYearHandler(w http.ResponseWriter, r *http.Request) {
	slug := types.YearSlug(app.ReadNamedParam(r, "slug"))
	if err := app.commands.Year.Delete(r.Context(), slug); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
