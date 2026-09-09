# 153 — decide hq's telemetry scope and measure replay cost

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §8. To be settled **before** the event, not after. Depends on task 141 being deployed
against real traffic.

Unlike every other stream hq consumes, `TELEMETRY` grows with wall time × participants and is
**retained indefinitely**. The ceiling is ~3,600 points per person per 30-hour race (sparse
recording will make the reality less, but it accumulates every year). hq replays every
projection from the stream on **every api restart**, which makes boot time the plausible
breaking point.

Decide, on measurement rather than instinct:

- **Year-scoped consumer subject** (`TELEMETRY.{currentYear}.track.*.reported`) so hq replays
  only the current event. Cheapest by far — the consumer already declares its subjects, so this
  is a one-line change rather than an architecture. **Recommended default.** Cost: last year's
  tracks are not in hq's read model (acceptable? that is the decision).
- A downsampled point table alongside the raw one.
- A consumer that does not replay from the beginning.

Measure and record in PRD 011: rows in `track_point`, api boot time with and without the
telemetry projection, and the p95 of the patrol track endpoint (target < 300 ms, reduced payload
< ~500 KB) against a well-recorded 30-hour six-member patrol.

## Acceptance Criteria

- [ ] Row count and growth rate measured against real traffic
- [ ] Api boot time measured with and without the telemetry projection
- [ ] Patrol track endpoint p95 and payload size measured on a worst-realistic-case patrol
- [ ] Scope decision made (year-scoped vs. all history) and implemented
- [ ] Decision and numbers recorded in PRD 011 §8

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
