package main

import (
	"net/http"
	"time"

	"nathejk.dk/internal/live"
)

// streamHandler serves the SPA's live-update stream.
//
// Thin on purpose: the streaming mechanics live in internal/live so they can be
// tested without a server, and this only supplies what the request context knows
// — namely what "current year" means, matching YearSlug()'s fallback for requests
// that do not say.
//
// Note the year arrives as a query parameter rather than the X-YearSlug header
// used everywhere else in the API: EventSource cannot set headers.
func (app *application) streamHandler() http.Handler {
	return live.StreamHandler{
		Hub: app.live,
		DefaultYear: func() string {
			return time.Now().Format("2006")
		},
		// Derived once at wiring time from the same consumers the hub is fed by,
		// so it cannot describe a stream different from the one being served.
		Entities: app.liveEntities,
	}
}
