# 147 — GET /api/telemetry/person/:personId/track

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §8. Depends on tasks 141, 145, 146.

One person's track, segmented and reduced:

```
GET /api/telemetry/person/:personId/track?from=<ms>&to=<ms>&maxPoints=<n>
→ { personId, personType, coverage, resolution, segments: [ … ] }
```

`:personId` is accepted from either id space (`memberID` or `crewmemberID`) without
qualification — they are opaque and do not collide, and the projection does not care which it
holds.

`from`/`to` are epoch milliseconds to match the stored `ts` exactly; no date parsing, no
timezone ambiguity. Sensible defaults when omitted (the current year's window) and a bounded
`maxPoints` default so an unqualified request can never return the raw ceiling.

Requires OpenAPI annotations.

## Acceptance Criteria

- [x] Route registered; returns segments + coverage + resolution
- [x] `from`/`to`/`maxPoints` honoured, all optional, with bounded defaults
- [x] Works for both `memberID` and `crewmemberID` without a type hint
- [x] Unknown `personId` returns an empty track, not a 404 or an error
- [x] Authenticated; OpenAPI annotations present
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `buildTrack` composing segment → reduce → coverage, shared with task 149, plus param reading and the route.
- 2026-09-03 — **Coverage is computed from the unreduced segments, and the order is a decision, not an accident.** Coverage describes how much was *recorded*; measured after reduction it would describe how much survived the point budget, so a well-recorded track would appear to thin out the moment an operator zoomed out — the number would be reporting a property of the request rather than of the data. There is a test asserting coverage is identical at `maxPoints=50` and `maxPoints=100000`.
- 2026-09-03 — Unparseable `from`/`to`/`maxPoints` are treated as **absent rather than rejected with a 400**. These parameters are a view's framing of a picture, not a command: a garbled value should show an operator a sensible default track, not an error page in the middle of an incident. `maxPoints` is also clamped down to the default, so no request can ask for the raw ceiling.
- 2026-09-03 — Unknown `personId` returns an **empty track, not a 404**. "We have no positions for this scout" is an answer, and the caller is a dialog opened from a name in a list — the person's existence is already established.
- 2026-09-03 — Fixed my own broken test data: the two-segment test placed the second run *inside* the first (100 points at 30 s spans 2,970,000 ms, not 600,000) and then asserted they were apart. The interval arithmetic is now written out as named constants rather than eyeballed. Third fixture bug in this PRD — the pattern is me being casual with millisecond arithmetic, so the constants are now spelled out everywhere.
- 2026-09-03 — ✅ **Verified against the running dev API with real data, which is the best outcome of this task.** There are already **1,202 real points** for one person in dev, and the endpoint handles them exactly as designed:
  - segmented into **2 segments** across a ~10-hour silence, so nothing draws a line through the gap
  - **coverage 0.50** — honestly reporting that half the window has no data
  - `maxPoints=6` reduced 1,202 points to 4: the first segment is stationary (identical coordinates for ten hours) and correctly collapsed to its endpoints, which is the behaviour task 146 documented
  - `personType: ""` — these are messages from **before** `userType` was added. Stored as sent rather than derived, which is exactly the case that justified that decision: had hq joined against today's directory it would have invented a role for historical data.
- 2026-09-03 — Side finding: the api has been running this consumer through hot-reload without fatalling, and points are materialising — so **the `TELEMETRY` stream already exists in dev** and task 139's dev half is effectively satisfied. Noted there.
- 2026-09-03 — Completed.
