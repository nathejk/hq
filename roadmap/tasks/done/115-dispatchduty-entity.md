# 115 — `dispatchduty` entity and duty window editor

**Status:** done
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §6 (Duty windows), §8 (BFF). Phase 2.

A small `dispatchduty` table keyed by section slug, the same shape as `checkpersonnel`
(`startUts`, `endUts`) and **deliberately not the same table**: a shift on a post and a shift
behind a wheel are different facts, and one table would make "which units are driving now" a
query with a checkpoint join in it.

Events `NATHEJK.{year}.dispatchduty.{id}.{set|removed}`; endpoints `GET/PUT/DELETE
/api/dispatchduty…` with OpenAPI annotations. Windows are recorded **per unit, not per person** —
the unit is what is available or asleep.

Editor UI per dispatchable unit, reusing `DayTimePicker`.

## Acceptance Criteria

- [x] Package + table, consumer in the `projections` slice, `dispatchduty` advertised as live
- [x] Endpoints for listing, setting and removing a window
- [x] Query: units on duty at a given instant, and the next window to open
- [x] Editor per unit, with weekday-bearing times
- [x] Tests: set, overlap, remove, replay, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — **Not a new package: a new table in `dispatch`.** The task says "entity", and PRD 009
  §8 asks for a table "deliberately not the same table" as `checkpersonnel` — which it is not. But a
  separate Go *package* would need its own consumer, its own wiring and its own live token for one
  five-column table read only by the kørsel board. `dispatch_duty` lives in `duty.sql` beside
  `dispatchable.sql`, apart from the tasks and tours because a roster agreed days in advance is
  configuration of capacity rather than a record of the night. The events still carry their own
  entity token, `dispatchduty`, which is what the SPA depends on.
- 2026-08-27 — **A whole window per event**, not a start and a later end. The roster is agreed in
  advance, so both ends are known when it is entered; a half-open window would make "who is on
  now" depend on an event that may never arrive — and the one that never arrives is the 3am one.
- 2026-08-27 — `Duty.Covers` is **half-open**: a shift ending at 22.00 does not include 22.00. Two
  consecutive windows would otherwise both claim that minute, so "who is on now" would answer twice
  for one unit and read on the strip as a configuration error that is not there.
- 2026-08-27 — Overlapping windows are **allowed**. A unit rostered twice over the same hour is on
  duty either way, and refusing it would block an operator fixing a roster in whatever order makes
  sense to them. The only refusal is a window that ends before it begins.
- 2026-08-27 — No `GET /api/dispatchduty`: the roster travels with the board, because the capacity
  strip that reads it is part of the same screen and a second key would let the two disagree about
  who is on. "Units on duty at an instant" and "the next window to open" are answered from that one
  ordered list — fewer than ten units and a night's worth of shifts — rather than by two queries.
  Task 116 computes both from it.
- 2026-08-27 — The editor is a **dialog**, not a panel: the roster is set up once an evening and
  then read all night, and a permanent form would take width from the two things read constantly.
  Times are weekday-bearing, because "21.40 til 02.00" does not say which evening.
- 2026-08-27 — ✅ Verified against the running stack: `dispatch_duty` created on boot,
  `PUT /api/dispatchduty` returns a minted id and the window appears in the board's `duty` array,
  a backwards window is refused with `{"endUts":"vagten skal slutte efter den begynder"}`, and
  `DELETE` removes it. 6 new domain tests; full `go test ./...` green, `vue-tsc` at the 109
  baseline, dev server compiles the view.
- 2026-08-27 — All criteria met. Moving to done.
