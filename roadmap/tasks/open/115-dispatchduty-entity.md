# 115 — `dispatchduty` entity and duty window editor

**Status:** open
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Package + table, consumer in the `projections` slice, `dispatchduty` advertised as live
- [ ] Endpoints for listing, setting and removing a window
- [ ] Query: units on duty at a given instant, and the next window to open
- [ ] Editor per unit, with weekday-bearing times
- [ ] Tests: set, overlap, remove, replay, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
