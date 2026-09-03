package track

import "math"

// Making a 30-hour track fit down a wire and onto a screen (PRD 011).
//
// # Why the server reduces
//
// The ceiling is ~3,600 points per person per event, so a six-member patrol is ~21,600. Sparse
// recording usually makes it far less, but an endpoint cannot rely on that — and the one patrol that
// did keep its phones awake is exactly the one somebody is looking at during an incident. Raw points
// at the ceiling are megabytes of JSON and a visibly janky Leaflet map.
//
// It is also more detail than any screen can show: a 30-hour route at display zoom cannot resolve
// 30-second steps. So reduction is not a compromise on fidelity here, it discards information that
// could not have been seen.
//
// # Douglas–Peucker, not every nth point
//
// Dropping every other point is simpler and wrong in a specific way: it thins straight stretches and
// corners equally, so a sharp turn — the thing that makes a route recognisable as a place someone
// walked — erodes at the same rate as a hundred metres of straight road. Douglas–Peucker keeps the
// points that carry the shape and drops the ones that lie on a line between their neighbours, which
// is precisely the distinction worth making when the output is a picture.
//
// # Reduction never bridges a gap
//
// Every function here works *within* one segment. That is the constraint that keeps task 145's
// guarantee intact: simplification that could span a gap would reintroduce the straight-line-across-
// three-hours-of-silence lie by the back door, with the added insult that the line would look
// deliberate.

// DefaultMaxPoints bounds a track when the caller does not say.
//
// 2,000 across a whole track: enough that a reduced route is visually indistinguishable from the raw
// one at any zoom a browser will render, and small enough that a six-member patrol stays a few
// hundred kilobytes rather than megabytes. A caller wanting the raw data asks for a narrow time
// window instead, which is the honest way to get more detail — more points over less time.
const DefaultMaxPoints = 2000

// Reduce simplifies segments so their combined point count is at most maxPoints.
//
// The budget is shared across segments in proportion to how many points each holds, so a long
// stretch is not thinned to the same handful as a two-minute one. Every segment keeps at least its
// endpoints: a segment reduced out of existence would be a gap that never happened.
//
// Returns the segments and whether anything was actually dropped, so the endpoint can tell the client
// what it received rather than leaving it to guess whether it is looking at everything.
func Reduce(segments []Segment, maxPoints int) ([]Segment, bool) {
	if maxPoints <= 0 {
		maxPoints = DefaultMaxPoints
	}

	total := 0
	for _, s := range segments {
		total += len(s.Points)
	}
	if total <= maxPoints {
		return segments, false
	}

	// Two points per segment are spoken for before anything is shared out — the endpoints are not
	// negotiable — so only the remainder is distributed by size.
	floor := 2 * len(segments)
	budget := maxPoints - floor
	if budget < 0 {
		budget = 0
	}

	out := make([]Segment, len(segments))
	reduced := false
	for i, s := range segments {
		allowance := 2
		if total > 0 {
			allowance += int(float64(budget) * float64(len(s.Points)) / float64(total))
		}

		points := simplify(s.Points, allowance)
		if len(points) != len(s.Points) {
			reduced = true
		}
		out[i] = Segment{From: s.From, To: s.To, Points: points}
	}
	return out, reduced
}

// simplify reduces one segment's points to at most max, preserving shape.
//
// Douglas–Peucker takes a distance tolerance rather than a point count, so the tolerance is found by
// bisection: about twenty iterations of an O(n log n) pass, which is nothing next to the query that
// fetched the points, and it means callers can think in "how many points may I have" rather than in
// degrees of latitude.
func simplify(points []Point, max int) []Point {
	if max < 2 {
		max = 2
	}
	if len(points) <= max {
		return points
	}

	// An upper bound big enough that everything between the endpoints is dropped: two points always
	// satisfy any budget, so the search cannot fail to terminate on something usable.
	lo, hi := 0.0, 1.0
	best := []Point{points[0], points[len(points)-1]}

	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		kept := douglasPeucker(points, mid)
		if len(kept) > max {
			// Too much detail: demand a coarser tolerance.
			lo = mid
		} else {
			// Fits. Keep it and try to afford more detail.
			best = kept
			hi = mid
		}
	}
	return best
}

// douglasPeucker keeps the points that carry the line's shape to within epsilon.
//
// Iterative rather than recursive: a 3,600-point segment of a straight road is the pathological case
// for the recursive form, and a stack overflow in a read endpoint is a poor way to learn that.
func douglasPeucker(points []Point, epsilon float64) []Point {
	if len(points) < 3 {
		return points
	}

	keep := make([]bool, len(points))
	keep[0] = true
	keep[len(points)-1] = true

	type span struct{ first, last int }
	stack := []span{{0, len(points) - 1}}

	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if s.last <= s.first+1 {
			continue
		}

		var (
			worst    float64
			worstIdx int
		)
		for i := s.first + 1; i < s.last; i++ {
			if d := perpendicularDistance(points[i], points[s.first], points[s.last]); d > worst {
				worst, worstIdx = d, i
			}
		}

		if worst <= epsilon {
			continue
		}
		keep[worstIdx] = true
		stack = append(stack, span{s.first, worstIdx}, span{worstIdx, s.last})
	}

	out := make([]Point, 0, len(points))
	for i, k := range keep {
		if k {
			out = append(out, points[i])
		}
	}
	return out
}

// perpendicularDistance is how far p sits off the line ab, in degrees.
//
// Degrees, not metres: no cos(latitude) correction and no earth radius, because the only use is
// *comparing* deviations within one short segment to pick the most significant point. Over a few
// kilometres at Danish latitudes the distortion is a constant factor, which cancels in a comparison.
// Converting properly would cost trigonometry per point and change nothing about which point wins.
func perpendicularDistance(p, a, b Point) float64 {
	dx := b.Lng - a.Lng
	dy := b.Lat - a.Lat

	// A degenerate line — the endpoints coincide, which happens when a phone reports the same fix
	// repeatedly while stationary. Fall back to plain distance from the point.
	if dx == 0 && dy == 0 {
		return math.Hypot(p.Lng-a.Lng, p.Lat-a.Lat)
	}

	// Twice the triangle's area over the base length.
	area := math.Abs(dx*(a.Lat-p.Lat) - dy*(a.Lng-p.Lng))
	return area / math.Hypot(dx, dy)
}
