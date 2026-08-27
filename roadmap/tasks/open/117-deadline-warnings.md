# 117 — deadline warnings and the at-risk filter

**Status:** open
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §6, §7. Phase 2. Dinner at 19:00 must be visible as at risk while there is still time
to act.

- Deadline tasks show **time until `skal leveres`**, and are flagged when the plan lands after
  the deadline, or when they are still queued inside a configurable window of it.
- **Deadline banner** reusing the checkgroup-teams-dialog pattern: a `Message` when a deadline
  is inside the next hour and the task is unplanned or planned late, with a shortcut that
  filters the board to it. Deliberately the same vocabulary — two race-night screens should not
  invent two ways to say "you are about to be late".
- The board answers at a glance: how many tasks are unplanned, how long the oldest has waited,
  how many tours are out, which deadlines are at risk.
- A planned stop time in the past with the stop unvisited is shown **overdue**.

## Acceptance Criteria

- [ ] Countdown to `skal leveres` on deadline tasks, advancing live
- [ ] At-risk flagged for both causes (plan lands late; still queued near the deadline)
- [ ] Banner with a filter shortcut, matching the existing pattern
- [ ] Summary counts visible without scrolling
- [ ] Overdue unvisited stops marked
- [ ] Threshold configurable rather than hard-coded

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
