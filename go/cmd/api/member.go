package main

import (
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
)

// showMemberCareHandler serves the count of members Nathejk is currently
// responsible for: the number that has to reach zero before the organisers can go
// home.
//
// Year-scoped from X-YearSlug like everything else. It is served as an event-wide
// figure rather than per case, because that is what it is — a member in our care is
// our problem whether or not anybody has opened a case about them.
//
// The response carries the oldest `waiting` timestamp rather than a "somebody has
// waited too long" boolean, so the threshold stays in one place and can change
// without a new deploy of this endpoint. It is still unsettled (PRD 006 §11, task
// 082).
func (app *application) showMemberCareHandler(w http.ResponseWriter, r *http.Request) {
	care, err := app.models.SpejderStatus.InOurCare(r.Context(), app.YearSlug(r))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"care": care}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
