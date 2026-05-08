package main

import (
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/payment"
)

func (app *application) listPaymentsHandler(w http.ResponseWriter, r *http.Request) {
	filter := payment.Filter{Year: app.YearSlug(r)}
	payments, err := app.models.Payment.GetAll(r.Context(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"payments": payments}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
