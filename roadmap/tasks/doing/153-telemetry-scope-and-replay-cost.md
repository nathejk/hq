# 153 — decide hq's telemetry scope and measure replay cost

**Status:** doing
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (with knj)
**Started:** 2026-09-09
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

- [ ] Row count and growth rate measured against real traffic — **blocked: needs an event**
- [ ] Api boot time measured with and without the telemetry projection — **baseline only, see below**
- [x] Patrol track endpoint p95 and payload size measured on a worst-realistic-case patrol
- [x] Scope decision made (year-scoped vs. all history) and implemented
- [ ] Decision and numbers recorded in PRD 011 §8

## Decision: year-scoped consumer subject (implemented)

`track`'s consumer now subscribes to `TELEMETRY.{currentYear}.track.*.reported` instead of
`TELEMETRY.*.track.*.reported`. `currentYear` already existed in `cmd/api/main.go` and is
already passed to the payment, order and patruljenumber projections, so this cost one
parameter rather than an architecture — which is why this option was the recommended default.

What it buys: boot-time replay is bounded by the current event instead of by every race the
event has ever run. This is the only stream HQ consumes that grows with wall time ×
participants and is retained indefinitely, so it is the only one where replay cost compounds
year on year.

What it costs, both deliberate:

- **Last year's tracks are not in HQ's read model.** Accepted. HQ is a race-support tool;
  nothing in it asks where a patrol walked in a previous event. It is also a privacy
  improvement rather than a regret — position history is the most personal data HQ holds, and
  not carrying it forward is consistent with `roadmap/api/telemetry-erasure.md`.
- **The year is fixed for the process lifetime**, because a consumer declares its subjects once
  at construction. A new year needs a restart, which happens on every deploy anyway and cannot
  bite mid-race: the boundary is 1 January.

An empty year falls back to the wildcard rather than building `TELEMETRY..track.*.reported`.
That matters because the one failure mode this change could introduce — a subject that matches
nothing — is indistinguishable from an event where nobody has the app open. For the same
reason `Consumes()` now logs the subject it subscribes to, and a test asserts the scoped
pattern matches a **real** captured subject (and does not match another year's).

## Measurements, 2026-09-09 (dev)

### Patrol track endpoint, worst-realistic recording — done

Synthetic 30-hour unbroken recording at 30 s sampling: **3,600 points for one member**, which
is the ceiling this PRD sizes against, inserted into `track_point` and removed afterwards.
`GET /api/telemetry/person/{id}/track`, measured inside the api container:

| request | time | payload |
| --- | --- | --- |
| default budget | 14–23 ms (12 runs, p95 ≈ 23 ms, steady-state ≈ 15 ms) | 223 KB |
| `maxPoints=300` | 17 ms | 34 KB |
| `maxPoints=1000` | 18 ms | 111 KB |
| `maxPoints=5000` | 20 ms | 223 KB |

Against the PRD's targets (p95 < 300 ms, reduced payload < ~500 KB): **an order of magnitude
inside both**, for one member. A six-member patrol is ~6× the work in the worst case, so ~90–140
ms and ~1.3 MB unreduced — still inside the time target, but the payload only meets the 500 KB
target with a budget applied, which is what `maxPoints` is for.

Note what this does *not* prove: it is a single-client measurement on an idle dev database with
~1.2k real rows, so it isolates query + reduction cost, not behaviour under concurrent load.

### Boot time — baseline only

Api ready (health endpoint answering) **4 s** after `docker compose restart api`, against the
dev stream (~1.2k track points plus all NATHEJK events).

This is a baseline and a method, not the measurement the criterion asks for. "With and without
the telemetry projection" needs a stream with a realistic telemetry backlog to be meaningful,
and a dev stream with 1.2k points cannot distinguish the two. Re-take it during or after the
event, the same way: restart, poll `/api/v1/healthcheck`, compare.

### Row count and growth — blocked

`track_point` holds 1,202 rows, all from development testing, so there is no growth rate to
measure. Also worth recording for whoever picks this up: the **JetStream monitoring port is not
enabled** on the local `nathejk-jetstream-1` (8222 is mapped but nothing serves it) and the
container has no `nats` CLI, so stream message counts and bytes cannot be read from the dev
environment at all. Enabling `-m 8222` on the nats-server would make that possible, and is
probably worth doing before the event — stream size is the number this task actually turns on.

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-09 15:57 — Picked up (previous owner gone).
- 2026-09-09 16:20 — Implemented the year-scoped subject: `track.New` takes a year,
  `subjectReported` builds the pattern, empty falls back to the wildcard, `Consumes()` logs
  what it subscribed to. Three tests added, including one that matches the scoped pattern
  against a real captured subject — a pattern compared only to itself would pass while HQ
  received nothing.
- 2026-09-09 16:35 — Measured the track endpoint against a synthetic 3,600-point recording:
  p95 ≈ 23 ms, 223 KB unreduced, well inside the PRD's targets. Synthetic rows deleted.
- 2026-09-09 16:40 — Boot-time baseline 4 s; the with/without comparison is not meaningful on a
  dev stream, and stream size cannot be read locally because the nats monitoring port is not
  enabled. Both remain open, and are blocked on real traffic rather than on work.
- 2026-09-09 16:45 — Staying in `doing/`: the decision is made and implemented, but two criteria
  need the event. Not closing a task whose measurements have not been taken.
