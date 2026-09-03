# 145 — segment tracks on a gap threshold, report coverage

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] A `Segment` type and a segmenting function over `[]Point` in `table/track`
- [x] Splits on a threshold held in one named const (`GapThresholdMs`)
- [x] Per-track `coverage` (window, recorded duration, ratio)
- [x] Unit tests: no points, one point, one continuous run, several gaps, gap exactly at the threshold
- [x] `go test ./...` clean for the package

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `segment.go` as pure functions over `[]Point` — no db, no request — plus a table-driven test file.
- 2026-09-03 — Gap threshold set to **5 minutes**, i.e. ten samples at ~30 s. Recorded the reasoning in the const because both directions fail visibly: too small and an ordinary track shatters into confetti (a phone missing two samples has not stopped recording in any meaningful sense), too large and we bridge a gap we should have shown — the exact failure this file exists to prevent.
- 2026-09-03 — **Coverage is deliberately conservative: an isolated point contributes zero recorded time.** A single fix evidences an instant, not an interval, so twenty scattered points report ~0% coverage. That looks harsh but it is the honest answer, and the alternative — crediting each lone point a sampling interval — would inflate the number with an assumption about producer behaviour. Since this figure exists precisely to stop an operator over-trusting sparse data, erring toward understatement is the right bias. Added `Points` to the payload so a thin-but-wide track stays distinguishable from an empty one, which is the one thing the conservative ratio genuinely loses.
- 2026-09-03 — ⚠️ **A test caught a real bug, worth recording.** I first detected "no window given" by testing each bound for zero — so a window legitimately starting at epoch 0 was mistaken for unset, and the ratio was silently rescaled against the observed span instead (0.4 where 0.333 was correct). Real requests carry ~1.7e12 so this would never have bitten in production, and would therefore never have been found there either. Now tested as `To <= From`, with a regression test pinning that a window from epoch 0 is honoured.
- 2026-09-03 — `Segments` returns `[]` and never nil: it serialises straight to JSON, and `null` would make every client choose between a guard and a crash.
- 2026-09-03 — Also pinned that segmentation **keeps every point** — it groups, it does not filter. Dropping data here would be indistinguishable from the phone never having recorded it.
- 2026-09-03 — ✅ All criteria. 12 tests pass (`go test ./nathejk/table/track/...`), including the threshold boundary in both directions. Completed.
