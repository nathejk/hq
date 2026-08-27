# 111 — tour API — create, stops, underway, stop visited, complete

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §8. Depends on tasks 109 and 110.

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/dispatch/tour` | create a tour for a unit |
| PATCH | `/api/dispatch/tour/:id` | departure, unit, notes |
| PUT | `/api/dispatch/tour/:id/stops` | set the ordered stops and their tasks |
| POST | `/api/dispatch/tour/:id/underway` | tour has set off |
| POST | `/api/dispatch/tour/:id/stop/:stopId/visited` | stop reached |
| POST | `/api/dispatch/tour/:id/completed` | tour done |
| POST | `/api/dispatch/tour/:id/cancelled` | cancel, with reason |

`PUT …/stops` sets the whole ordered list in one call — a reorder is one operator intent, and
per-stop endpoints would make a half-applied reorder possible (`/api/sections/sorted` has the
same shape).

Rules to enforce server-side:
- Visited stops are fixed; a payload that reorders or removes one is rejected.
- A task's unload stop may not be ordered before its load stop.
- Planned stop times are **derived** from departure + per-leg allowance, with any stop
  overridable by hand; an overridden stop is marked, and later stops re-derive from it.
- A tour with no remaining unvisited stops may complete.
- Warn (do not refuse) when unvisited pickups exceed the vehicle's `seatCount`, and when a tour
  is planned outside its unit's duty window.

State transitions stay **first-class endpoints** — the driver app will call them (§8).

## Acceptance Criteria

- [ ] Seven endpoints, routed, with OpenAPI annotations
- [ ] Whole-list stop PUT; visited stops immutable; load-before-unload enforced
- [ ] Stop times derived from departure + allowance; per-stop override respected and marked
- [ ] Seat and duty-window overruns returned as warnings on the response, not errors
- [ ] Marking a stop visited progresses or completes the tasks at it
- [ ] Handler tests for each transition plus the rejected reorder cases

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
