# 142 — GET /api/telemetry/presence

**Status:** doing
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:**

## Description

PRD 011 §6, §8. Depends on task 141.

One endpoint that makes the position indicator possible without widening ten existing
handlers: for the current year, every `personId` that has ever reported a position, with its
last-reported timestamp.

```
GET /api/telemetry/presence
→ { "presence": { "<personId>": { "ts": 1788437919856, "personType": "gøgler" }, … } }
```

Keyed by the **raw `personId`**, with **no name resolution and no join against any people
table**. This is possible because `personId` is either a `memberID` (spejder, senior) or a
`crewmemberID` (= `userId`, shared with `personnel` for gøgler/friend/bandit), both opaque
non-colliding ids that every people-list row already carries. So the frontend asks "is my
row's id in here?" and nothing needs mapping.

Reads `track_latest` only — never `track_point`. It is fetched on nearly every page, so it
must stay a single indexed scan of one row per person.

Requires OpenAPI annotations, like every endpoint in this repo.

## Acceptance Criteria

- [ ] Route registered in `cmd/api/routes.go`
- [ ] Returns id → `{ ts, personType }` for the current year, from `track_latest`
- [ ] No join against `spejder`/`senior`/`personnel`/`crewmember`
- [ ] Authenticated like other member-data endpoints; no unauthenticated exposure
- [ ] OpenAPI annotations present
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: add `Track track.Queries` to `data.Models`, a `cmd/api/telemetry.go` handler with swagger annotations in the `dispatch.go` house style, and the route.
