# 087 — shelter projection for placering

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 007 records **where in Hønsegården a scout has been bedded down**. New local table
package `go/nathejk/table/shelter/` following the house layout (`table.go`, `consumer.go`,
`querier.go`, `table.sql`).

Deliberately **not** a column on `spejderstatus`:

1. `spejderstatus` owns status and team membership; a bed is a fact about the shelter.
2. `spejderstatus` is queued for lifting to shared-go verbatim (task 083) — an hq-specific
   column makes that lift a rewrite.
3. `CREATE TABLE IF NOT EXISTS` never alters an existing table, so a new column would be
   silently absent from every existing database. A new table would not.

Consumes:

- `NATHEJK:*.spejder.*.shelter.accepted` — upsert the row, with the placering if the event
  carries one (task 088)
- `NATHEJK:*.spejder.*.shelter.placed` — set the placering
- `NATHEJK:*.spejder.*.handover.completed` — delete the row; the scout is no longer in the
  shelter's care and their bed is free

Idempotent under replay (projections rebuild from JetStream on every API boot), and
tolerant of events for members it has never seen and of arrival in any order — same
contract as the `spejderstatus` consumer.

Also add `DistinctPlacements(ctx, year) ([]Placement, error)`, the placeringer in use this
year with a count, most-used first. This is what makes the zone vocabulary define itself at
race start with no configuration (PRD 007 §6) — there is no zone entity by design.

**Wire the consumer into the `projections` slice in `cmd/api/main.go`**, not straight into
the mux. Outside that slice it is wrapped by nothing, emits no live signal, and the screen
would look live and never update.

The schema is still free to change: no race has run, so there is nothing to preserve. That
ends at the first race night — get the columns right now.

## Acceptance Criteria

- [ ] `shelter` package with `table.go`, `consumer.go`, `querier.go`, `table.sql`
- [ ] Row keyed `(year, memberId)`; holds placering and when it was set
- [ ] `handover.completed` deletes the row
- [ ] Replaying the same events twice produces identical rows
- [ ] `DistinctPlacements` returns placeringer with counts, most-used first, year-scoped
- [ ] Consumer added to the `projections` slice in `cmd/api/main.go`
- [ ] Consumer tests covering: accept, place, re-place, handover, unknown member, replay

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
