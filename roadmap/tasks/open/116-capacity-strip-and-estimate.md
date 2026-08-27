# 116 — capacity strip, unit readiness and the queued estimate

**Status:** open
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Capacity strip with vehicle, driver, window, on-duty state and next-on-duty fallback
- [ ] Not-ready units say what is missing; multiple vehicles flagged
- [ ] Queued estimate computed as specified and labelled *anslået*
- [ ] Planned time shown in preference to an estimate wherever both exist
- [ ] Waiting time shown on every task, advancing live
- [ ] With no duty data the estimate degrades to `now + allowance`, not to nonsense

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
