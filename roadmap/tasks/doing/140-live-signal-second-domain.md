# 140 — widen live signals to a second stream domain

**Status:** doing
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:**

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

- [ ] `SignalFromSubject` accepts `TELEMETRY.2026.track.{id}.reported` → `Entity: "track"`, `ID: id`, `Year: "2026"`, `Event: "reported"`
- [ ] An unknown domain still returns `ErrNotASignal`
- [ ] Existing live tests pass unchanged
- [ ] A test covers the new domain and a rejected one
- [ ] `go vet ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: replace the `domain` const with a set of known domains, keep every shape rule and the case-insensitive compare, extend `signal_test.go`.
