# 110 — dispatch API — board and task create/edit/cancel

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §8 (API endpoints). Depends on task 109.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/dispatch` | the board: queued tasks, tours, units, duty |
| POST | `/api/dispatch/task` | create a task |
| GET | `/api/dispatch/task/:id` | one task with its timeline |
| PATCH | `/api/dispatch/task/:id` | edit times, description, priority, places |
| POST | `/api/dispatch/task/:id/pickedup` | people aboard (member transitions in task 118) |
| POST | `/api/dispatch/task/:id/cancelled` | cancel, with a required reason |

`commands.go` on `dispatch` publishing the events, dirty-checking before publishing as the repo
does. Every state change appends a timeline entry.

**Every endpoint needs OpenAPI annotations** — a repo requirement, and here also the way the
future driver app learns the shape without reading Go (§8).

Collections serialise as `[]`, never `null`.

## Acceptance Criteria

- [ ] Six endpoints, routed, with OpenAPI annotations
- [ ] `GET /api/dispatch` returns queued tasks (oldest first), tours with ordered stops, and units
- [ ] Cancel without a reason is rejected with 422
- [ ] Every write appends a `dispatch_activity` entry with timestamp, actor and note
- [ ] Handler tests covering create, edit, cancel, pickedup and the board shape

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
