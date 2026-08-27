# 109 — `dispatch` entity — tasks, tours, stops, timeline

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §8 (BFF, Data/storage). New hq-owned package `go/nathejk/table/dispatch/` with the
repo layout (`table.sql`, `table.go`, `messages.go`, `consumer.go`, `querier.go`, `filter.go`;
`commands.go` in tasks 110/111), owning **both** tasks and tours — one aggregate, since a stop
is meaningless without its tour and a task's state derives from its stops.

Tables: `dispatch_task`, `dispatch_tour`, `dispatch_stop`, `dispatch_activity` (the timeline).

Events:
- `NATHEJK.{year}.dispatch.{taskId}.{created|updated|planned|unplanned|underway|pickedup|completed|cancelled}`
- `NATHEJK.{year}.tour.{tourId}.{created|stops.changed|underway|stop.visited|completed|cancelled}`

Details that must be right now, because `CREATE TABLE IF NOT EXISTS` never alters a table
(§8 Data/storage):
- Places are **type + reference + label** (`kind`, `refId`, `label`), not a foreign key.
- A task's link to a tour lives on the **stop**, not on the task (a task may occupy two stops).
- Priority is a dispatch-local three-value type `green|yellow|red` — deliberately not an import
  of the `sos` package (§8 "Priority: mirrored from SOS").
- Times: `createdUts` (the waiting clock), `notBeforeUts`, `deadlineUts`.
- Task state `queued|planned|underway|done|cancelled`, plus `pickedUpUts` for pickups.
- Unplanning must not reset `createdUts`.

**Wire the consumer into the `projections` slice in `cmd/api/main.go`** — outside it the board
would look live and never update. New live entity tokens: `dispatch`, `tour`.

## Acceptance Criteria

- [x] Package with the files above; subjects as specified
- [x] All four tables created on boot, with the `schemaMigrations` hook from the start
- [x] Queries: board (queued tasks, tours with ordered stops), task by id with timeline
- [x] Replaying the same events twice produces identical rows
- [x] Dropping a task from a tour returns it to `queued` with `createdUts` untouched
- [x] Consumer in the `projections` slice; `dispatch` and `tour` advertised as live entities
- [x] Consumer tests: create, plan, reorder, visit, unplan, cancel, replay, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — Picked up. Plan: `table.sql`, `types.go`, `messages.go`, `consumer.go`,
  `querier.go`, `filter.go`. No `commands.go` beyond the section flag — publishing is tasks
  110/111, and taking a publisher for events nothing can yet emit would be a field nobody could
  explain (the precedent is task 099).
- 2026-08-27 — **Amended the PRD: a fifth table, `dispatch_stop_task`.** PRD 009 §8 lists four,
  but "the tasks actioned at a stop" is a many-to-many the moment a task occupies both a load
  and an unload stop — which the PRD itself argues for ("a `tourId` column on the task would be
  a lie half the time"). With the pairs in a JSON column, "which stops does this task sit on",
  the question a task's state is derived from, becomes an unindexed `JSON_CONTAINS` scan of
  every stop of every tour. §8's list now says so.
- 2026-08-27 — **Times are stored as unix seconds (`…Uts`), not DATETIME**, unlike `sos` and
  `shelter`. Every number on this screen is arithmetic — waited-for, time-to-deadline, departure
  plus a leg allowance — and seconds keep the timezone question out of each step.
  `checkpersonnel` already stores shifts this way. Nullable times are nullable columns and
  pointers in Go: 0 is 1970, and a zero deadline would render as a task that was late in 1970.
- 2026-08-27 — **A task's state comes from its own events, never from inspecting its stops.**
  Hence `dispatch.{id}.planned|unplanned|underway|completed` alongside the tour's
  `stops.changed`. The redundancy is the point: the tour event is the plan, the task event is
  the consequence for one task, the timeline reads correctly for both, and the projection needs
  no derivation order.
- 2026-08-27 — **The bug this projection could most easily have had, found while writing the
  upsert.** Replay re-delivers `created` on every boot, *after* the transitions that came later.
  With `state` in the update list, every finished task would have quietly returned to the queue
  on restart — and the board would show a night nobody had worked. `state`, `doneUts`,
  `pickedUpUts`, `cancelledUts` and `cancelReason` are excluded from the replayed insert, pinned
  by `TestReplayingCreatedDoesNotResetState`. The same class of trap task 099 hit with note text.
- 2026-08-27 — Named the state constants `TaskStateQueued` / `TourStatePlanned` rather than
  `TaskQueued` / `TourPlanned`: the short names belong to the event payloads, where `TaskPlanned`
  is the event that says a task went into a tour. The compiler found this collision, and the
  events won.
- 2026-08-27 — `stops.changed` rebuilds the list delete-then-insert rather than diffing. The
  event carries each stop's `visitedUts`, so a rebuild loses nothing, and it is the only
  approach that cannot leave behind a stop the new plan does not mention. `sortOrder` is written
  from the slice index, so the ordering has exactly one source of truth.
- 2026-08-27 — Caught in review of my own `attachStops`: it held `*TourStop` pointers into a
  slice it was still appending to, so a reallocation would have written the stop's tasks into a
  copy nobody reads — stops would have rendered with no tasks, intermittently and only on tours
  long enough to grow the slice. Now a (tour, index) pair.
- 2026-08-27 — ✅ All criteria. 20 tests in the package, full `go test ./...` green. The DDL was
  executed against a real MariaDB 10.8 (a scratch database in a container from another project,
  dropped afterwards) — all five tables create, and a representative goqu-generated upsert runs.
  Not verified end-to-end: the hq stack is not running locally, so nothing has replayed a real
  stream yet; that arrives with task 110, which can actually publish.
- 2026-08-27 — Moving to done.
