# 047 — SOS REST handlers and routes

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §8. New handler file `go/cmd/api/sos.go`, one `<verb>SosHandler` per
route, reading via `app.models.Sos` and writing via `app.commands.Sos`, using the
`app.ReadJSON` / `app.WriteJSON` / `app.ServerErrorResponse` helpers. Do **not** copy the
legacy `sos-routes.go` switch-on-method+path style.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/sos` | List cases (open/closed), ordered by last activity |
| GET | `/api/sos/:id` | One case with timeline + associated teams |
| POST | `/api/sos` | Create (headline + description, both required); returns the case |
| PATCH | `/api/sos/:id` | `headline`, `description`, `severity`, `assigneeSectionSlug`, `status` |
| DELETE | `/api/sos/:id` | Soft delete |
| POST | `/api/sos/:id/comment` | Add comment; returns its id |
| PATCH | `/api/sos/:id/comment/:commentId` | Edit comment (appends, does not overwrite) |
| PUT | `/api/sos/:id/team/:teamId` | Associate a patrol |
| DELETE | `/api/sos/:id/team/:teamId` | Disassociate |

Notes:

- The `PATCH` input struct needs **pointer or presence-tracked fields** so "absent" is
  distinguishable from "set to empty" — same pattern as `updateYearHandler` /
  `patchKlanHandler`.
- Year comes from `X-YearSlug` via `app.YearSlug(r)`; **no year in the path or query**.
- The handler resolves the actor (`requestctx.UserFrom`) and passes it to the command.
- Soft-deleted case → 404 on the detail route.
- Only patrols may be associated; reject klan ids.

## Acceptance Criteria

- [x] All nine routes registered and working end to end against the dev stack
- [x] Partial patch semantics correct (absent ≠ empty), each changed field one event
- [x] Create rejects empty headline or description with a validation error
- [x] Deleted case returns 404 on `GET /api/sos/:id` and is absent from the list
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up. Plan: `cmd/api/sos.go` with nine handlers, one error mapper,
  and the routes; verify each route against the running dev stack rather than by
  inspection.
- 2026-08-11 — `GET /api/sos` groups into `open` / `closed` server-side. Both halves come
  from one query and the split is how the screen is laid out, so deriving it per client
  is duplicated work.
- 2026-08-11 — `GET /api/sos/:id` enriches each associated patrol with number, name,
  group, korps and contact phone, read live from the patrulje projection rather than
  copied into `sos_team` — so a renamed patrol is never stale on a case. A patrol that
  cannot be read still returns a row: losing the association because a projection is
  behind would hide that the case is about somebody.
- 2026-08-11 — **Found while testing, and it mattered:** `POST /api/sos` read the case
  back immediately and lost the race with the projection, answering with just an id. The
  SPA navigates straight to the new case, so it would have shown not-found until the
  first signal. Now it synthesises the case from what was published and answers 202
  ("accepted, not yet projected") for the client to seed its cache with.
- 2026-08-11 — **The same race made PATCH actively harmful.** Reading back after publish
  returned *pre-patch* values, so the SPA — which applies the operator's change
  optimistically — would have flickered backwards to the old value and forwards again
  when the signal arrived. Exactly the "stale value that looks live" failure PRD 004 §12
  describes. PATCH now answers 202 with an echo of the accepted patch and reads nothing
  back.
- 2026-08-11 — Added json tags to `sos.PatchCommand`: the echo was serialising Go field
  names (`"Headline"`). Caught by looking at the actual response rather than trusting it.
- 2026-08-11 — An assignee must be a section flagged assignable for the year, enforced in
  the handler, so the API cannot route a case somewhere the Organisation page says it
  should not go. Clearing the assignee stays allowed.
- 2026-08-11 — Disassociation deliberately does **not** check the patrol exists: undoing
  a mistake must keep working even if the patrol has become unreadable.
- 2026-08-11 — ✅ Verified against the dev stack, every route: create (201/202), create
  with empty description (400), list, single case with timeline in stream order, patch
  severity + assignee, assignee not-assignable (400), comment, comment edit, unknown
  comment (404), associate patrol, associate klan (400), close, close again (one `closed`
  entry in the timeline — idempotent), reopen, delete, then 404 on the detail and absent
  from the list.
- 2026-08-11 — All criteria met. Moving to done.
