package main

import (
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
)

func (app *application) createCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CheckpointID string     `json:"checkpointId"`
		UserID       string     `json:"userId"`
		Start        *time.Time `json:"start"`
		End          *time.Time `json:"end"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	var tr *types.TimeRange
	if input.Start != nil && input.End != nil {
		tr = &types.TimeRange{
			Start: *input.Start,
			End:   *input.End,
		}
	}

	yearSlug := app.YearSlug(r)
	checkpersonnelID, err := app.commands.Checkpersonnel.Create(r.Context(), yearSlug, types.CheckpointID(input.CheckpointID), types.UserID(input.UserID), tr)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	envelope := jsonapi.Envelope{
		"checkpersonnelId": checkpersonnelID,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	id := types.CheckpersonnelID(app.ReadNamedParam(r, "id"))

	var input struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	tr := types.TimeRange{
		Start: input.Start,
		End:   input.End,
	}
	if err := app.commands.Checkpersonnel.SetTimeRange(r.Context(), id, tr); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"updated": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) deleteCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	id := app.ReadNamedParam(r, "id")
	if err := app.commands.Checkpersonnel.Delete(r.Context(), types.CheckpersonnelID(id)); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
