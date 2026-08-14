# 059 — BFF: wire the number saga into hq's mux

**Status:** done
**Priority:** high
**Created:** 2026-08-13
**Picked up by:** zed agent session
**Started:** 2026-08-13
**Completed:** 2026-08-13

## Description

Follows 057 and 058. Construct the saga in `go/cmd/api/main.go` and add it to the
`projections` slice so `live.NotifyAll` wraps it and the mux subscribes it.

**hq is the only service that may mount this saga** (PRD 003 §8 Ownership).
Subscriptions are ephemeral ordered consumers with no queue group, so every
process receives every message; two mounts would both find a patrulje unnumbered
and both publish, and the projector's `UPDATE patrulje SET teamNumber=?` is
unconditional, so the numbers would fight rather than converge. Record that at the
call site, next to the comment already explaining why the order Pay saga is
*absent* here — the two decisions are mirror images and are easy to misread.

It goes in the `projections` slice rather than a bare `mux.AddConsumer` because
that slice is what `live.NotifyAll` wraps, and the wrapper is what forwards
`CaughtUp` (`internal/live/notify.go`). Adding it outside the slice would leave
the saga permanently non-live and silently publishing nothing — the failure this
whole design turns on.

Dependencies to pass: the publisher, `ordertable` (order Queries),
`producttable` (product Queries), `patruljetable` (patrulje Queries) and
`currentYear`. All already exist in `main.go`.

## Acceptance Criteria

- [x] Saga constructed in `main.go` and present in the `projections` slice.
- [x] A comment at the call site states that hq is the sole owner and must remain
      so, and why it must be inside the wrapped slice.
- [x] `live.EntitiesFrom` still advertises a sensible token set — the saga
      contributes `order` and `patrulje`, both already advertised, so the set is
      unchanged.
- [x] Builds and tests clean on both resolution paths: `go build ./...` /
      `go test ./...` and the same with `GOWORK=off`.
- [x] `gofmt -l .` and `go vet ./...` clean.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
- 2026-08-13 18:40 — Picked up. Plan: construct after `ordertable` (which it
  reads), add to the `projections` slice, and put the ownership note next to the
  existing comment explaining why the order Pay saga is absent — the two
  decisions are mirror images and a reader needs both to make sense of either.
- 2026-08-13 18:45 — Constructed as `patruljenumbers` after `ordertable` (its
  reader) and added to `projections` next to `patruljetable`, so the slice reads
  in domain order. Ownership comment written directly below the one explaining the
  Pay saga's absence, and it spells out the asymmetry: the single-owner rule binds
  *harder* here, because the patrulje projector's UPDATE is unconditional, so two
  mounts would overwrite each other rather than converge the way duplicate
  order.paid events do.
- 2026-08-13 18:50 — ✅ Entity-token criterion verified rather than assumed: the
  saga's subjects are `*.order.*.paid` and `*.patrulje.*.numberassigned`, which
  yield the tokens `order` and `patrulje`; `ordertable` already consumes
  `*.order.*.paid` and `patruljetable` already consumes
  `*.patrulje.*.numberassigned`, and `EntitiesFrom` dedupes through a map — so the
  advertised set is byte-identical.
- 2026-08-13 18:52 — ✅ Remaining criteria complete: `gofmt`, `go vet` and
  `go build`/`go test ./...` clean under both `go.work` and `GOWORK=off`.
- 2026-08-13 18:55 — Completed. The feature is now live-wired end to end; 060
  verifies the replay/restart properties against a running stack and ships the PRD.
