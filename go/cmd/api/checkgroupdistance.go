package main

import (
	"context"
	"math"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/checkpoint"
)

// How far away a patrol that has not arrived is, and how old that knowledge is.
//
// # Why the missing list needs a distance at all
//
// "Mangler" answers *whether* a patrol is out there but nothing about what to do next, and the two
// cases behind one word could not be more different: a team half a kilometre from the post is
// walking in and needs nothing, while a team eleven kilometres out with the line closing in twenty
// minutes is a decision. Same status, same red tag, opposite response — so the distance is what
// turns the list into something an operator can triage rather than merely read.
//
// # Why the age is not optional
//
// A distance without a timestamp is worse than no distance: HQ's knowledge of where a patrol is
// goes stale by hours between posts, and "2,1 km" read as current when it was true at midnight
// sends somebody to the wrong place. The two are therefore one value, never separate fields the
// client could render independently.

// postLocation is one post of a line, as a target to measure to.
type postLocation struct {
	Name string
	Lat  float64
	Lng  float64
}

// teamDistance is a patrol's last known position expressed as a distance to this line.
//
// Ts is epoch **milliseconds**, matching every other telemetry payload in this API, so an age can
// be computed against the same clock the map uses.
type teamDistance struct {
	Meters float64 `json:"meters"`
	// CheckpointName is the post measured to: the nearest one on the line, which on a line whose
	// posts are kilometres apart is the only distance that means anything.
	CheckpointName string `json:"checkpointName"`
	Ts             int64  `json:"ts"`
	// Source is "track" or "scan", the same vocabulary as the map layer: a scan is a witnessed
	// fact at a post, a track point a phone's best guess, and an operator reading a distance is
	// entitled to know which.
	Source string `json:"source"`
}

// earthRadiusMeters is the mean radius, which is what the haversine formula assumes.
const earthRadiusMeters = 6371000.0

// haversineMeters is the great-circle distance between two positions.
//
// Straight-line, deliberately: a patrol's real remaining walk follows roads and paths this service
// knows nothing about, so the honest number is the one that is obviously a lower bound. Presenting
// a routed distance would imply knowledge of the route the team will take, which nobody has.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// nearestPost finds the closest post of the line, reporting whether there was one to find.
//
// ok=false when the line has no post with coordinates — a real state, since position is optional on
// a checkpoint, and one that must produce no distance rather than a distance to nowhere.
func nearestPost(lat, lng float64, posts []postLocation) (postLocation, float64, bool) {
	var best postLocation
	bestMeters := math.Inf(1)
	found := false
	for _, p := range posts {
		if d := haversineMeters(lat, lng, p.Lat, p.Lng); d < bestMeters {
			best, bestMeters, found = p, d, true
		}
	}
	return best, bestMeters, found
}

// distanceToLine turns a last-known position into a row's distance, or nothing.
//
// Nothing is the common case and not a failure: most patruljer never report telemetry, and one
// whose scans carry no coordinates has no position at all. The column reads "—" and the operator
// learns that HQ does not know, which is true and useful; a zero or a guess would not be.
func distanceToLine(fix positionFix, ok bool, posts []postLocation) *teamDistance {
	if !ok {
		return nil
	}
	post, meters, found := nearestPost(fix.Lat, fix.Lng, posts)
	if !found {
		return nil
	}
	return &teamDistance{
		Meters:         meters,
		CheckpointName: post.Name,
		Ts:             fix.Ts,
		Source:         fix.Source,
	}
}

// linePosts reads the coordinates of a line's posts.
//
// Posts without a position are skipped rather than defaulted: 0,0 is off the coast of Africa, and a
// patrol reported as 900 km from a post is a worse answer than no answer. Same reasoning as
// parseScanCoordinate, for the same reason.
func (app *application) linePosts(ctx context.Context, cgID types.CheckgroupID) ([]postLocation, error) {
	cps, err := app.models.Checkpoint.GetAll(ctx, checkpoint.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	if err != nil {
		return nil, err
	}
	posts := make([]postLocation, 0, len(cps))
	for _, cp := range cps {
		if cp.Latitude == nil || cp.Longitude == nil || *cp.Latitude == 0 || *cp.Longitude == 0 {
			continue
		}
		posts = append(posts, postLocation{
			Name: cp.Name,
			Lat:  float64(*cp.Latitude),
			Lng:  float64(*cp.Longitude),
		})
	}
	return posts, nil
}

// lastKnownPositions is every patrol's last known position this year, from either source.
//
// Two year-wide aggregates folded in Go, rather than a query per team, and the same merge the map
// layer uses (mergePosition) so a distance here and a marker there cannot disagree about where a
// team is. Skipped entirely when the line has no located post, since nothing would be done with
// the answer.
func (app *application) lastKnownPositions(ctx context.Context, year types.YearSlug) (map[string]positionFix, error) {
	points, err := app.models.Track.PatruljeLatest(ctx, string(year))
	if err != nil {
		return nil, err
	}
	scans, err := app.models.Scan.TeamPositions(ctx, string(year))
	if err != nil {
		return nil, err
	}

	tracks := foldTrackFixes(points)
	scanFixes := foldScanFixes(scans)

	out := make(map[string]positionFix, len(tracks)+len(scanFixes))
	for teamID := range tracks {
		if fix, ok := mergePosition(tracks[teamID], scanFixes[teamID]); ok {
			out[teamID] = fix
		}
	}
	for teamID := range scanFixes {
		if _, done := out[teamID]; done {
			continue
		}
		if fix, ok := mergePosition(tracks[teamID], scanFixes[teamID]); ok {
			out[teamID] = fix
		}
	}
	return out, nil
}
