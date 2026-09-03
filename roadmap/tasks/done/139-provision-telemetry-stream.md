# 139 — provision the TELEMETRY stream in NATS

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** knj
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §8. **Blocking and not code** — needs someone with access to the NATS deployment.

`stream.New()` in `github.com/jrgensen/stream` has its `CreateStream` block commented out, so
hq never creates a stream. A consumer declaring `TELEMETRY.*.track.*.reported` resolves the
JetStream stream name from the subject's **domain**, so the stream must already exist and must
be named **`TELEMETRY`** — uppercase, exactly. If it is missing or differently cased,
`OrderedConsumer` fails, `mux.Run` returns an error and the api `PrintFatal`s at boot.

So the failure mode of getting this wrong is not "telemetry is missing", it is "hq does not
start". Task 141 must not be deployed before this is confirmed in the target environment.

Retention: the stream is retained indefinitely by design (per-person subjects are the erasure
unit, purged with `nats stream purge --subject`). Nothing to decide here; hq's *own* retention
is task 153.

## Acceptance Criteria

- [x] A stream named `TELEMETRY` exists in dev, with subjects `TELEMETRY.>`
- [x] Same confirmed in stage
- [x] `hej-api` is observed publishing `TELEMETRY.{year}.track.{personId}.reported` into it
- [x] Confirmed that a hq api boot against the stream does not fatal (dev)

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — **Dev half appears already satisfied**, found while verifying task 147. The api has been running the `table/track` consumer through hot-reload without fatalling, and `track_point` holds 1,202 real points for one person — so a stream named `TELEMETRY` exists in dev and `hej-api` is publishing into it. Left open because stage is unverified, and stage is where a missing or mis-cased stream stops the api from booting at all.
- 2026-09-03 — **knj: the stream is created and available across all environments.** That closes the last criterion and, with it, the deployment gate on task 141 — `table/track` was the first consumer in hq whose subject domain is not `NATHEJK`, so until now deploying it risked the api failing to boot rather than merely lacking telemetry.
- 2026-09-03 — Completed. PRD 011's implemented work is now deployable; what remains (152, 153) does not block a release.
