# 058 — BFF: seed the number saga from existing teamNumbers

**Status:** done
**Priority:** high
**Created:** 2026-08-13
**Picked up by:** zed agent session
**Started:** 2026-08-13
**Completed:** 2026-08-13

## Description

Follows 057. The saga rebuilds `assigned` and `maxNumber` from replayed
`numberassigned` events, but that is not the only source of numbers: patruljer
already carry `teamNumber`s that no `numberassigned` event ever produced —
assigned manually, or written directly by other projectors (`klan/consumer.go`
inserts patrulje rows with a `teamNumber`).

If those are not counted, the first auto-assignment collides with an existing
number, and the projector's `UPDATE patrulje SET teamNumber=?` is unconditional,
so a collision silently overwrites nothing — it just hands two patruljer the same
number.

At `CaughtUp`, before going live, read the patrulje read model for the year and
fold it in: every non-empty numeric `teamNumber` marks its team assigned and
raises `maxNumber`. PRD 003's example: a manual 300 means the next auto number is
301, not 1.

**Notes**

- Read via `patrulje.Queries.GetAll(ctx, patrulje.Filter{YearSlug: year})`; the
  saga therefore gains a third read dependency, declared as a narrow local
  interface rather than taking the whole `Queries`.
- Seeding at `CaughtUp` rather than at construction is deliberate: at
  construction the projections are empty, since hq rebuilds its read model from
  the stream on every start. Seeding must happen after the patrulje projector has
  caught up — note in the code that this is not strictly guaranteed (independent
  consumers), and that the failure mode is a too-low `maxNumber`.
- Non-numeric or empty `teamNumber`s must be skipped, not treated as 0.
- A seeding read error must not be swallowed silently: log it, and prefer to stay
  not-live over going live with a `maxNumber` that could re-issue a number.

## Acceptance Criteria

- [x] `CaughtUp` folds existing `teamNumber`s for the year into `assigned` and
      `maxNumber` before the saga starts publishing.
- [x] Test: an existing `teamNumber` of 300 and no history ⇒ the next assignment
      is 301.
- [x] Test: a patrulje that already has a `teamNumber` is never assigned a second
      one, even with no `numberassigned` event in history.
- [x] Test: empty and non-numeric `teamNumber`s are skipped and do not affect
      `maxNumber`.
- [x] Test: `maxNumber` takes the highest of (replayed events, existing
      teamNumbers) — verified with the larger value on each side in turn.
- [x] A failing seed read leaves the saga not-live (publishes nothing) and logs.
- [x] `go test ./...`, `go vet ./...`, `gofmt -l .` clean.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
- 2026-08-13 18:00 — Picked up. Plan: add a narrow `PatruljeReader` interface
  (GetAll by year only), fold existing numbers in from `CaughtUp` before flipping
  `live`, and keep the saga not-live if that read fails — going live with a
  too-low `maxNumber` would re-issue a number that is already on a patrulje.
- 2026-08-13 18:20 — `CaughtUp` now seeds before opening the gate: reads the
  patrulje read model for the year via the new `PatruljeReader`, folds every
  non-empty `teamNumber` into `assigned`+`maxNumber`, and on a read error logs
  and returns *without* going live. Left an explicit note that catch-up of this
  consumer does not prove the patrulje projector finished its replay, so the
  seeded mark can be low — mitigated by the replayed `numberassigned` events
  also feeding `assigned`, and by every issued number being observed back.
- 2026-08-13 18:25 — Decision: added a mutex. Seeding moved state-mutation out of
  the single HandleMessage goroutine — `CaughtUp` can fire from the subscribe
  path's own goroutine (immediately, for an empty backlog), so `assigned` and
  `maxNumber` are now guarded, and the publish+mark in `attempt` is held under
  the lock so a number cannot be handed out twice. Contention is nil (one writer
  plus a one-shot seed).
- 2026-08-13 18:35 — ✅ All criteria complete. Added a `-race` test that runs
  `CaughtUp` and `HandleMessage` concurrently. 28 tests total (was 18), pass
  under `-race`; `gofmt`, `go vet`, `go test ./...` clean on both resolution
  paths. Still unwired — 059 wires it.
- 2026-08-13 18:37 — Completed. Seeding from existing teamNumbers implemented,
  manual/legacy numbers respected, failed-seed stays dormant.
