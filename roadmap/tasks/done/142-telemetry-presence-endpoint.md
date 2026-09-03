# 142 — GET /api/telemetry/presence

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Route registered in `cmd/api/routes.go`
- [x] Returns id → `{ ts, personType }` for the current year, from `track_latest`
- [x] No join against `spejder`/`senior`/`personnel`/`crewmember`
- [x] Authenticated like other member-data endpoints; no unauthenticated exposure — **with a caveat, see log**
- [x] OpenAPI annotations present
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: add `Track track.Queries` to `data.Models`, a `cmd/api/telemetry.go` handler with swagger annotations in the `dispatch.go` house style, and the route.
- 2026-09-03 — Response is a JSON **object keyed by personId**, not an array. The only question asked of this payload is "is this id in here, and when?", which a map answers without every list view building an index of its own on each render. Emitted as `{}` rather than `null` when empty, so an early-season page can index into it unconditionally.
- 2026-09-03 — Deliberately carries **no coordinates**, only `ts` and `personType`. It is fetched on nearly every page, so this keeps it small *and* avoids disclosing where people are on pages that have no business showing it. Positions are read only when someone asks for a route (tasks 147, 149).
- 2026-09-03 — Included `personType` even though the glyph does not branch on it: it makes a presence response readable on its own, and lets a future per-population staleness rule land without a new field.
- 2026-09-03 — Removed a `var _ track.Queries = (track.Queries)(nil)` line I had written as a "compile-time assurance". It is a tautology and asserts nothing — better deleted than left looking meaningful.
- 2026-09-03 — ⚠️ **Finding worth escalating.** The auth criterion is only half true. Everything under `/api/` goes through `app.authenticate`, so this endpoint is exactly as protected as `/api/patrulje/:id` — but that middleware attributes every request to an *anonymous* user and enforces nothing; authentication lives in an external service (`AUTH_BASEURL`). PRD 011 §6 claimed telemetry is "restricted to authenticated HQ users", which overstates what this repo does. Qualified the PRD in place rather than leaving the claim standing, and noted in `telemetry.go` that the promise is kept by whatever fronts the api. If position history is meant to be held to a *higher* bar than the rest of the read model, that is unimplemented and needs its own task — knj's call.
- 2026-09-03 — ✅ All criteria. `go build`, `go vet`, `gofmt` clean. Wired `Track` into `data.Models` and `NewModels`.
- 2026-09-03 — Completed. `GET /api/telemetry/presence` serves the glyph for every people list without any other endpoint changing shape.
