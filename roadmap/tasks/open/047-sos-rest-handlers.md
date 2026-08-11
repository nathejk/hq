# 047 — SOS REST handlers and routes

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] All nine routes registered and working end to end against the dev stack
- [ ] Partial patch semantics correct (absent ≠ empty), each changed field one event
- [ ] Create rejects empty headline or description with a validation error
- [ ] Deleted case returns 404 on `GET /api/sos/:id` and is absent from the list
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
