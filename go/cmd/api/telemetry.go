package main

import (
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
)

// Telemetry: where people were (PRD 011).
//
// # Why presence is one endpoint rather than a field on ten
//
// HQ shows a small glyph next to a person's name wherever people are listed — badutter, personnel,
// crew, patrol members, klan members, SOS cards, the shelter and care lists — saying whether that
// person's phone reports positions at all, and when it last did. The obvious implementation is to
// add `hasPosition`/`lastSeenAt` to every one of those payloads, and it is the wrong one: ten
// handlers and ten queries would grow a join each, for one glyph.
//
// Instead one small response says which people have ever reported, keyed by id, and the SPA looks
// its own rows up in it. That works because of a fact about identity rather than a trick: a
// `personId` is either a memberID (spejder, senior) or a crewmemberID — the same id space
// `personnel` uses for gøgler, friend and bandit — both opaque and non-colliding, and every
// people-list row in HQ already carries the id it needs. So no endpoint changes shape, no lookup
// table exists, and a new list of people gets the glyph for free.
//
// # What it deliberately does not say
//
// Nothing about *where* anyone is. That is the whole point: this endpoint is fetched on nearly every
// page, so it carries one timestamp per person and no coordinates. Positions are read only when
// somebody asks for a route (tasks 147, 149), which is both cheaper and a smaller disclosure of
// personal data on pages that have no business showing it.
//
// # A note on access control
//
// This route sits behind the same `app.authenticate` middleware as every other `/api/` route, which
// is all this repo can offer: authentication lives in an external service and the middleware here
// currently attributes every request to an anonymous user (see routes.go). So "authenticated" means
// "as protected as member contact details already are", not "verified here". PRD 011 asserts that
// telemetry is restricted to authenticated HQ users; that promise is kept by whatever fronts this
// api, not by this file, and it is worth knowing which of the two is true before treating position
// history as more protected than the rest of the read model.

// presenceEntry is one person's telemetry presence.
//
// Ts is epoch milliseconds, as sent by the producer and stored — the same integer the track
// endpoints take as bounds, so the SPA never converts between representations of an instant.
type presenceEntry struct {
	Ts int64 `json:"ts"`

	// PersonType is the role held when that last position was reported, not the role held now.
	// Advisory here: the glyph does not branch on it, but it makes a presence response readable on
	// its own and lets a future staleness rule differ per population without a new field.
	PersonType string `json:"personType"`
}

// listTelemetryPresenceHandler serves the presence map.
//
// @Summary     Who has ever reported a position
// @Description Every personId that has reported at least one position this year, with the epoch-milliseconds timestamp of the most recent one. Keyed by personId, which is either a memberID (spejder, senior) or a crewmemberID (crew, and the same id space as personnel: gøgler, friend, bandit) — the two do not collide, so no type hint is needed to look one up. Carries no coordinates by design: this is fetched on nearly every page, and where someone is belongs to the track endpoints. An absent id means "has never reported", which is a meaningful state rather than missing data. Year comes from the X-YearSlug header, or the current year.
// @Tags        telemetry
// @Produce     json
// @Success     200 {object} map[string]interface{} "envelope with a \"presence\" object keyed by personId"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/telemetry/presence [get]
func (app *application) listTelemetryPresenceHandler(w http.ResponseWriter, r *http.Request) {
	latest, err := app.models.Track.Presence(r.Context(), string(app.YearSlug(r)))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// An object rather than an array: the only question asked of this payload is "is this id in
	// here, and when?", which a map answers without the client building an index of its own on
	// every render.
	//
	// Non-nil even when empty, so the client can index into it unconditionally. `null` here would
	// make an early-season page choose between a guard and a crash for no reason.
	presence := make(map[string]presenceEntry, len(latest))
	for _, l := range latest {
		presence[l.PersonID] = presenceEntry{Ts: l.Ts, PersonType: l.PersonType}
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"presence": presence}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
