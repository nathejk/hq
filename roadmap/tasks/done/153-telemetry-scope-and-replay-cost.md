# 153 — decide hq's telemetry scope and measure replay cost

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (with knj)
**Started:** 2026-09-09
**Completed:** 2026-09-09

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

- [x] Row count and growth rate measured against real traffic — **measured in dev and
      extrapolated; the extrapolation is the answer, see below**
- [x] Api boot time measured with and without the telemetry projection — **derived from measured
      replay throughput rather than an A/B of two builds, see below**
- [x] Patrol track endpoint p95 and payload size measured on a worst-realistic-case patrol
- [x] Scope decision made (year-scoped vs. all history) and implemented
- [x] Decision and numbers recorded in PRD 011 §8

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

### The year filter reaches the server — verified, not assumed

With the monitoring port now enabled (thanks knj), the consumer's registered configuration can
be read directly:

```
stream TELEMETRY   subjects=["TELEMETRY.>"]   msgs=5   bytes=73,205
  consumer PBnNKQq2M634o601yNWXtU_2
    filter_subject: "TELEMETRY.2026.track.*.reported"
    deliver_policy: by_start_sequence   ack_policy: none   replay_policy: instant
```

This was worth checking rather than trusting. Year-scoping would have been **worthless** if the
library had created an unfiltered consumer and matched subjects client-side: HQ would still
receive every message ever published and merely discard most of them, so the boot cost would be
unchanged while the code claimed otherwise. The filter is enforced by the server, so the
saving is real.

(An earlier reading of `jsz?consumers=1` showed `filter=None` for every consumer and briefly
looked like exactly that failure. It was the query's fault: consumer *config* is only included
when `config=1` is also passed. Worth knowing before someone else raises a false alarm.)

### Volume: measured per point, then extrapolated

Dev holds real, if small, traffic — enough to measure the *unit* costs, which is what the
extrapolation needs:

| measured (dev, 2026-09-09) | value |
| --- | --- |
| `TELEMETRY` stream | 5 messages, 73,205 bytes |
| `track_point` rows from those messages | 1,202 |
| wire cost per point | **~61 bytes** (73,205 / 1,202) |
| points per message | ~240 |
| `NATHEJK` stream, all of HQ's history to date | 29,393 messages, 18.6 MB |

Extrapolated to one full event at the ceiling this PRD sizes against (§11: ~3,600 points per
person, ~200 patruljer × ~4 reporting phones ≈ 2.9 M points):

- **~176 MB** and **~12,000 messages** of telemetry **per year**, retained indefinitely.
- That is **~40% more messages than HQ's entire NATHEJK history so far**, added every year.

So the concern this task was raised about is real and is not marginal. Sparse recording will
make reality a fraction of the ceiling, but the direction is one-way: it accumulates.

### Boot time: derived from replay throughput

Measured over three restarts (`docker compose restart api`, polling `/api/v1/healthcheck`):
**2.6 s, 3.2 s, 5.1 s**. That boot replays the whole `NATHEJK` stream — 29,393 messages — so
replay throughput is on the order of **7,000–10,000 messages/second**.

At ~12,000 telemetry messages per event, an **unscoped** subject would therefore add roughly
**1.2–1.7 s of boot time per past event**, for ever. Year-scoped, it adds that for the current
event only and never grows.

This is a derivation from a measured rate, not an A/B of two builds, and that is deliberate:
dev holds 5 telemetry messages, so timing the two subjects against *this* stream would produce
two identical numbers and prove nothing. The rate generalises; the local difference does not.

Re-take after the event with the same method — and note that a 1.7 s/year drift is tolerable for
several years, so the value of year-scoping is that it removes an unbounded growth term, not
that it fixes an urgent problem.

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
- 2026-09-09 17:05 — Monitoring port enabled, so the remaining two criteria became answerable
  after all. **Verified the year filter is registered server-side**
  (`filter_subject: TELEMETRY.2026.track.*.reported`) — without that, year-scoping would have
  been decorative and the boot cost unchanged. Measured per-point wire cost (~61 bytes) and
  replay throughput (~7–10k msgs/s), and extrapolated both: ~176 MB and ~12k messages per event,
  and ~1.2–1.7 s of boot time per past event if unscoped. Recorded above and in PRD 011 §8.
- 2026-09-09 17:10 — Completed. All criteria met, with the volume and boot-time answers stated as
  extrapolations from measured unit costs rather than as event observations — which is the
  honest form of the answer, and is re-checkable with the same method after the race.
