# 145 — segment tracks on a gap threshold, report coverage

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §5, §6, §8. Pure server-side logic, shared by tasks 146 and 149. Worth doing before
either endpoint so neither invents its own shape.

**Gaps are the normal shape of this data, not an anomaly.** Nobody records unbroken for a
30-hour race — phones are locked, apps are backgrounded and killed, batteries die — so tracks
will be a handful of recorded stretches separated by hours of nothing.

Therefore a track is modelled as ordered **segments**, split when the delta between
consecutive points exceeds a gap threshold:

```
{ personId, personType, coverage: { window: {from,to}, recorded: <ms>, ratio: 0.31 },
  segments: [ { from, to, points: [ {ts,lat,lng,accuracy}, … ] }, … ] }
```

This is the load-bearing decision of the whole feature: if the API returns segments, a client
**cannot** draw a solid line across three hours of silence and present it as a walked route. A
lie drawn confidently on a map is worse than a visible gap.

`coverage` exists so an operator can tell a thin track from a well-recorded one *before*
reasoning from it — absence of data must not be mistakable for evidence of absence.

Gap threshold is an open question in PRD 011 §11; start at **5 minutes** (10× the ~30 s
sampling interval), in one named const, and log the choice.

Implement as a plain function over ordered points so it is testable without a database or a
request — the same reason the producer keeps `track.Clean` out of its HTTP handler.

## Acceptance Criteria

- [ ] A `Segment` type and a segmenting function over `[]Point` in `table/track`
- [ ] Splits on a threshold held in one named const
- [ ] Per-track `coverage` (window, recorded duration, ratio)
- [ ] Unit tests: no points, one point, one continuous run, several gaps, gap exactly at the threshold
- [ ] `go test ./...` clean for the package

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
