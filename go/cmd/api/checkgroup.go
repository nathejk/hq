package main

import (
	"fmt"
	"net/http"
	"time"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/validator"
)

func (app *application) listCheckgroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	status, metadata, err := app.models.GetCheckgroupsStatus(input.Filters)
	//checkgroups, metadata, err := app.models.Checkgroups.GetAll(input.Filters) //, startedTeamIDs, discontinuedTeamIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"metadata": metadata,
		"status":   status,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

/*
func (a *application) checkgroupListHandler(w http.ResponseWriter, r *http.Request) {
	id, err := a.ReadIDParam(r)
	if err != nil {
		a.NotFoundResponse(w, r)
		return
	}
	resp, err := a.nova.StudyTemplateTasks(socrat.StudyTemplateID(id))
	if err != nil {
		a.BadRequestResponse(w, r, err)
		return
	}
	tasks, err := a.tasks(resp.Resources)
	if err != nil {
		a.BadRequestResponse(w, r, err)
		return
	}

	template := struct {
		title           string
		description     string
		stimuliMinCount int
		stimuliMaxCount int
	}{}
	tmplResp, err := a.nova.StudyTemplate(socrat.StudyTemplateID(id))
	if err != nil {
		a.BadRequestResponse(w, r, err)
		return
	}

	env := app.Envelope{
		"title":           template.title,
		"description":     template.description,
		"stimuliMinCount": template.stimuliMinCount,
		"stimuliMaxCount": template.stimuliMaxCount,
		"tasks":           tasks,
	}
	err = a.WriteJSON(w, http.StatusAccepted, env, nil)
	if err != nil {
		a.ServerErrorResponse(w, r, err)
	}
}
*/
