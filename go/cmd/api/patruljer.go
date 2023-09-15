package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

func (app *application) listPatruljerHandler(w http.ResponseWriter, r *http.Request) {
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

	patruljer, metadata, err := app.models.Teams.GetPatruljer(input.Filters)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"metadata":  metadata,
		"patruljer": patruljer,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) viewPatruljerHandler(w http.ResponseWriter, r *http.Request) {
	filters := data.Filters{}
	filters.TeamID = types.TeamID(app.ReadNamedParam(r, "id"))
	patrulje, err := app.models.Teams.GetPatrulje(filters.TeamID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	spejdere, _, err := app.models.Members.GetSpejdere(filters)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
	soses, err := app.models.Sos.GetByTeam(filters.TeamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
	envelope := jsonapi.Envelope{
		"patrulje": patrulje,
		"spejdere": spejdere,
		"soses":    soses,
		"url":      fmt.Sprintf("https://tilmelding.nathejk.dk/spejder/%s:%s", filters.TeamID, types.LegacyID(filters.TeamID).Checksum()),
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
