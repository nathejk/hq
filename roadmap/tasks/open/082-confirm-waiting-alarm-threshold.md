# 082 — Confirm the waiting alarm threshold with organizers

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

## Notes

- **Do not tune it against the first event's data.** Until the car and shelter interfaces
  ship, a member put into `waiting` has no automatic way out — the override is the interim
  path — so `InOurCare()` will not drain on its own and *everybody* will eventually breach
  any threshold. The number has to come from what organisers consider acceptable in the
  field, not from observed data that reflects missing software.
- Related but separate: whether the alarm should be able to **request a pickup** is deferred
  to the car PRD (PRD 006 §11). Worth asking the same conversation, since it is the only piece
  of car dispatch that might belong in the nødtelefon interface.
- Task 078 implements the alarm; a placeholder threshold is fine to build against, so this
  does not block it.

## Acceptance Criteria

- [ ] Threshold agreed with organizers and recorded in the log with the reasoning
- [ ] Fixed vs per-severity decided
- [ ] Home for the value decided (and aligned with task 074 if it is year config)
- [ ] PRD 006 §11 updated: moved from Still open to Decisions
- [ ] Task 078 updated to use the agreed value

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
