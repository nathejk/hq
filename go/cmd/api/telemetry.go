package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/spejderstatus"
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

// patruljeTrackResponse is a patrol's whole movement.
//
// # Why the patrol, not the person
//
// For a spejder the unit that matters during a race is the patrol. Scouts move between patrols
// mid-race, phones die and get shared, and one member's track answers almost nothing on its own —
// "where has this patrol been?" is the question actually asked, and answering it means everyone who
// has been on the team, current and former, plus the fixed points we know they touched.
type patruljeTrackResponse struct {
	TeamID types.TeamID `json:"teamId"`

	// Members are the per-member tracks, each clipped to that member's membership interval.
	Members []patruljeMemberTrack `json:"members"`

	// Scans are the team's QR scans, exact and never reduced.
	//
	// They are the only *certain* positions on the map — a scan happened at a known post, at a known
	// time, witnessed by a person — whereas a track point is a phone's best guess. They are also few.
	// So they get their own list rather than being folded into the tracks, and nothing simplifies them.
	Scans []patruljeScan `json:"scans"`
}

type patruljeMemberTrack struct {
	trackResponse

	// MembershipFrom/To bound the stretch this member belonged to the patrol, in epoch milliseconds.
	// To is nil while they are still on the team. The legend shows this, so an operator can see that a
	// line stops because a scout left rather than because their phone did.
	MembershipFrom int64  `json:"membershipFrom,omitempty"`
	MembershipTo   *int64 `json:"membershipTo"`
}

// patruljeScan is one QR scan, with its time expressed the same way as a track point.
//
// This type exists for one reason, and it is the trap in this endpoint: `scan.uts` is **seconds**
// while `track_point.ts` is **milliseconds**. Handing both to a client in their native units would
// put the scans 54 years before the tracks on a shared time axis, and it would look like a data
// problem rather than a units problem. The conversion happens here, once, at the boundary that joins
// them — not in the SPA, and not per caller.
type patruljeScan struct {
	QrID       types.QrID   `json:"qrId"`
	TeamID     types.TeamID `json:"teamId"`
	TeamNumber int          `json:"teamNumber"`
	ScannerID  string       `json:"scannerId"`
	Ts         int64        `json:"ts"`
	Lat        string       `json:"lat"`
	Lng        string       `json:"lng"`
}

// showPatruljeTrackHandler serves a patrol's whole movement.
//
// @Summary     A patrol's track: every member, current and former, plus its scans
// @Description Where a patrol has been. Returns one track per person who has *ever* been on the team — including members who moved away and members whose spejder row was deleted when they withdrew, whom the client labels "tidligere medlem" — with each track clipped to the interval that member actually belonged to this patrol, so one patrol's map never shows another's movement. Tracks are split into segments wherever recording stopped for more than a few minutes, and simplified to at most maxPoints within segments, never across them; each carries coverage so an operator can tell a well-recorded track from a nearly empty one. The team's QR scans are returned separately and are never reduced: they are the only certain positions on the map, and their timestamps are converted from seconds to epoch milliseconds so tracks and scans share one time axis. A patrol with no telemetry still returns its scans.
// @Tags        telemetry
// @Produce     json
// @Param       teamId path string true "patrulje team id"
// @Param       from query int false "window start, epoch milliseconds"
// @Param       to query int false "window end, epoch milliseconds"
// @Param       maxPoints query int false "point budget per member track"
// @Success     200 {object} map[string]interface{} "envelope with a \"patrulje\" object"
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/telemetry/patrulje/{teamId}/track [get]
func (app *application) showPatruljeTrackHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "teamId"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}

	window, maxPoints := app.readTrackParams(r)
	year := app.YearSlug(r)

	// Membership first, because it decides which tracks to read at all — and it comes from the
	// lifecycle log rather than the `spejder` table, which hard-deletes withdrawn scouts and would
	// therefore omit exactly the people whose movement is being reconstructed.
	memberships, err := app.models.SpejderStatus.TeamMemberships(r.Context(), year, teamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	response := patruljeTrackResponse{
		TeamID:  teamID,
		Members: []patruljeMemberTrack{},
		Scans:   []patruljeScan{},
	}

	for _, m := range memberships {
		// The window actually queried is the requested window intersected with the membership, so a
		// member who left at 11:00 contributes nothing after 11:00 even though their phone kept
		// reporting for another patrol.
		clipped := clipWindow(window, m)

		points, err := app.models.Track.Points(r.Context(), track.Filter{
			PersonID: string(m.MemberID),
			FromTs:   clipped.From,
			ToTs:     clipped.To,
		})
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return
		}

		// Members with no positions are included anyway. "This scout's phone reported nothing" is
		// something the legend should say; dropping them would leave an operator unsure whether the
		// member was absent from the patrol or merely from the data.
		mt := patruljeMemberTrack{
			trackResponse:  buildTrack(string(m.MemberID), points, clipped, maxPoints),
			MembershipFrom: msOf(m.From),
		}
		mt.Name = m.Name
		if m.To != nil {
			to := msOf(*m.To)
			mt.MembershipTo = &to
		}
		response.Members = append(response.Members, mt)
	}

	scans, _, err := app.models.Scan.GetAll(r.Context(), scan.Filter{TeamID: teamID})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	for _, s := range scans {
		if s == nil {
			continue
		}
		ts := int64(s.Uts) * 1000
		if window.From > 0 && ts < window.From {
			continue
		}
		if window.To > 0 && ts > window.To {
			continue
		}
		response.Scans = append(response.Scans, patruljeScan{
			QrID:       s.QrID,
			TeamID:     s.TeamID,
			TeamNumber: s.TeamNumber,
			ScannerID:  s.ScannerID,
			Ts:         ts,
			Lat:        s.Latitude,
			Lng:        s.Longitude,
		})
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"patrulje": response}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// clipWindow narrows a requested window to a membership interval.
//
// This is what stops one patrol's map showing another's movement. A zero membership bound means
// "unknown", not "epoch" — a member whose patrol never started has no event to date their start — so
// an unknown bound leaves the requested window's bound in place rather than clipping to 1970.
func clipWindow(requested track.Window, m spejderstatus.Membership) track.Window {
	out := requested

	if from := msOf(m.From); from > out.From {
		out.From = from
	}
	if m.To != nil {
		if to := msOf(*m.To); out.To == 0 || to < out.To {
			out.To = to
		}
	}
	return out
}

// msOf converts a time to epoch milliseconds, mapping the zero time to 0 rather than to the year 1 in
// milliseconds — a large negative number, which as a window bound would clip nothing and as a
// membership start would look like 1970.
func msOf(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
