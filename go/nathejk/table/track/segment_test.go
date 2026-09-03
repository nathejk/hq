package track

import "testing"

// Segmenting is the decision that makes an honest map possible, so it is tested as a pure function —
// no database, no request, no broker. Same reason the producer keeps `track.Clean` out of its HTTP
// handler.

// pts builds points a fixed interval apart, starting at base. Timestamps are epoch ms.
func pts(base int64, stepMs int64, n int) []Point {
	out := make([]Point, n)
	for i := range out {
		out[i] = Point{Ts: base + int64(i)*stepMs, Lat: 55.7, Lng: 12.6}
	}
	return out
}

func TestSegmentsEmpty(t *testing.T) {
	got := Segments(nil)
	if got == nil {
		t.Fatal("Segments(nil) must return an empty slice, not nil — it is serialised straight to JSON")
	}
	if len(got) != 0 {
		t.Errorf("want 0 segments, got %d", len(got))
	}
}

func TestSegmentsSinglePoint(t *testing.T) {
	got := Segments([]Point{{Ts: 1000, Lat: 55.7, Lng: 12.6}})
	if len(got) != 1 {
		t.Fatalf("want 1 segment, got %d", len(got))
	}
	// A lone point is a zero-length segment: it evidences an instant, and From == To says so.
	if got[0].From != 1000 || got[0].To != 1000 {
		t.Errorf("want From==To==1000, got %d..%d", got[0].From, got[0].To)
	}
}

func TestSegmentsContinuousRunIsOneSegment(t *testing.T) {
	// 30 s sampling for an hour: 120 points, no gap anywhere near the threshold.
	points := pts(1_000_000, 30_000, 120)
	got := Segments(points)
	if len(got) != 1 {
		t.Fatalf("want 1 segment for a continuous run, got %d", len(got))
	}
	if len(got[0].Points) != 120 {
		t.Errorf("want all 120 points in the segment, got %d", len(got[0].Points))
	}
	if got[0].From != points[0].Ts || got[0].To != points[119].Ts {
		t.Errorf("segment bounds wrong: %d..%d", got[0].From, got[0].To)
	}
}

func TestSegmentsSplitsOnGaps(t *testing.T) {
	// Three stretches separated by an hour each — the ordinary shape of a real 30-hour track.
	var points []Point
	points = append(points, pts(0, 30_000, 10)...)         // 0 .. 270_000
	points = append(points, pts(4_000_000, 30_000, 5)...)  // one hour-ish later
	points = append(points, pts(9_000_000, 30_000, 20)...) // and again
	got := Segments(points)

	if len(got) != 3 {
		t.Fatalf("want 3 segments, got %d", len(got))
	}
	for i, want := range []int{10, 5, 20} {
		if len(got[i].Points) != want {
			t.Errorf("segment %d: want %d points, got %d", i, want, len(got[i].Points))
		}
	}
}

// The boundary has to fall somewhere and the const's wording decides where: a silence *longer than*
// the threshold breaks the track, so exactly at the threshold does not.
func TestSegmentsGapExactlyAtThresholdDoesNotSplit(t *testing.T) {
	points := []Point{{Ts: 0}, {Ts: GapThresholdMs}}
	if got := Segments(points); len(got) != 1 {
		t.Errorf("a gap exactly at the threshold must not split: got %d segments", len(got))
	}

	points = []Point{{Ts: 0}, {Ts: GapThresholdMs + 1}}
	if got := Segments(points); len(got) != 2 {
		t.Errorf("a gap one ms over the threshold must split: got %d segments", len(got))
	}
}

// Every point must survive segmentation. Splitting is grouping, not filtering — dropping data here
// would be indistinguishable from the phone never having recorded it.
func TestSegmentsKeepEveryPoint(t *testing.T) {
	var points []Point
	points = append(points, pts(0, 30_000, 7)...)
	points = append(points, pts(10_000_000, 30_000, 3)...)
	points = append(points, pts(20_000_000, 30_000, 11)...)

	var total int
	for _, s := range Segments(points) {
		total += len(s.Points)
	}
	if total != len(points) {
		t.Errorf("want %d points across segments, got %d", len(points), total)
	}
}

func TestCoverageOfWindow(t *testing.T) {
	// Two ten-minute stretches inside a one-hour window: 20 of 60 minutes recorded.
	var points []Point
	points = append(points, pts(0, 30_000, 21)...)         // 0 .. 600_000 (10 min)
	points = append(points, pts(2_400_000, 30_000, 21)...) // 40 min .. 50 min
	segments := Segments(points)

	cov := CoverageOf(segments, Window{From: 0, To: 3_600_000})
	if cov.RecordedMs != 1_200_000 {
		t.Errorf("want 1_200_000 ms recorded, got %d", cov.RecordedMs)
	}
	if cov.Ratio < 0.33 || cov.Ratio > 0.34 {
		t.Errorf("want ratio ~0.333, got %v", cov.Ratio)
	}
	if cov.Points != 42 {
		t.Errorf("want 42 points, got %d", cov.Points)
	}
}

// The conservative choice, pinned deliberately: scattered lone points evidence instants, not
// intervals, so they report near-zero coverage. `Points` is what stops that reading as "no data".
func TestCoverageOfIsolatedPointsReportsNoRecordedTime(t *testing.T) {
	points := []Point{{Ts: 0}, {Ts: 10_000_000}, {Ts: 20_000_000}}
	cov := CoverageOf(Segments(points), Window{From: 0, To: 30_000_000})

	if cov.RecordedMs != 0 {
		t.Errorf("isolated points must contribute no recorded time, got %d", cov.RecordedMs)
	}
	if cov.Ratio != 0 {
		t.Errorf("want ratio 0, got %v", cov.Ratio)
	}
	if cov.Points != 3 {
		t.Errorf("want Points=3 so this is distinguishable from an empty track, got %d", cov.Points)
	}
}

// An unqualified request has no window, so the observed span stands in for it — and the response says
// so, rather than leaving the client to guess which question it was answered.
func TestCoverageOfUnboundedWindowUsesObservedSpan(t *testing.T) {
	points := pts(5_000_000, 30_000, 11) // 5 min of data
	cov := CoverageOf(Segments(points), Window{})

	if cov.Window.From != 5_000_000 {
		t.Errorf("want window From at the first point, got %d", cov.Window.From)
	}
	if cov.Window.To != 5_300_000 {
		t.Errorf("want window To at the last point, got %d", cov.Window.To)
	}
	if cov.Ratio != 1 {
		t.Errorf("a single continuous run spans its own window entirely: want 1, got %v", cov.Ratio)
	}
}

// A window that legitimately starts at epoch 0 must be honoured, not mistaken for an unset one.
// Written because the first implementation tested each bound for zero and silently rescaled the
// ratio — harmless with real timestamps (~1.7e12), which is exactly why it needs a test to stay
// fixed.
func TestCoverageOfWindowStartingAtEpochZeroIsHonoured(t *testing.T) {
	points := pts(0, 30_000, 21) // 10 min of data
	cov := CoverageOf(Segments(points), Window{From: 0, To: 1_200_000})

	if cov.Window.To != 1_200_000 {
		t.Errorf("requested window must be kept, got %+v", cov.Window)
	}
	if cov.Ratio != 0.5 {
		t.Errorf("want ratio 0.5 against the requested window, got %v", cov.Ratio)
	}
}

func TestCoverageOfEmpty(t *testing.T) {
	cov := CoverageOf(Segments(nil), Window{From: 100, To: 200})
	if cov.RecordedMs != 0 || cov.Ratio != 0 || cov.Points != 0 {
		t.Errorf("want a zero coverage for no data, got %+v", cov)
	}
	// The requested window is echoed back even with no data: "we have nothing for this hour" is a
	// different statement from "we have nothing".
	if cov.Window.From != 100 || cov.Window.To != 200 {
		t.Errorf("want the requested window echoed, got %+v", cov.Window)
	}
}

// A window narrower than the data inside it would otherwise report over 100%, which reads as an
// arithmetic bug.
func TestCoverageOfRatioIsCapped(t *testing.T) {
	points := pts(0, 30_000, 121) // an hour of data
	cov := CoverageOf(Segments(points), Window{From: 0, To: 60_000})
	if cov.Ratio != 1 {
		t.Errorf("want ratio capped at 1, got %v", cov.Ratio)
	}
}
