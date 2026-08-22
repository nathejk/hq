# 087 — shelter projection for placering

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] `shelter` package with `table.go`, `consumer.go`, `querier.go`, `table.sql`
- [x] Row keyed `(year, memberId)`; holds placering and when it was set
- [x] `handover.completed` deletes the row
- [x] Replaying the same events twice produces identical rows
- [x] `DistinctPlacements` returns placeringer with counts, most-used first, year-scoped
- [x] Consumer added to the `projections` slice in `cmd/api/main.go`
- [x] Consumer tests covering: accept, place, re-place, handover, unknown member, replay

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 11:20 — Picked up. Correction to the task's own premise, found while doing 086:
  `spejderstatus/table.go` already has a `schemaMigrations` hook — idempotent ALTERs run at
  boot — added precisely because `CREATE TABLE IF NOT EXISTS` skips existing tables. So
  "the schema is still free to change because no race has run" is the weaker argument; the
  `shelter` table gets the same hook from the start, and a column added in November works
  without anybody dropping a database. Plan: goqu throughout, following spejderstatus and
  sos rather than lok's `fmt.Sprintf("%q")` style.
- 2026-08-23 11:35 — Table, consumer and querier written. Event bodies are imported from
  `spejderstatus` rather than redeclared: one definition of the wire format, and when that
  package is lifted to shared-go the import moves with it.
- 2026-08-23 11:45 — **The bug this projection was most likely to have, found while writing
  the upsert.** Replay re-delivers the acceptance on every boot, and an acceptance need not
  carry a placering. With a plain `VALUES(placement)` update, a scout moved into Telt 4 at
  01:10 would be back to nowhere after the next API restart — leaving the crew hunting for a
  child the screen says is nowhere. Fixed with a conditional update,
  `IF(VALUES(placement) = '', placement, VALUES(placement))`, and pinned by
  `TestEmptyPlaceringDoesNotWipeAStoredOne`.
- 2026-08-23 11:50 — `acceptedAt` deliberately left out of the update list for the same class
  of reason: it records when the shelter took charge of a child, and replay must not turn
  "in our care since 00:42" into "since the last restart". Test added.
- 2026-08-23 11:55 — Decision: `placedAt` stays NULL for an arrival with no placering, rather
  than being stamped with the acceptance time. "Arrived 00:42, not yet placed" is the crew's
  next job; stamping it would make every arrival look dealt with.
- 2026-08-23 12:00 — Decision: `shelter.placed` upserts rather than updates, so an event for a
  member whose acceptance is missing (truncated history, or replay ordering) still lands. A
  scout with a placering and no recorded arrival is a better read model than no scout at all:
  the crew can find them, which is the entire purpose of the table.
- 2026-08-23 12:05 — Wired into the `projections` slice in `main.go` (not the mux directly —
  outside that slice it emits no live signal and the screen would look live and never update).
  Empty `schemaMigrations` hook included from the start, per the correction logged above.
- 2026-08-23 12:10 — ✅ All criteria met. 15 tests in the package, all green; `go build ./...`
  and the full `go test ./...` green. Honest scope note: the two queries are asserted at
  statement level (ordering, the empty-placering exclusion, year scoping, the empty-set
  short-circuit), not against rows in a MySQL server — the package has no DB-backed test
  harness and adding one is a larger decision than this task. The projection's *writes* are
  covered properly, since those are emitted SQL.
- 2026-08-23 12:12 — Moving to done.
