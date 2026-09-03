# 149 — GET /api/telemetry/patrulje/:teamId/track

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §1, §6, §8. Depends on tasks 145, 146, 147, 148. This is the endpoint the whole PRD
exists for.

For a spejder, the unit that matters is the patrol, not the person. So one call returns the
patrol's whole movement:

```
GET /api/telemetry/patrulje/:teamId/track?from&to&maxPoints
→ { team, members: [ { memberId, name?, personType, membership:{from,to?}, coverage,
                       resolution, segments:[…] } ],
    scans: [ { qrId, uts, latitude, longitude, … } ] }
```

Three sources joined at read time:

1. membership intervals (current **and former**) from task 148
2. points from `track_point`, per member, **clipped to that member's membership interval** —
   so a scout who left at 11:00 has a line that ends at 11:00, and their later movement with
   another team does not leak onto this patrol's map
3. that team's scans from `scan.GetAll(Filter{TeamID})` — **never reduced**, and note `scan.uts`
   is **seconds** while `track_point.ts` is **milliseconds**; convert at this boundary or the
   two will not line up on one time axis

Tracks are segmented (145) and reduced (146). Scans are not.

Requires OpenAPI annotations.

## Acceptance Criteria

- [x] Returns per-member segmented tracks + the team's scans in one response
- [x] Points clipped to each member's membership interval
- [x] Former members included, with a name where one survives
- [x] Scan `uts` (s) and point `ts` (ms) reconciled to one unit in the response
- [x] Scans returned exactly, never reduced
- [x] A team with no telemetry still returns its scans
- [x] Authenticated; OpenAPI annotations present
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: join membership (148) × points (141) × scans, reusing `buildTrack` from 147 so both endpoints produce one shape.
- 2026-09-03 — Clipping is done by **narrowing the queried window** per member (`clipWindow`) rather than filtering points afterwards — the database does the work, and a member who left at 11:00 never has their later points read at all. Six sub-tests, including the one that matters: a zero membership bound means *unknown*, not epoch, so it must neither clip to 1970 nor widen a requested window.
- 2026-09-03 — Members with **no positions are still listed**. Dropping them would leave an operator unable to tell whether a scout was absent from the patrol or merely from the data — which is the distinction this whole PRD keeps insisting on.
- 2026-09-03 — Scans get their own list and their own type, never folded into the tracks and never reduced. They are the only *certain* positions on the map — a scan happened at a known post, at a known time, witnessed by a person — whereas a track point is a phone's best guess.
- 2026-09-03 — ⚠️ **Found and fixed a pre-existing bug in `scan`'s querier, unrelated to this PRD but blocking this task.** `GetAll` scanned `uts` into a `var uts` declared outside the row loop and never assigned it to the row, so **every scan it returned carried `uts=0`** while the table held the real value. It had been silently wrong for `GET /api/patrulje/:id/scans` too — I verified the old endpoint reproduced it before touching anything, so this was not my regression. It surfaced here because this endpoint puts scans and tracks on one time axis, where every marker landed in 1970 instead of merely being an unused field. One-line fix, with the history in a comment; `/api/patrulje/:id/scans` now returns real timestamps, which is a visible behaviour change to an existing endpoint — a bugfix, but worth knowing.
- 2026-09-03 — ✅ **Verified against real dev data.** For a 2025 patrol: 4 members returned with names and open memberships, 36 scans, first scan `ts` 1758320577000 → 2025-09-19T22:22:57 (correct, and correctly in milliseconds). The members have no track segments because telemetry only exists for 2026 — the honest result, and they are still listed.
- 2026-09-03 — Noted inconsistency, not fixed: **membership is year-scoped but `scan.GetAll` is not** (it filters on teamId alone, pre-existing). So requesting a previous year's team under the current year slug returns its scans with zero members. Harmless in practice — the SPA only navigates to current-year teams — but it is a seam somebody will trip over eventually.
- 2026-09-03 — Full repo suite and `go vet ./...` clean. Completed.
