package main

import (
	"net/http"
	"strconv"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/track"
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

// trackResponse is one person's track, as both track endpoints report it.
//
// Shaped around segments rather than a flat point array, which is the load-bearing decision of PRD
// 011: if the API returns segments, a client *cannot* draw a solid line across three hours of silence
// and present it as a walked route. Making the misrendering structurally impossible is worth more
// than documenting that it would be wrong.
type trackResponse struct {
	PersonID   string `json:"personId"`
	PersonType string `json:"personType"`

	// Name where one is known, for the patrol endpoint's legend. Empty for a person hq has no row
	// for — which is normal rather than an error, and never a reason to withhold the track.
	Name string `json:"name,omitempty"`

	Coverage track.Coverage  `json:"coverage"`
	Segments []track.Segment `json:"segments"`

	// Reduced says whether points were dropped to fit the budget, so the UI can offer full fidelity
	// for a narrower window instead of quietly implying this is everything.
	Reduced   bool `json:"reduced"`
	MaxPoints int  `json:"maxPoints"`
}

// buildTrack turns raw points into the response shape: segment, then reduce within segments, then
// measure coverage.
//
// The order matters and is not interchangeable. Coverage is computed from the **unreduced** segments
// because it describes how much was recorded, not how much survived the budget — measuring after
// reduction would make a well-recorded track look thin the moment somebody zoomed out.
func buildTrack(personID string, points []track.Point, w track.Window, maxPoints int) trackResponse {
	segments := track.Segments(points)
	coverage := track.CoverageOf(segments, w)
	reduced, wasReduced := track.Reduce(segments, maxPoints)

	return trackResponse{
		PersonID:  personID,
		Coverage:  coverage,
		Segments:  reduced,
		Reduced:   wasReduced,
		MaxPoints: maxPoints,
	}
}

// readTrackParams reads the time window and point budget from the query string.
//
// Bounds are epoch **milliseconds**, matching the stored `ts` exactly — no date parsing and no
// timezone to misread, and the same integers the SPA already holds from the presence endpoint.
//
// Unparseable values are treated as absent rather than rejected with a 400. These are a view's
// framing of a picture, not a command: a garbled `maxPoints` should show the operator a sensible
// default track, not an error page in the middle of an incident.
func (app *application) readTrackParams(r *http.Request) (track.Window, int) {
	qs := r.URL.Query()

	parse := func(key string) int64 {
		v, err := strconv.ParseInt(app.ReadString(qs, key, ""), 10, 64)
		if err != nil {
			return 0
		}
		return v
	}

	window := track.Window{From: parse("from"), To: parse("to")}

	maxPoints := int(parse("maxPoints"))
	if maxPoints <= 0 {
		maxPoints = track.DefaultMaxPoints
	}
	// An unbounded request must not be able to ask for the raw ceiling: a caller that names a
	// ridiculous budget gets the default, not megabytes.
	if maxPoints > track.DefaultMaxPoints {
		maxPoints = track.DefaultMaxPoints
	}

	return window, maxPoints
}

// showPersonTrackHandler serves one person's track.
//
// @Summary     One person's track
// @Description Where one person has been, as ordered segments rather than a flat list of points: a track is split wherever recording stopped for more than a few minutes, so a client cannot draw a line across a gap and imply a walk that never happened. Gaps of hours are normal — phones lock, apps are killed, batteries die — so the response also reports coverage, letting an operator tell a well-recorded track from a nearly empty one before reasoning from it. personId is either a memberID (spejder, senior) or a crewmemberID (crew, gøgler, friend, bandit); the two id spaces do not collide, so no type hint is needed. from/to are epoch milliseconds, matching the stored timestamps exactly. Points are simplified server-side to at most maxPoints, always within segments and never across them, and `reduced` says whether anything was dropped — ask for a narrower window to see more detail. A person who has never reported returns an empty track, not a 404.
// @Tags        telemetry
// @Produce     json
// @Param       personId path string true "member id or crewmember id"
// @Param       from query int false "window start, epoch milliseconds"
// @Param       to query int false "window end, epoch milliseconds"
// @Param       maxPoints query int false "point budget for the whole track"
// @Success     200 {object} map[string]interface{} "envelope with a \"track\" object"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/telemetry/person/{personId}/track [get]
func (app *application) showPersonTrackHandler(w http.ResponseWriter, r *http.Request) {
	personID := app.ReadNamedParam(r, "personId")
	if personID == "" {
		app.NotFoundResponse(w, r)
		return
	}

	window, maxPoints := app.readTrackParams(r)

	points, err := app.models.Track.Points(r.Context(), track.Filter{
		PersonID: personID,
		FromTs:   window.From,
		ToTs:     window.To,
	})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// A person who has never reported gets an empty track rather than a 404. "We have no positions
	// for this scout" is an answer, and the caller — a dialog opened from a name in a list — has
	// already established that the person exists.
	response := buildTrack(personID, points, window, maxPoints)
	if latest, err := app.models.Track.LatestFor(r.Context(), personID); err == nil && latest != nil {
		response.PersonType = latest.PersonType
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"track": response}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
