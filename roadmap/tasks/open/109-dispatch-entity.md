# 109 — `dispatch` entity — tasks, tours, stops, timeline

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Package with the files above; subjects as specified
- [ ] All four tables created on boot, with the `schemaMigrations` hook from the start
- [ ] Queries: board (queued tasks, tours with ordered stops), task by id with timeline
- [ ] Replaying the same events twice produces identical rows
- [ ] Dropping a task from a tour returns it to `queued` with `createdUts` untouched
- [ ] Consumer in the `projections` slice; `dispatch` and `tour` advertised as live entities
- [ ] Consumer tests: create, plan, reorder, visit, unplan, cancel, replay, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
