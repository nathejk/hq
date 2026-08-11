package main

import (
	"net/http"

	"nathejk.dk/internal/requestctx"
	"nathejk.dk/nathejk/table/sos"
)

// actor resolves who is making the request, for domain commands that record it.
//
// This exists so the sos package does not have to: it is written to shared-go's
// guidelines and may not import nathejk.dk/... (PRD 001 §8), and the handler is
// the layer that knows about HTTP anyway. Every other local table package reaches
// into requestctx itself, which is exactly what makes those packages harder to
// move.
//
// Today the middleware puts an anonymous user with an empty id on every request —
// authentication is perimeter-only, basic auth on stage and production and nothing
// in dev, with a JWT service planned (PRD 001 §6 Auth). So this returns an empty
// actor in practice. It is wired anyway: when identity arrives, the events start
// carrying it with no change here or in the domain.
func (app *application) actor(r *http.Request) sos.Actor {
	u, ok := requestctx.UserFrom(r.Context())
	if !ok {
		return sos.Actor{}
	}
	return sos.Actor{UserID: u.ID, Name: u.Name}
}
