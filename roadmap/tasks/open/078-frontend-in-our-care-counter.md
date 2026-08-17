# 078 — I vores varetægt counter

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

Visible without opening a case, because it is an event-wide number rather than a per-case
one: this is the count that has to reach zero before anybody goes home.

Fed by `GET /api/member/care` (task 068), which is done and returns exactly this shape.

**No `waiting` alarm** (decided 2026-08-17, task 082). It is deferred to the PRD covering
the dispatch dashboard, because nothing in this feature resolves a `waiting` member — the
car and shelter interfaces do not exist — so an alarm here would fire for every member and
stay firing. The endpoint already returns `oldestWaitingAt`, so the fact is available the
day the dashboard wants it; **do not render a warning state off it here.**

## Notes

- **`dependsOn: ['spejder']`** — the entity *type*, not an instance. A member whose id the
  client has never seen must still move the number, and a count has no id at all (PRD 004 §8).
  `spejder` is confirmed present in the advertised token set at `/api/stream`, so the dev-only
  dependency validation will accept it.
- **Timeliness is load-bearing here in a way it is not elsewhere.** A count of the people we
  are responsible for that is quietly a minute out of date is worse than no count. This is
  the reason live updates exist in this feature; adopt `useLiveResource` in the first commit,
  not a later pass.
- **A wrong number is worse than no number.** Until PRD 005's boot gate ships, a post-restart
  window can serve a partially rebuilt read model. The honest interim behaviour is to show
  that the screen cannot reach the server rather than to render a figure — reuse the existing
  connection-state indicator (task 026) rather than inventing a second signal.
- The breakdown includes in-care statuses at zero (the endpoint guarantees it), so the
  display must not collapse them: "transit: 0" is information — no car is carrying anybody.
- Expect the number to sit at 0 in normal dev data. Task 072's verification produced and then
  cleared real `waiting` members, so the mechanism is proven end to end; to see it non-zero,
  drive `PUT /api/member/:id/waiting` from a case.

## Acceptance Criteria

- [ ] Counter in the `SosListView` header with total and per-status breakdown
- [ ] No warning state, no threshold, no alarm
- [ ] `dependsOn: ['spejder']`; no dev-console dependency warnings
- [ ] Updates live in a second tab when a member's status changes
- [ ] Shows a cannot-reach-server state rather than a number when the API is unavailable
- [ ] Reads zero cleanly when nobody is in care
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 068.
- 2026-08-17 — Amended before pickup: the `waiting` alarm is removed from scope and
  deferred to the dispatch dashboard PRD (task 082). The counter is unaffected — it was
  always the half that works without the car and shelter interfaces existing.
