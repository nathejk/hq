# 140 — widen live signals to a second stream domain

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §8. Prerequisite for task 141 — without it the telemetry projection is silently
invisible to the SPA.

`go/internal/live/signal.go` has a hardcoded `domain = "NATHEJK"` const and
`SignalFromSubject` returns `ErrNotASignal` for any other domain. A `TELEMETRY.…` subject
therefore produces no signal at all, so `live.NotifyAll` would wrap the new consumer and emit
nothing — the failure is silent, which is exactly why this is its own task.

Widen the check to a set of known domains (`NATHEJK`, `TELEMETRY`) keeping every existing
shape rule (3 parts → year, 4 → collection, ≥5 → entity + id + event) and the existing
case-insensitivity. `live.EntitiesFrom` then advertises `track` over the SSE `entities` event
with no further change.

This is the only change to shared live plumbing in PRD 011. Existing `NATHEJK` behaviour must
not shift by a hair.

## Acceptance Criteria

- [x] `SignalFromSubject` accepts `TELEMETRY.2026.track.{id}.reported` → `Entity: "track"`, `ID: id`, `Year: "2026"`, `Event: "reported"`
- [x] An unknown domain still returns `ErrNotASignal`
- [x] Existing live tests pass unchanged
- [x] A test covers the new domain and a rejected one
- [x] `go vet ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: replace the `domain` const with a set of known domains, keep every shape rule and the case-insensitive compare, extend `signal_test.go`.
- 2026-09-03 — Kept it a **closed set** (`knownDomains` + `knownDomain()`) rather than relaxing the check to "any domain with enough parts". An unrecognised domain is far more likely a typo or an unwired stream than a new integration, and inventing a signal for it would invalidate caches under a token no page depends on.
- 2026-09-03 — Decision worth noting: the **three-part form stays NATHEJK-only**. `NATHEJK.{year}.{event}` means "the year entity changed", which is a NATHEJK convention; a hypothetical `TELEMETRY.{year}.{event}` is not a statement about the year, and reporting it as one would invalidate every year-dependent page for an unrelated reason. Per `ErrNotASignal`'s own contract — no signal beats a wrong one — that shape is now rejected for other domains, and there is a test for it.
- 2026-09-03 — ✅ All criteria: `go test ./internal/live/...` ok, `go vet` clean. Added four cases — telemetry subject (from the real payload in PRD 011 §4a), lowercase telemetry domain, `TELEMETRY.2026.reported` rejected, and `TELEMETRYX`/`NATS` rejected so the set stays closed.
- 2026-09-03 — Checked whether the SPA needs `track` adding to an allow-list: **it does not.** `vue/src/plugins/live/entities.ts` validates against the set the server advertises, and `live.EntitiesFrom` derives that from each wired consumer's `Consumes()` via this same function — so `track` appears in the dev-warning allow-list for free the moment task 141 wires the projection. No frontend change in this task.
- 2026-09-03 — Completed. Telemetry subjects now produce signals; nothing else changed shape.
