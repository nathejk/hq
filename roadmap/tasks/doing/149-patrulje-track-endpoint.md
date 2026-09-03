# 149 — GET /api/telemetry/patrulje/:teamId/track

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Returns per-member segmented tracks + the team's scans in one response
- [ ] Points clipped to each member's membership interval
- [ ] Former members included, with a name where one survives
- [ ] Scan `uts` (s) and point `ts` (ms) reconciled to one unit in the response
- [ ] Scans returned exactly, never reduced
- [ ] A team with no telemetry still returns its scans
- [ ] Authenticated; OpenAPI annotations present
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
