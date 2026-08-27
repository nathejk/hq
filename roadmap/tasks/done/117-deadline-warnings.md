# 117 — deadline warnings and the at-risk filter

**Status:** done
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

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

- [x] Countdown to `skal leveres` on deadline tasks, advancing live
- [x] At-risk flagged for both causes (plan lands late; still queued near the deadline)
- [x] Banner with a filter shortcut, matching the existing pattern
- [x] Summary counts visible without scrolling
- [x] Overdue unvisited stops marked
- [x] Threshold configurable rather than hard-coded

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — `deadlineRisk` returns **why**, not just whether: `late` (the plan lands after the
  deadline, or it has already passed) and `soon` (still unplanned with the deadline inside the
  window). They call for different actions — a late plan needs another car, an unplanned task needs
  a plan — so collapsing them to a boolean would have thrown away the useful half.
- 2026-08-27 — **Finished and cancelled work is never at risk**, however late it was. A red row for
  dinner that was delivered is how a board teaches its operator to ignore red rows, which is the
  failure mode a warning system has to avoid above all others.
- 2026-08-27 — Deadline trouble is pinned above the oldest wait in the queue. A scout who has waited
  an hour is a problem the desk already knows about; dinner that is about to be late is one it does
  not, and the whole point of §5's worked example is knowing at 16:00 rather than at 19:20.
- 2026-08-27 — The banner's shortcut **filters rather than navigates**, and it is a toggle: the same
  click that narrows the board widens it again. A navigation would have left the operator to find
  their way back.
- 2026-08-27 — `DEADLINE_WARNING_MINUTES = 60`, one constant in one place. Chosen to be useful
  rather than correct: it catches the dinner run while there is still time to send a second car,
  without lighting up every delivery entered in the afternoon.
- 2026-08-27 — Overdue *stops* were already marked when the tour card was written (task 113) — a
  planned time in the past with the stop unvisited, and only on a tour that has actually set off,
  because an un-departed plan is not yet a promise. Left as it was rather than reimplemented here.
- 2026-08-27 — ✅ Verified: the risk rules are covered by the vitest suite added in task 116 (both
  causes, both exclusions, the threshold boundary), 173 tests green; `vue-tsc` at the 109 baseline
  and the dev server compiles the view.
- 2026-08-27 — All criteria met. Moving to done.
