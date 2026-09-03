package main

import (
	"testing"

	"nathejk.dk/nathejk/table/track"
)

// buildTrack is where segmentation, reduction and coverage are composed, and the order they are
// composed in is a decision rather than an accident. That is what these tests pin — the three pieces
// have their own tests in table/track.

func trackPoints(base int64, stepMs int64, n int) []track.Point {
	out := make([]track.Point, n)
	for i := range out {
		out[i] = track.Point{
			Ts:  base + int64(i)*stepMs,
			Lat: 55.7 + float64(i)*0.0002,
			Lng: 12.6 + float64(i)*0.0003,
		}
	}
	return out
}

// The important one: coverage must describe how much was *recorded*, not how much survived the point
// budget. Measured after reduction, a well-recorded track would appear to thin out the moment an
// operator zoomed out — the number would then be reporting a property of the request rather than of
// the data.
func TestBuildTrackCoverageIsUnaffectedByReduction(t *testing.T) {
	points := trackPoints(1_000_000_000_000, 30_000, 2000)
	window := track.Window{From: 1_000_000_000_000, To: 1_000_000_000_000 + 2000*30_000}

	full := buildTrack("p-1", points, window, 100_000)
	thin := buildTrack("p-1", points, window, 50)

	if full.Coverage.RecordedMs != thin.Coverage.RecordedMs {
		t.Errorf("coverage changed with the point budget: %d vs %d",
			full.Coverage.RecordedMs, thin.Coverage.RecordedMs)
	}
	if full.Coverage.Points != thin.Coverage.Points {
		t.Errorf("coverage point count changed with the budget: %d vs %d",
			full.Coverage.Points, thin.Coverage.Points)
	}
	if !thin.Reduced {
		t.Error("want Reduced=true when the budget bites")
	}
	if full.Reduced {
		t.Error("want Reduced=false when the budget is generous")
	}
}

// A person who has never reported gets an empty track, not an error and not null segments.
func TestBuildTrackEmpty(t *testing.T) {
	got := buildTrack("p-1", nil, track.Window{From: 100, To: 200}, 0)

	if got.Segments == nil {
		t.Error("segments must serialise as [] rather than null")
	}
	if len(got.Segments) != 0 {
		t.Errorf("want no segments, got %d", len(got.Segments))
	}
	if got.Reduced {
		t.Error("nothing was reduced")
	}
	// The requested window is echoed even with no data: "nothing for this hour" differs from
	// "nothing at all".
	if got.Coverage.Window.From != 100 || got.Coverage.Window.To != 200 {
		t.Errorf("want the requested window echoed, got %+v", got.Coverage.Window)
	}
}

// Segments survive into the response as segments — the whole point of the shape.
func TestBuildTrackKeepsGaps(t *testing.T) {
	const base int64 = 1_000_000_000_000

	// 100 points at 30 s spans 99 intervals, so the first run ends at base+2_970_000. The second
	// starts a clear ten minutes after that — written out rather than eyeballed, because the first
	// version of this test put the second run *inside* the first and then asserted they were apart.
	const firstRunEnd = base + 99*30_000
	const secondRunStart = firstRunEnd + 10*60_000

	var points []track.Point
	points = append(points, trackPoints(base, 30_000, 100)...)
	points = append(points, trackPoints(secondRunStart, 30_000, 100)...)

	got := buildTrack("p-1", points, track.Window{}, 0)
	if len(got.Segments) != 2 {
		t.Fatalf("want 2 segments across the gap, got %d", len(got.Segments))
	}
	if got.Segments[0].To != firstRunEnd {
		t.Errorf("first segment should end at %d, got %d", firstRunEnd, got.Segments[0].To)
	}
	if got.Segments[1].From != secondRunStart {
		t.Errorf("second segment should start at %d, got %d", secondRunStart, got.Segments[1].From)
	}
}
