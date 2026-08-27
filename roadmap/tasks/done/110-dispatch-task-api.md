# 110 — dispatch API — board and task create/edit/cancel

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

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

- [x] Six endpoints, routed, with OpenAPI annotations
- [x] `GET /api/dispatch` returns queued tasks (oldest first), tours with ordered stops, and units
- [x] Cancel without a reason is rejected with 422
- [x] Every write appends a `dispatch_activity` entry with timestamp, actor and note
- [x] Handler tests covering create, edit, cancel, pickedup and the board shape

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — Picked up. Plan: task commands in `dispatch/commands.go`, then the six handlers in
  `cmd/api/dispatch.go` with swagger annotations in the house style.
- 2026-08-27 — **The board is one payload with one cache key.** Tasks, tours and units in a single
  GET, and every task state in one query. Three filtered queries would be three round trips to
  draw one screen, and — more importantly — two cache keys would let a task dragged into a tour
  leave two panes disagreeing about where it is.
- 2026-08-27 — Units are assembled in the handler from sections + vehicles + crew, because the
  dispatch entity knows nothing about vehicles and must not: the unit *is* a subsection of the
  organisation tree. `Vehicles` is a list, since two cars in one unit is a mistake PRD 009 flags
  rather than forbids, and a single field would have to pick one silently. A unit whose section
  has been renamed away falls back to its slug, so the drift §8 warns about is visible.
- 2026-08-27 — **The test found a real bug that would have been invisible until a race night.** I
  had typed the PATCH times as `**int64`, believing an explicit `null` would decode to a non-nil
  outer pointer holding nil. It does not: encoding/json leaves the outer pointer nil for both an
  absent field *and* a null, so "ryd deadline" would silently have meant "lad den stå" — an
  editor that cannot clear the dinner deadline it just set. Replaced with a small `optionalUts`
  Unmarshaler, which json *does* call for a null. Pinned by
  `TestPatchDistinguishesAnAbsentFieldFromAnExplicitNull`.
- 2026-08-27 — Second find from the same test run: the board serialised `tasks`/`tours` as `null`
  when a query returned nil. The queriers already return `[]`, but the handler now coerces too —
  two lines against the failure mode that has broken rendering in this repo three times, once
  taking a dialog's close button with it.
- 2026-08-27 — `MarkPickedUp` is idempotent (pressing Hentet twice at night with a driver still
  talking must not log two custody changes), and cancelling an already-cancelled task answers
  200 while cancelling a *done* one is refused: two operators pressing cancel is a race the desk
  should not think about, cancelling finished work is a misread board. Cancelling a task whose
  people are already aboard is currently allowed — the `ErrAlreadyCollected` precedence PRD 009
  §5 describes lives with the member transition, so it belongs to task 118.
- 2026-08-27 — ✅ **Verified end-to-end against the running stack** (started `docker compose up`
  plus the shared jetstream container). Boot advertises the new live tokens —
  `…,dispatch,…,tour,…` — and creates all six tables. Then, over HTTP: created a red pickup,
  read it back with its timeline, patched priority and deadline (timeline entry `task.updated`
  with `"priority, deadlineUts"`), recorded `pickedup` with unit `bil-2`, had a reasonless cancel
  refused with `{"reason":"angiv en årsag"}`, then cancelled it properly and saw
  `state: cancelled` with the reason on the board. No dispatch-related errors in the log; the
  duplicate-`checkpersonnel` and "unexpected end of JSON input" replay errors are pre-existing.
  One cancelled test task now exists in the dev stream for 2026, deliberately left rather than
  faked away — the stream is the dev stream.
- 2026-08-27 — All criteria met. Moving to done.
