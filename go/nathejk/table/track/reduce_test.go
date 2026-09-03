package track

import (
	"math"
	"testing"
)

// Reduction is tested for the two properties that matter to a reader of the map: the payload gets
// smaller, and the route still looks like the route. Everything else is an implementation detail.

// wanderingPts builds a path that actually goes somewhere, with the small deviations a real GPS trail
// has.
//
// Needed because `pts` (segment_test.go) repeats one coordinate, which is a *stationary* track — and
// Douglas–Peucker correctly collapses that to its endpoints however large the budget, since there is
// no shape to preserve. Same for a perfectly straight line. That is the algorithm working, but it
// makes such data useless for testing that a budget is *used*.
func wanderingPts(base int64, stepMs int64, n int) []Point {
	out := make([]Point, n)
	for i := range out {
		f := float64(i)
		out[i] = Point{
			Ts: base + int64(i)*stepMs,
			// A gentle arc with a fine wobble on top, so no three consecutive points are collinear.
			Lat: 55.7 + f*0.0002 + math.Sin(f/3)*0.0004,
			Lng: 12.6 + f*0.0003 + math.Cos(f/2)*0.0005,
		}
	}
	return out
}

func TestReduceLeavesSmallTracksAlone(t *testing.T) {
	segments := Segments(wanderingPts(1_000_000, 30_000, 50))

	out, reduced := Reduce(segments, 2000)
	if reduced {
		t.Error("a track well inside the budget must not be reported as reduced")
	}
	if len(out[0].Points) != 50 {
		t.Errorf("want all 50 points, got %d", len(out[0].Points))
	}
}

func TestReduceHonoursTheBudget(t *testing.T) {
	// A worst-realistic-case patrol member: 30 hours at 30 s, continuous.
	segments := Segments(wanderingPts(1_000_000, 30_000, 3600))

	out, reduced := Reduce(segments, 500)
	if !reduced {
		t.Error("want reduced=true")
	}

	total := 0
	for _, s := range out {
		total += len(s.Points)
	}
	if total > 500 {
		t.Errorf("want at most 500 points, got %d", total)
	}
	// Not merely "at most": a budget that returns 3 points when 500 were allowed is technically
	// within bounds and useless. Bisection should land near the ceiling.
	if total < 400 {
		t.Errorf("want the budget roughly used (>=400), got %d", total)
	}
}

// The guarantee task 145 exists to provide: reduction must not be able to bridge a gap. Segment count
// and every segment's declared interval survive untouched.
func TestReduceNeverMergesSegments(t *testing.T) {
	var points []Point
	points = append(points, pts(0, 30_000, 400)...)
	points = append(points, pts(50_000_000, 30_000, 400)...)
	points = append(points, pts(90_000_000, 30_000, 400)...)
	segments := Segments(points)

	out, _ := Reduce(segments, 30)

	if len(out) != len(segments) {
		t.Fatalf("segment count must survive reduction: want %d, got %d", len(segments), len(out))
	}
	for i := range out {
		if out[i].From != segments[i].From || out[i].To != segments[i].To {
			t.Errorf("segment %d interval changed: %d..%d became %d..%d",
				i, segments[i].From, segments[i].To, out[i].From, out[i].To)
		}
	}
}

// A segment reduced out of existence would be a gap that never happened, so every segment keeps at
// least its endpoints even when the budget is absurdly tight.
func TestReduceKeepsEndpointsOfEverySegment(t *testing.T) {
	var points []Point
	for i := 0; i < 10; i++ {
		points = append(points, pts(int64(i)*50_000_000, 30_000, 100)...)
	}
	segments := Segments(points)

	out, _ := Reduce(segments, 1) // less than two points per segment is impossible to honour

	if len(out) != 10 {
		t.Fatalf("want 10 segments, got %d", len(out))
	}
	for i, s := range out {
		if len(s.Points) < 2 {
			t.Errorf("segment %d has %d points; endpoints are not negotiable", i, len(s.Points))
		}
		if s.Points[0].Ts != s.From || s.Points[len(s.Points)-1].Ts != s.To {
			t.Errorf("segment %d must keep its first and last point", i)
		}
	}
}

// The reason for Douglas–Peucker rather than every nth point: a corner is what makes a route
// recognisable, and it must survive when the straight stretches around it do not.
func TestReducePreservesCorners(t *testing.T) {
	// A right angle: 100 points due east, then 100 points due north.
	var points []Point
	var ts int64
	for i := 0; i < 100; i++ {
		points = append(points, Point{Ts: ts, Lat: 55.0, Lng: 12.0 + float64(i)*0.001})
		ts += 30_000
	}
	corner := points[len(points)-1]
	for i := 1; i <= 100; i++ {
		points = append(points, Point{Ts: ts, Lat: 55.0 + float64(i)*0.001, Lng: corner.Lng})
		ts += 30_000
	}
	segments := Segments(points)

	out, reduced := Reduce(segments, 10)
	if !reduced {
		t.Fatal("want reduced=true")
	}

	// The turn must be in there. A nth-point reduction would keep whichever points happened to land
	// on its stride and would miss this by up to a stride's width.
	var foundCorner bool
	for _, p := range out[0].Points {
		if p.Ts == corner.Ts {
			foundCorner = true
			break
		}
	}
	if !foundCorner {
		t.Errorf("the corner at ts=%d was dropped; got %d points: %+v", corner.Ts, len(out[0].Points), out[0].Points)
	}
}

// A stationary phone reports the same fix repeatedly. The endpoints of such a run coincide, which is
// a degenerate line — it must not divide by zero or panic.
func TestReduceHandlesStationaryRun(t *testing.T) {
	points := make([]Point, 500)
	var ts int64
	for i := range points {
		points[i] = Point{Ts: ts, Lat: 55.5, Lng: 12.5}
		ts += 30_000
	}

	out, reduced := Reduce(Segments(points), 20)
	if !reduced {
		t.Error("want reduced=true")
	}
	// Nothing is worth keeping in the middle of a run that never moved, so the result should collapse
	// to roughly its endpoints rather than spending the budget on identical points.
	if len(out[0].Points) > 20 {
		t.Errorf("want at most 20 points, got %d", len(out[0].Points))
	}
}

// A straight road, or a phone that never moved, has no shape to preserve — so it collapses to its
// endpoints no matter how generous the budget. That is the algorithm being right, and it is worth a
// test so nobody later "fixes" it into spending the budget on redundant points.
func TestReduceCollapsesAStraightLine(t *testing.T) {
	points := make([]Point, 1000)
	var ts int64
	for i := range points {
		points[i] = Point{Ts: ts, Lat: 55.0, Lng: 12.0 + float64(i)*0.001}
		ts += 30_000
	}

	out, reduced := Reduce(Segments(points), 500)
	if !reduced {
		t.Fatal("want reduced=true")
	}
	if len(out[0].Points) != 2 {
		t.Errorf("a straight line needs only its endpoints, got %d points", len(out[0].Points))
	}
}

func TestReduceEmpty(t *testing.T) {
	out, reduced := Reduce(Segments(nil), 100)
	if reduced {
		t.Error("nothing to reduce")
	}
	if len(out) != 0 {
		t.Errorf("want 0 segments, got %d", len(out))
	}
}

// A zero or negative budget is a caller that did not decide, not a caller asking for nothing.
func TestReduceZeroBudgetFallsBackToDefault(t *testing.T) {
	segments := Segments(wanderingPts(1_000_000, 30_000, 5000))

	out, reduced := Reduce(segments, 0)
	if !reduced {
		t.Fatal("want reduced=true")
	}
	total := 0
	for _, s := range out {
		total += len(s.Points)
	}
	if total > DefaultMaxPoints {
		t.Errorf("want at most DefaultMaxPoints (%d), got %d", DefaultMaxPoints, total)
	}
}

// Points must stay in time order after reduction — the client draws them in the order given.
func TestReduceKeepsTimeOrder(t *testing.T) {
	segments := Segments(wanderingPts(1_000_000, 30_000, 2000))
	out, _ := Reduce(segments, 100)

	for _, s := range out {
		for i := 1; i < len(s.Points); i++ {
			if s.Points[i].Ts <= s.Points[i-1].Ts {
				t.Fatalf("points out of order at %d: %d then %d", i, s.Points[i-1].Ts, s.Points[i].Ts)
			}
		}
	}
}
