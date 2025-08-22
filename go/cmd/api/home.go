package main

import (
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/personnel"
)

func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	filter := patrulje.Filter{YearSlug: "2025"}
	teams, _ := app.models.Patrulje.GetAll(r.Context(), filter)
	teamCount := 0
	memberCount := 0
	for _, t := range teams {
		if t.PaidAmount == 0 {
			continue
		}
		teamCount++
		memberCount = memberCount + t.MemberCount
	}
	badutter, _ := app.models.Personnel.GetAll(r.Context(), personnel.Filter{YearSlug: "2025", UserTypes: []string{"gøgler"}})

	config := map[string]any{
		"timeCountdown": app.config.countdown.time,
		"videos":        app.config.countdown.videos,
		"patruljeCount": teamCount,
		"spejderCount":  memberCount,
		"badutCount":    len(badutter),
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
