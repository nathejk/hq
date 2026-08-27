# 116 — capacity strip, unit readiness and the queued estimate

**Status:** done
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §6 ("Answering when?"), §7 (capacity strip), §8 (the estimate). Depends on task 115.

**Capacity strip** above both panes: each dispatchable unit with its vehicle, driver, duty
window and whether it is on duty now; when none are, *"Næste enhed på vagt 22:00"*. A unit with
no vehicle or nobody in it is shown **not ready**, saying which is missing. More than one
vehicle in a unit is a configuration mistake — **flagged, not forbidden**.

**Estimate** for a queued task with no tour, visibly marked *anslået*:

```
estimate = max(now, tidligst, next unit on duty) + allowance(kind)
```

`allowance` is a small configured table (pickup 30 min, transport 20, …), **one allowance for
every vehicle** — per-vehicle speed is deliberately ignored (§8, open question 10). A tour's
planned stop time **beats** the estimate on screen. Every task also always shows how long it has
waited, from `oprettet` — the number that needs no model and is never wrong.

## Acceptance Criteria

- [x] Capacity strip with vehicle, driver, window, on-duty state and next-on-duty fallback
- [x] Not-ready units say what is missing; multiple vehicles flagged
- [x] Queued estimate computed as specified and labelled *anslået*
- [x] Planned time shown in preference to an estimate wherever both exist
- [x] Waiting time shown on every task, advancing live
- [x] With no duty data the estimate degrades to `now + allowance`, not to nonsense

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — The strip is `components/DispatchCapacityStrip.vue`; the arithmetic is pure functions
  in `composables/dispatch.ts` (`unitsOnDuty`, `nextDutyStart`, `estimateFor`, `unitReadiness`), so
  it could be tested without mounting anything — which is how the estimate got 23 tests rather than
  an eyeball.
- 2026-08-27 — **The judgement call in the estimate: a unit already driving counts as capacity.**
  The formula in §8 reads `max(now, tidligst, next unit on duty) + allowance`, and taken literally it
  would push every estimate out to the *next* shift even while a car is out working — pessimistic to
  the point of useless at 22.30 with two cars on the road. So the next-on-duty term applies only
  when nobody is on duty now, which is the case it was written for ("No unit on duty" in §5).
- 2026-08-27 — With no roster at all it degrades to `now + allowance`, deliberately: a stale roster
  makes the number optimistic, and a missing one makes it merely crude. That is the mitigation §8
  asks for, and it is a test.
- 2026-08-27 — Every estimate is prefixed *anslået* and sits **beside** the waited-for time, never
  instead of it. The plan beats the estimate wherever a tour exists, so `estimateFor` is only ever
  consulted for `queued` tasks.
- 2026-08-27 — `unitsOnDuty` is half-open at the end, matching the server's `Duty.Covers`. Two
  consecutive windows would otherwise both claim the same minute and one unit would read as on duty
  twice — which looks like a configuration error that is not there.
- 2026-08-27 — The strip distinguishes three things that would otherwise all be an empty box: no
  unit *configured* (fix it on Organisation), no unit *on duty* ("Næste enhed på vagt 22.00"), and a
  unit that is not *ready* — naming which of car or crew is missing, so it becomes something
  somebody can act on. More than one vehicle in a unit is flagged and not forbidden.
- 2026-08-27 — ✅ Verified: 23 new vitest cases for the arithmetic (all green; the whole suite is
  173 passing), `vue-tsc` at the 109 baseline, and the dev server compiles the view and the strip.
- 2026-08-27 — All criteria met. Moving to done.
