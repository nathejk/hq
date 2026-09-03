# 147 — GET /api/telemetry/person/:personId/track

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Route registered; returns segments + coverage + resolution
- [ ] `from`/`to`/`maxPoints` honoured, all optional, with bounded defaults
- [ ] Works for both `memberID` and `crewmemberID` without a type hint
- [ ] Unknown `personId` returns an empty track, not a 404 or an error
- [ ] Authenticated; OpenAPI annotations present
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
