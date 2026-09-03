# 151 — per-person telemetry erasure

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] A delete keyed by `personId` covering `track_latest` and `track_point`
- [x] Documented runbook: purge stream subject first, then delete hq rows
- [x] Documented where the command lives and how to run it in each environment
- [x] Stated in PRD 011 or the runbook that a replay after purge is what makes erasure durable
- [x] `go build ./...` clean (no Go changes — see log)

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: a runnable delete plus a runbook.
- 2026-09-03 — **Decision: documentation and SQL, not Go code, and deliberately no endpoint.** Three reasons, and I want them on the record because "add an endpoint" is the obvious instinct:
  1. **hq authenticates nobody.** `app.authenticate` attributes every request to an anonymous user; identity lives in an external service. A destructive, irreversible, unauthenticated endpoint that deletes personal data on request would be a larger risk than the one this task mitigates.
  2. **A Go helper with no caller is unreachable code.** hq has exactly one binary (`cmd/api`) and no CLI. A `track.Erase()` that nothing invokes would look like a capability while being dead weight, and would rot.
  3. **Two SQL statements are genuinely runnable today**, which is what the task asked for. Slower than a button, and safe.
  If erasure should become self-service it needs a verified actor first — real work, and not part of PRD 011. Flagged for knj.
- 2026-09-03 — Landed as `roadmap/api/telemetry-erasure.md`. The load-bearing part is the **ordering**: purge the stream *first*, delete hq's rows *second*. Reversed, the deletion accomplishes nothing — hq replays every projection from the stream on every api restart, so the next deploy, crash or dev hot-reload puts every deleted point straight back. Doing it in the right order is precisely what makes the erasure durable.
- 2026-09-03 — Documented the `*` in the year position of the purge subject: a person who took part in more than one event has a subject per year, and all of them should go.
- 2026-09-03 — Also documented what is **not** erased and why — QR scans (an operational record of a team touching a post, witnessed by crew, not a phone-derived trail), and membership/status history (what the event is administered from). Leaving that implicit is how a request gets half-answered.
- 2026-09-03 — ✅ **Verified the runbook's SQL against the dev database using a synthetic person** (`erase-test-1`) rather than real data: inserted into both tables, confirmed 1 row each, ran the documented deletes, confirmed 0 rows each. The procedure is tested, not just written.
- 2026-09-03 — Completed.
