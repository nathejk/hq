# 101 — Note endpoints

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

Three endpoints over task 100's commands (PRD 008 §8), year-scoped via `X-YearSlug`:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/member/:memberId/notes` | The thread, oldest first |
| POST | `/api/member/:memberId/notes` | Add a note |
| PATCH | `/api/member/:memberId/notes/:noteId` | Correct a note |

Registered on the member, beside the lifecycle routes. Deliberately **not** folded into
`GET /api/member/:memberId`: that endpoint feeds a modal opened one scout at a time, while the
thread is a separately cacheable, separately invalidated resource the SPA holds per member.

Watch httprouter's constraint — a static segment cannot sit where a sibling holds a wildcard.
`/api/member/:memberId/notes` is fine (the wildcard is the parent), but check the router still
builds; a conflict panics at boot, and `stream_test.go`'s single `app.routes()` call is what
catches it.

Refusals map to Danish messages phrased as what to do, following `shelterCommandError`.

**OpenAPI annotations on all three** — `@Summary`, `@Description`, `@Tags`, `@Produce`,
`@Success`, `@Failure`, `@Router` — per `cmd/api/order.go`.

## Acceptance Criteria

- [ ] Three routes registered and handled; router boots
- [ ] No `sosId` required
- [ ] Empty/over-long/wrong-member refusals answered with Danish field messages, not raw domain
      errors
- [ ] Unchanged edit answers 200
- [ ] OpenAPI annotations complete on all three
- [ ] Handler tests for the happy paths and each refusal

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
