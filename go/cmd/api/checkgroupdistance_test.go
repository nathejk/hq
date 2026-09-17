package main

import (
	"math"
	"testing"
)

// Real coordinates from the race area, so a broken formula shows up as a wrong number of
// kilometres rather than as a plausible-looking abstraction.
var (
	// Two points about 2.1 km apart in northern Zealand.
	postA = postLocation{Name: "Post 2A", Lat: 55.9400, Lng: 12.3000}
	postB = postLocation{Name: "Post 2B", Lat: 55.9000, Lng: 12.3600}
)

func TestHaversineMetersKnownDistance(t *testing.T) {
	// One degree of latitude is ~111.2 km; a tenth of one is ~11.1 km.
	got := haversineMeters(55.9, 12.3, 56.0, 12.3)
	if math.Abs(got-11119) > 60 {
		t.Errorf("distance = %.0f m, want ~11119 m (0.1 degree of latitude)", got)
	}
}

func TestHaversineMetersIsZeroForTheSamePoint(t *testing.T) {
	if got := haversineMeters(55.9, 12.3, 55.9, 12.3); got != 0 {
		t.Errorf("distance = %v, want 0", got)
	}
}

func TestHaversineMetersIsSymmetric(t *testing.T) {
	there := haversineMeters(55.9, 12.3, 55.94, 12.36)
	back := haversineMeters(55.94, 12.36, 55.9, 12.3)
	if math.Abs(there-back) > 0.001 {
		t.Errorf("%.3f one way, %.3f the other", there, back)
	}
}

func TestNearestPostPicksTheClosestOfTheLine(t *testing.T) {
	// Sitting almost on top of post B.
	post, meters, ok := nearestPost(55.9005, 12.3595, []postLocation{postA, postB})
	if !ok {
		t.Fatal("a line with two located posts must have a nearest one")
	}
	if post.Name != "Post 2B" {
		t.Errorf("nearest = %q, want Post 2B", post.Name)
	}
	if meters > 100 {
		t.Errorf("distance = %.0f m, want under 100 m", meters)
	}
}

func TestNearestPostWithNoLocatedPosts(t *testing.T) {
	// Position is optional on a checkpoint, so a whole line can be unlocated. There is then no
	// distance to state, and stating one to 0,0 would put every patrol 900 km out.
	if _, _, ok := nearestPost(55.9, 12.3, nil); ok {
		t.Error("no posts means no nearest post")
	}
}

func TestDistanceToLineCarriesTheAgeAndSource(t *testing.T) {
	fix := positionFix{Lat: 55.9005, Lng: 12.3595, Ts: 1_700_000_000_000, Source: positionSourceTrack}

	d := distanceToLine(fix, true, []postLocation{postA, postB})

	if d == nil {
		t.Fatal("a known position and a located line must produce a distance")
	}
	if d.CheckpointName != "Post 2B" {
		t.Errorf("measured to %q, want Post 2B", d.CheckpointName)
	}
	if d.Ts != fix.Ts {
		t.Errorf("ts = %d, want the position's own timestamp: a distance without its age misleads", d.Ts)
	}
	if d.Source != positionSourceTrack {
		t.Errorf("source = %q, want %q", d.Source, positionSourceTrack)
	}
}

func TestDistanceToLineWithoutAKnownPosition(t *testing.T) {
	// The common case: most patruljer report no telemetry, and a scan without coordinates is not
	// a position. Nil means "HQ does not know", which the column shows as a dash.
	if d := distanceToLine(positionFix{}, false, []postLocation{postA}); d != nil {
		t.Errorf("distance = %+v, want none", d)
	}
}

func TestDistanceToLineWithAnUnlocatedLine(t *testing.T) {
	fix := positionFix{Lat: 55.9, Lng: 12.3, Ts: 1, Source: positionSourceScan}
	if d := distanceToLine(fix, true, nil); d != nil {
		t.Errorf("distance = %+v, want none: there is nothing to measure to", d)
	}
}
