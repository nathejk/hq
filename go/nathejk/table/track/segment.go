package track

// Turning a sequence of points into something a map can draw honestly (PRD 011).
//
// # A track is a set of segments, not a line
//
// This is the load-bearing decision of the whole feature, and it is a modelling one rather than a
// rendering preference. Nobody records unbroken for a 30-hour race: phones lock, apps are
// backgrounded and killed, batteries die, and nobody is watching their screen while walking at
// 3 a.m. Gaps of hours are the *normal* shape of this data.
//
// A flat array of points invites exactly one bug, and it is a bad one: a client joins consecutive
// points into a polyline and draws a straight line across three hours of silence, presenting it as a
// walked route. A lie drawn confidently on a map is worse than a visible gap — an operator deciding
// where to send a car will believe the line.
//
// Returning segments makes that misrendering structurally impossible rather than something a future
// frontend developer has to remember. That is why the split happens here, on the server, once.

// GapThresholdMs is how long a silence must be before it breaks a track in two.
//
// Five minutes, which is ten times the ~30 s sampling interval. Both directions are wrong in ways
// worth naming: too small and an ordinary track shatters into confetti, because a phone that misses a
// couple of samples has not stopped recording in any meaningful sense; too large and we bridge a gap
// we should have shown, which is the failure this whole file exists to prevent.
//
// PRD 011 §11 leaves the value open pending real data. Ten samples is a defensible starting point,
// not a finding — when there is real traffic, look at the actual distribution of deltas before
// defending this number.
const GapThresholdMs int64 = 5 * 60 * 1000

// Segment is one unbroken stretch of recording.
//
// From and To are the first and last point's timestamps, carried explicitly so a client can label a
// stretch without reaching into the points array — and so a reduced segment (task 146) still reports
// the interval it actually covers rather than the interval its surviving points suggest.
type Segment struct {
	From   int64   `json:"from"`
	To     int64   `json:"to"`
	Points []Point `json:"points"`
}

// Window is the interval a track was asked for.
type Window struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// Coverage says how much of the requested window actually has data.
//
// It exists because this data is sparse *and irregularly sparse*, so an operator cannot tell a
// well-recorded track from a nearly-empty one by looking at a map — both are lines. Stating coverage
// is what lets them know whether to reason from it. Absence of data must not be mistakable for
// evidence of absence.
type Coverage struct {
	Window Window `json:"window"`

	// RecordedMs is the summed duration of the segments.
	//
	// Deliberately **conservative**: an isolated point contributes nothing, because a single fix
	// evidences an instant, not an interval. A track of twenty scattered points therefore reports
	// near-zero coverage, which is the honest answer — we know twenty instants and nothing about
	// the time between them. Attributing each lone point a sampling interval would inflate the
	// figure with an assumption about the producer's behaviour, and this number's whole job is to
	// stop an operator over-trusting the data.
	RecordedMs int64 `json:"recordedMs"`

	// Ratio is RecordedMs over the window's length, 0..1. Zero when the window has no length.
	Ratio float64 `json:"ratio"`

	// Points is how many positions the track holds, before any reduction.
	//
	// Reported alongside the ratio because the two say different things: a thin-but-wide track (many
	// isolated points, low ratio) and a genuinely empty one (no points) both have a low ratio, and
	// only this distinguishes them.
	Points int `json:"points"`
}

// Segments splits ordered points into unbroken stretches.
//
// Points must be in ascending ts order, which is what the querier returns — ordering here as well
// would hide a bug in the query rather than fix one, and the primary key makes the sort free there.
//
// Returns an empty slice rather than nil for no points: this is serialised straight to JSON, and
// `[]` lets every client iterate unconditionally while `null` makes each one choose between a guard
// and a crash.
func Segments(points []Point) []Segment {
	segments := make([]Segment, 0, 1)
	if len(points) == 0 {
		return segments
	}

	start := 0
	for i := 1; i < len(points); i++ {
		if points[i].Ts-points[i-1].Ts <= GapThresholdMs {
			continue
		}
		segments = append(segments, segmentOf(points[start:i]))
		start = i
	}
	return append(segments, segmentOf(points[start:]))
}

func segmentOf(points []Point) Segment {
	return Segment{
		From:   points[0].Ts,
		To:     points[len(points)-1].Ts,
		Points: points,
	}
}

// CoverageOf reports how much of a window the segments cover.
//
// When the window has no positive span — as an unqualified request produces — the observed span,
// first point to last, is used instead. The ratio then answers a subtly different but still useful
// question: how much of the period this person was reporting at all do we have continuous data for.
// Reported in Window either way, so a client is never left guessing which question it got an answer
// to.
//
// The test for "unspecified" is `To <= From` rather than either bound being zero, which matters: a
// window legitimately starting at epoch 0 is indistinguishable from an unset one if you test the
// bounds separately, and it silently rescales the ratio. Real requests carry ~1.7e12, so this never
// bit in production — it bit in a test, which is the cheap place for it to happen.
func CoverageOf(segments []Segment, w Window) Coverage {
	cov := Coverage{Window: w}

	var recorded int64
	for _, s := range segments {
		recorded += s.To - s.From
		cov.Points += len(s.Points)
	}
	cov.RecordedMs = recorded

	if cov.Window.To <= cov.Window.From && len(segments) > 0 {
		cov.Window = Window{
			From: segments[0].From,
			To:   segments[len(segments)-1].To,
		}
	}

	if span := cov.Window.To - cov.Window.From; span > 0 {
		cov.Ratio = float64(recorded) / float64(span)
		// A window narrower than the data it contains would otherwise report more than 100%
		// coverage, which is meaningless to read and looks like an arithmetic bug.
		if cov.Ratio > 1 {
			cov.Ratio = 1
		}
	}

	return cov
}
