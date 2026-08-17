# 082 — Confirm the waiting alarm threshold with organizers

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

> **Closed without implementing.** The alarm itself is deferred to the PRD covering the
> dispatch dashboard, so there is no threshold to agree yet. Kept rather than deleted
> because the argument for *why* it could not sensibly be answered now is the useful part.

## Description

From **PRD 006** §11. A product decision, not code.

How long may a member be `waiting` — left the route, awaiting collection — before the
nødtelefon dashboard warns?

Their patrol is **blocked for the whole duration**, so this is the one number operators will
actually feel. Too low and the dashboard cries wolf all night; too high and a patrol sits by
a road for an hour with nobody noticing.

Questions to settle:

- A single fixed threshold, or one per case severity (green/yellow/red)?
- Is it a configuration value, and if so does it belong with the year config alongside the
  3-member minimum (task 074)?
- Does the warning escalate, or is it binary?

## Acceptance Criteria

- [x] ~~Threshold agreed with organizers~~ — not yet answerable, see log
- [x] ~~Fixed vs per-severity decided~~ — deferred with the alarm
- [x] ~~Home for the value decided~~ — deferred with the alarm
- [x] PRD 006 §11 updated: moved from Still open to Decisions
- [x] Task 078 updated — the alarm is removed from its scope, the counter remains

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
- 2026-08-17 — **Closed by product decision: the `waiting` alarm is deferred to the PRD
  covering the dispatch dashboard.**

  The reason is the one this task already recorded as a warning against tuning:
  **nothing in this feature resolves a `waiting` member.** The car and shelter interfaces
  do not exist, so no event moves anybody out of `waiting` except a manual correction —
  which means an alarm built here would fire for every member and stay firing, and any
  threshold agreed against that behaviour would be a threshold tuned against missing
  software rather than against what organisers consider acceptable in the field.

  Deferring it is therefore not postponement for its own sake: the alarm only becomes a
  meaningful signal once something can act on it, and that something is dispatch.
- 2026-08-17 — Nothing is lost by waiting. `GET /api/member/care` already returns
  `oldestWaitingAt`, so the fact the alarm needs is projected and available the day the
  dashboard wants it — no backend change, no migration.
- 2026-08-17 — Task 078 amended: the in-our-care counter stays, the alarm half is removed.
  PRD 006 §6, §8 and §11 updated to match. Moving to done as closed-unimplemented.
