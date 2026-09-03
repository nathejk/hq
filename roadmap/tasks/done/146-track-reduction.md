# 146 — server-side track reduction

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §6, §8. Depends on task 145.

At ~30 s sampling the ceiling for a 30-hour race is ~3,600 points per person, ~21,600 for a
six-member patrol. Sparse recording will usually make it far less, but the endpoint cannot rely
on that. Raw points at the ceiling are megabytes of JSON and a janky Leaflet map — and more
detail than any screen can show, since a 30-hour route at display zoom cannot resolve
30-second steps.

So both track endpoints reduce: a `maxPoints` parameter, either nth-point or Douglas–Peucker
(prefer Douglas–Peucker — it keeps corners, which is what makes a route legible).

Two rules:

- **Reduction is applied within a segment, never across one.** Simplification must not be able
  to bridge a gap; that would undo task 145.
- **The response states the resolution it was reduced to**, so the UI can say so and offer full
  fidelity for a narrow time window.
- **Scans are never reduced** — they are few, exact, and the anchor points an operator reasons
  from (relevant to task 149).

Testable without a database, like task 145.

## Acceptance Criteria

- [x] Reduction function over a segment's points, honouring `maxPoints`
- [x] Never merges across segment boundaries
- [x] Endpoints report the applied resolution / whether reduction occurred
- [x] First and last point of a segment always retained
- [x] Unit tests including "fewer points than the budget" (no-op) and a corner-preservation case
- [x] `go test ./...` clean for the package

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `reduce.go`, pure functions, Douglas–Peucker preferred over nth-point.
- 2026-09-03 — Chose **Douglas–Peucker** and recorded why in the file: nth-point thins straight stretches and corners at the same rate, so a sharp turn — the thing that makes a route recognisable as somewhere a person walked — erodes as fast as a hundred metres of straight road. DP drops points that lie on a line between their neighbours, which is exactly the right distinction when the output is a picture. There is a test asserting a right-angle turn survives a reduction to 10 points.
- 2026-09-03 — DP takes a distance tolerance, not a point count, so `simplify` **bisects the tolerance** (24 iterations) to hit a budget. Callers get to think in "how many points may I have" rather than in degrees of latitude, and the cost is nothing next to the query that fetched the points.
- 2026-09-03 — Budget is shared **in proportion to segment size**, with two points per segment reserved before anything is distributed. A segment reduced out of existence would be a gap that never happened, so endpoints are not negotiable — tested with an absurd budget of 1 across 10 segments.
- 2026-09-03 — Iterative DP, not recursive: a 3,600-point straight stretch is the pathological case for the recursive form, and a stack overflow in a read endpoint is a poor way to discover that.
- 2026-09-03 — `perpendicularDistance` works in **degrees with no cos(latitude) correction**, deliberately. The only use is *comparing* deviations within one short segment to find the most significant point; at Danish latitudes over a few km the distortion is a constant factor that cancels in a comparison. Proper conversion would cost trigonometry per point and change nothing about which point wins.
- 2026-09-03 — ⚠️ **Second test-data trap in a row, same root cause.** `TestReduceHonoursTheBudget` failed at first (got 2 points where ≥400 was expected) because `pts()` from segment_test.go repeats a single coordinate — a *stationary* track, which DP correctly collapses to its endpoints however large the budget. Same for a perfectly straight line. So the code was right and the fixture was wrong. Added `wanderingPts` (a gentle arc with a fine wobble, so no three consecutive points are collinear) for budget tests, and turned the surprise into two explicit tests: a straight line **must** collapse to 2 points, and a stationary run must not divide by zero. Worth having, so nobody later "fixes" the collapse into spending the budget on redundant points.
- 2026-09-03 — ✅ All criteria. 10 reduction tests + 13 segmentation tests pass; `go vet` clean. Completed.
