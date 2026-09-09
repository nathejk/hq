package main

import (
	"net/http"
	"strconv"
	"strings"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/scan"
	"nathejk.dk/nathejk/table/track"
)

// The patrol position layer: one marker per patrulje, drawn from whatever last said where it was.
//
// # Why two sources rather than one
//
// HQ knows where a patrol is in two unrelated ways, and neither is sufficient alone. A scan is
// certain but rare — it happens at a post, when a human scans, so between posts it goes stale by
// hours. Telemetry is frequent but optional — it exists only for patrols whose members run the
// hej-app and left it running. During a race the two interleave: a team scans at 21:00, then walks
// for two hours reporting positions, then scans again. Preferring either source outright would
// produce a map that is confidently wrong, so the newer candidate wins per team.
//
// # Why one aggregate per source and not a query per team
//
// This endpoint is polled by a map layer, and there are ~200 patruljer with several members each.
// Per-team reads would be several hundred round trips per poll during the busiest hours of the
// event. Instead each source is scanned once for the whole year and the two are folded together in
// Go: three queries in total (identities, telemetry, scans), independent of how many teams exist.

// patruljePosition is one patrol's last known position, as the map layer consumes it.
//
// The field names are the wire contract and are not the internal ones: `latitude`/`longitude`
// spelled out (the SPA's map library takes them that way) and `ts` in epoch **milliseconds**,
// matching every other telemetry payload, so a marker and a track can share one time axis without
// the client knowing which source the marker came from.
type patruljePosition struct {
	TeamID     string  `json:"teamId"`
	TeamNumber string  `json:"teamNumber"`
	Name       string  `json:"name"`
	Lat        float64 `json:"latitude"`
	Lng        float64 `json:"longitude"`
	Ts         int64   `json:"ts"`

	// Source is "track" or "scan": which kind of evidence this marker rests on. The client needs
	// it, because the two mean different things — a scan is a witnessed fact at a known post, a
	// track point is a phone's best guess — and a marker that did not say which would let an
	// operator trust a stale scan as though it were live.
	Source string `json:"source"`
}

// Source values, exactly as the contract spells them.
const (
	positionSourceTrack = "track"
	positionSourceScan  = "scan"
)

// trackFix is a telemetry candidate: already numeric, already milliseconds.
type trackFix struct {
	Lat float64
	Lng float64
	Ts  int64
}

// scanFix is a scan candidate in the units and types the `scan` table actually stores them in.
//
// It exists in this unconverted form on purpose: the conversion is the trap this endpoint is most
// likely to be broken by, so it happens in one tested function rather than at whichever call site
// touched the data last. `Uts` is seconds; the coordinates are strings that may not be coordinates.
type scanFix struct {
	Lat string
	Lng string
	Uts int64
}

// positionFix is the merge's result: a position, or nothing.
type positionFix struct {
	Lat    float64
	Lng    float64
	Ts     int64
	Source string
}

// parseScanCoordinate turns a scan's VARCHAR latitude/longitude into numbers, reporting whether they
// were a position at all.
//
// `scan.latitude`/`longitude` are VARCHAR(99) — a known wart of that table — and a scanner that
// saved without a fix stores "" or "0". Those are the majority of unusable rows, and they must be
// *skipped*, never coerced: 0,0 is Null Island, off the coast of Africa, and a patrol drawn there
// looks like a patrol that has gone badly astray rather than like missing data.
//
// Zero is rejected on either axis. A real position in Denmark has neither a zero latitude nor a zero
// longitude, so there is no legitimate reading being thrown away, and half-parsed rows (one axis
// present, one blank) would otherwise plot on the equator or the prime meridian.
func parseScanCoordinate(lat, lng string) (float64, float64, bool) {
	latF, err := strconv.ParseFloat(strings.TrimSpace(lat), 64)
	if err != nil || latF == 0 {
		return 0, 0, false
	}
	lngF, err := strconv.ParseFloat(strings.TrimSpace(lng), 64)
	if err != nil || lngF == 0 {
		return 0, 0, false
	}
	return latF, lngF, true
}

// mergePosition picks a patrol's position from its two candidates.
//
// Either may be nil, and both being nil is the normal case for most of the year: it returns
// ok=false, and the caller omits the team entirely rather than emitting a marker at 0,0 or a row of
// nulls the client would have to filter.
//
// The two candidates are compared *after* the scan's seconds are multiplied to milliseconds. That
// multiplication is the whole reason this function exists: comparing `uts` against `ts` in their
// native units makes every scan look 54 years older than every track point, so telemetry would
// always win and the bug would look like "scans never show" rather than like a units error.
//
// An exact tie goes to the scan. Ties are only reachable at whole-second boundaries, and when the
// evidence is equally recent the witnessed position at a known post is the better one to draw.
func mergePosition(t *trackFix, s *scanFix) (positionFix, bool) {
	var candidates []positionFix

	if t != nil {
		candidates = append(candidates, positionFix{Lat: t.Lat, Lng: t.Lng, Ts: t.Ts, Source: positionSourceTrack})
	}
	if s != nil {
		if lat, lng, ok := parseScanCoordinate(s.Lat, s.Lng); ok {
			candidates = append(candidates, positionFix{Lat: lat, Lng: lng, Ts: s.Uts * 1000, Source: positionSourceScan})
		}
	}

	var best positionFix
	found := false
	for _, c := range candidates {
		// `>=` rather than `>`, with the scan appended last, is what gives a tie to the scan.
		if !found || c.Ts >= best.Ts {
			best, found = c, true
		}
	}
	return best, found
}

// listPatruljePositionsHandler serves the patrol position layer.
//
// @Summary     Last known position of every patrulje
// @Description One entry per patrulje that HQ knows a position for, newest evidence first-class: the position is whichever is more recent of the team's telemetry (a member's phone reporting through the hej-app) and its most recent QR scan, and `source` says which of the two this one is. `ts` is epoch **milliseconds** for both, so a marker shares its time axis with the track endpoints regardless of source. A patrulje with no known position is **omitted** from the array — that is "we do not know where they are", not an error and not a gap to retry; an empty array early in the season is the expected response. Scans whose stored coordinates are unusable (blank or zero) are skipped rather than reported as 0,0. Klaner and personnel are never included. Year comes from the X-YearSlug header, or the current year.
// @Tags        telemetry
// @Produce     json
// @Success     200 {object} map[string]interface{} "envelope with a \"positions\" array"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/telemetry/positions [get]
func (app *application) listPatruljePositionsHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)

	// Identities first, and they are also the filter: only teams in this map are emitted, which is
	// what keeps klaner and personnel out without either source needing to know about team types.
	identities, err := app.models.Patrulje.Identities(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	points, err := app.models.Track.PatruljeLatest(r.Context(), string(year))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	scans, err := app.models.Scan.TeamPositions(r.Context(), string(year))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	tracks := foldTrackFixes(points)
	scanFixes := foldScanFixes(scans)

	// Non-nil even when empty: `null` would make the map layer choose between a guard and a crash
	// for the entirely normal case of a year nobody has reported from yet.
	positions := []patruljePosition{}
	for _, id := range identities {
		fix, ok := mergePosition(tracks[string(id.TeamID)], scanFixes[string(id.TeamID)])
		if !ok {
			continue
		}
		positions = append(positions, patruljePosition{
			TeamID:     string(id.TeamID),
			TeamNumber: id.TeamNumber,
			Name:       id.Name,
			Lat:        fix.Lat,
			Lng:        fix.Lng,
			Ts:         fix.Ts,
			Source:     fix.Source,
		})
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"positions": positions}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// foldTrackFixes reduces one row per reporting member to one candidate per team.
//
// Both queries return their rows in ascending time order, so overwriting keeps the newest and this
// needs no comparison of its own. Cheaper than a comparison per row, and — more usefully — it means
// the ordering contract lives in exactly one place per query rather than being re-asserted here.
func foldTrackFixes(points []track.TeamPoint) map[string]*trackFix {
	out := make(map[string]*trackFix, len(points))
	for _, p := range points {
		out[p.TeamID] = &trackFix{Lat: p.Lat, Lng: p.Lng, Ts: p.Ts}
	}
	return out
}

// foldScanFixes reduces a year's scans to one candidate per team, discarding the ones that carry no
// position.
//
// Unusable coordinates are dropped *during* the fold, not after it. A team's newest scan is very
// often the one saved without a fix, and letting it into the map would shadow an older scan that
// does know where the team was — the marker would vanish rather than go stale, which is the more
// misleading of the two failures.
func foldScanFixes(scans []scan.TeamPosition) map[string]*scanFix {
	out := make(map[string]*scanFix, len(scans))
	for _, s := range scans {
		if _, _, ok := parseScanCoordinate(s.Latitude, s.Longitude); !ok {
			continue
		}
		out[string(s.TeamID)] = &scanFix{Lat: s.Latitude, Lng: s.Longitude, Uts: s.Uts}
	}
	return out
}
