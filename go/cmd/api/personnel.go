package main

import (
	"context"
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/personnel"
)

func (app *application) listPersonnelHandler(w http.ResponseWriter, r *http.Request) {
	filter := personnel.Filter{YearSlug: app.YearSlug(r), UserTypes: []string{"friend"}}
	list, err := app.models.Personnel.GetAll(context.Background(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"personnel": list}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
