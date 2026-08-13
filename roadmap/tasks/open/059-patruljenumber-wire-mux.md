# 059 — BFF: wire the number saga into hq's mux

**Status:** open
**Priority:** high
**Created:** 2026-08-13
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Saga constructed in `main.go` and present in the `projections` slice.
- [ ] A comment at the call site states that hq is the sole owner and must remain
      so, and why it must be inside the wrapped slice.
- [ ] `live.EntitiesFrom` still advertises a sensible token set — the saga
      contributes `order` and `patrulje`, both already advertised, so the set is
      unchanged.
- [ ] Builds and tests clean on both resolution paths: `go build ./...` /
      `go test ./...` and the same with `GOWORK=off`.
- [ ] `gofmt -l .` and `go vet ./...` clean.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
