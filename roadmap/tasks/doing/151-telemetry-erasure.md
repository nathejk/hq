# 151 — per-person telemetry erasure

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §6, §8. Compliance, not cleanup. Depends on task 141.

The producer publishes on a **per-person subject** precisely so one individual can be erased:
`nats stream purge --subject 'TELEMETRY.*.track.<personId>.reported'`. That guarantee is void
the moment hq projects the stream into MySQL — the purge removes the source, but hq's rows
persist, and a replay-built read model will never notice their absence. **Without this task, hq
becomes the place erased location data survives.**

Deliver a deliberate, runnable, documented delete keyed by `personId` across `track_latest` and
`track_point`, to be run alongside the stream purge. It does not need a UI; it needs to exist,
to be findable, and to be obviously correct.

Note the ordering trap: deleting hq's rows **before** the stream is purged accomplishes nothing,
because the next replay restores them. Document that the purge comes first, then the delete —
and that a replay after a purge is what makes the erasure durable.

Positions are personal data about named people, many of them minors. Treat the write-up as part
of the deliverable rather than a comment.

## Acceptance Criteria

- [ ] A delete keyed by `personId` covering `track_latest` and `track_point`
- [ ] Documented runbook: purge stream subject first, then delete hq rows
- [ ] Documented where the command lives and how to run it in each environment
- [ ] Stated in PRD 011 or the runbook that a replay after purge is what makes erasure durable
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
