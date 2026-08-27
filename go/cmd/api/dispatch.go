package main

import (
	"net/http"

	"nathejk.dk/nathejk/table/dispatch"
)

// dispatchActor is who the dispatch desk records as having acted.
//
// The fourth of these little conversions (sos, spejderstatus, spejdernote, dispatch), and
// deliberately still not a shared types.Actor: every one of these packages is written to be
// liftable to shared-go independently and none may import another, so a shared struct is a
// cross-repo change, not a local tidy-up. Worth doing when somebody is in shared-go anyway.
//
// Empty in practice until HQ has login. That matters less here than elsewhere, because the
// fact the desk actually needs — *which unit* took the job — is an explicit choice on the
// tour, not something inferred from who is typing (PRD 009 §8).
func (app *application) dispatchActor(r *http.Request) dispatch.Actor {
	user := app.actor(r)
	return dispatch.Actor{UserID: user.UserID, Name: user.Name}
}
