# 078 — I vores varetægt counter and the waiting alarm

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §6 and §7. **The organisers' go-home number.**

In the `vue/src/views/SosListView.vue` header, a permanent **I vores varetægt** counter:

- the total from `InOurCare()` — `waiting` + `transit` + `sheltered`
- a breakdown per status
- a warning state when any member has been `waiting` past the threshold

Visible without opening a case, because it is an event-wide number rather than a per-case
one: this is the count that has to reach zero before anybody goes home.

Fed by `GET /api/member/care` (task 068).

## Notes

- **`dependsOn: ['spejder']`** — the entity *type*, not an instance. A member whose id the
  client has never seen must still move the number, and a count has no id at all (PRD 004 §8).
- **Timeliness is load-bearing here in a way it is not elsewhere.** A count of the people we
  are responsible for that is quietly a minute out of date is worse than no count. This is
  the reason live updates exist in this feature; adopt `useLiveResource` in the first commit,
  not a later pass.
- **A wrong number is worse than no number.** Until PRD 005's boot gate ships, a post-restart
  window can serve a partially rebuilt read model. The honest interim behaviour is to show
  that the screen cannot reach the server rather than to render a figure — reuse the existing
  connection-state indicator (task 026) rather than inventing a second signal.
- The `waiting` alarm: a waiting member **blocks their entire patrol**, which is why this is
  the one state worth an alarm. The endpoint returns the oldest `waiting` timestamp; the view
  applies the threshold. The threshold itself is task 082.
- **Expect the alarm to fire for everybody at first.** A member put into `waiting` has no
  automatic way out until the car and shelter interfaces ship, so `InOurCare()` will not drain
  on its own and the override is the interim path. Do not tune the threshold against interim
  manual bookkeeping.
- Where the counter lives is still formally open (list view, `/api/home` dashboard, or both).
  The list view is the proposal and this task; revisit only if operators ask.

## Acceptance Criteria

- [ ] Counter in the `SosListView` header with total and per-status breakdown
- [ ] Warning state when the oldest `waiting` exceeds the threshold
- [ ] `dependsOn: ['spejder']`; no dev-console dependency warnings
- [ ] Updates live in a second tab when a member's status changes
- [ ] Shows a cannot-reach-server state rather than a number when the API is unavailable
- [ ] Reads zero cleanly when nobody is in care
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 068.
