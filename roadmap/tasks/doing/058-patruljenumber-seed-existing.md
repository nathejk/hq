# 058 — BFF: seed the number saga from existing teamNumbers

**Status:** doing
**Priority:** high
**Created:** 2026-08-13
**Picked up by:** zed agent session
**Started:** 2026-08-13
**Completed:**

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

- [ ] `CaughtUp` folds existing `teamNumber`s for the year into `assigned` and
      `maxNumber` before the saga starts publishing.
- [ ] Test: an existing `teamNumber` of 300 and no history ⇒ the next assignment
      is 301.
- [ ] Test: a patrulje that already has a `teamNumber` is never assigned a second
      one, even with no `numberassigned` event in history.
- [ ] Test: empty and non-numeric `teamNumber`s are skipped and do not affect
      `maxNumber`.
- [ ] Test: `maxNumber` takes the highest of (replayed events, existing
      teamNumbers) — verified with the larger value on each side in turn.
- [ ] A failing seed read leaves the saga not-live (publishes nothing) and logs.
- [ ] `go test ./...`, `go vet ./...`, `gofmt -l .` clean.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
- 2026-08-13 18:00 — Picked up. Plan: add a narrow `PatruljeReader` interface
  (GetAll by year only), fold existing numbers in from `CaughtUp` before flipping
  `live`, and keep the saga not-live if that read fails — going live with a
  too-low `maxNumber` would re-issue a number that is already on a patrulje.
