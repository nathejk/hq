# 057 — BFF: patrulje number-assignment saga

**Status:** done
**Priority:** high
**Created:** 2026-08-13
**Picked up by:** zed agent session
**Started:** 2026-08-13
**Completed:** 2026-08-13

## Description

Implement the acceptance saga from PRD 003: when a patrulje has paid for at
least 3 seats, publish `NATHEJK.{year}.patrulje.{teamId}.numberassigned` with
the next number for the year. The existing `patrulje` projector already consumes
that subject and writes `teamNumber`; nothing publishes it today.

New package `go/nathejk/table/patruljenumber/`. This task delivers the saga and
its unit tests only — seeding from existing `teamNumber`s is 058 and wiring is
059. It must not be added to the mux here.

Model it on the order Pay saga (`shared-go/tables/order/saga.go`), which solves
the same problems: year scoping, a live/replay gate, and reads against
projections that may lag.

**Behaviour**

- `Consumes()`: `NATHEJK:*.order.*.paid` (the trigger) **and**
  `NATHEJK:*.patrulje.*.numberassigned` (its own output, so replay rebuilds
  state).
- On `numberassigned`, replay or live: mark the team assigned, raise `maxNumber`
  to at least that number, publish nothing.
- On `order.paid`: the body carries only `OrderID`, so read the order for its
  owner and year. Ignore non-patrulje owners and other years.
- Eligibility: `seatCount >= 3`, where seatCount is the paid quantity across
  participation SKUs for that owner — `order.Queries.PaidQuantityBySKU` summed
  over the SKUs from `product.Queries.ListEligibleFor(year, TeamTypePatrulje)`
  filtered to `Kind == KindParticipation`. Do not hardcode
  `participation.patrulje`, and do not count merchandise.
- Publish only when live (post-`CaughtUp`) and only if the team is not already
  assigned. After publishing, optimistically mark assigned and bump `maxNumber`
  so two qualifiers in quick succession get distinct consecutive numbers.
- `TeamNumber` is a plain decimal string (`strconv.Itoa`).

**Projection lag must be handled.** `PaidQuantityBySKU` only counts orders whose
*projected* status is `paid`, and the order projector is an independent consumer,
so right after `order.paid` the saga can read seatCount 0 for an order that is
in fact paid — and nothing would ever re-trigger it. Re-read a bounded number of
times until the order reads `paid`, waiting between attempts, mirroring
`resultUnprojected` / `waitBeforeRetry` in the order saga. Distinguish "not
projected yet" (retry) from "genuinely fewer than 3 seats" (terminal), or a
2-seat patrulje burns the full budget on every event.

**Do not** publish during replay. That is the crux of the whole design: it would
emit duplicate assignments on every restart.

## Acceptance Criteria

- [x] `go/nathejk/table/patruljenumber/` exists with a constructor taking the
      publisher, order Queries, product Queries and the year, and returning a
      `cqrs.Consumer`.
- [x] Implements `CaughtUp()` so it can be recognised as a
      `stream.CatchupListener`, with a compile-time assertion pinning that.
- [x] Publishes nothing before `CaughtUp` — covered by a test that replays a
      qualifying `order.paid` and asserts zero published messages.
- [x] Publishes `numberassigned` with `maxNumber+1` for a live, unassigned,
      eligible patrulje — asserted on both subject and body.
- [x] Already-assigned patrulje produces no event (idempotent), covered for both
      "assigned via replayed event" and "assigned earlier in this process".
- [x] Ineligible patrulje (seatCount < 3) produces no event, and does not retry
      to the full attempt budget.
- [x] An order not yet projected as paid is retried, and transitions once the
      projection arrives — covered by a fake that misses the first read.
- [x] Non-patrulje owners and other years are ignored.
- [x] Two eligible patruljer in succession get consecutive distinct numbers.
- [x] `go test ./...`, `go vet ./...` and `gofmt -l .` clean; not wired into the
      mux by this task.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
- 2026-08-13 17:10 — Picked up. Plan: new package `patruljenumber` with the saga
  in `saga.go` (state + routing + eligibility) and narrow local interfaces for
  its three reads, mirroring the order saga's shape — tri-state attempt result so
  "not projected yet" is retryable while "under three seats" is terminal. Tests
  with hand-written fakes and an injectable sleep, as the order saga does, so the
  retry/wait behaviour is assertable without real time passing.
- 2026-08-13 17:35 — `saga.go` written. Two decisions worth recording:
  (a) eligibility is checked *before* the live gate rather than after, so the
  read path stays exercised during replay and a bug there surfaces on the next
  restart instead of on the first live payment;
  (b) `markAssigned` treats a non-numeric `teamNumber` as "assigned" but not as a
  high-water mark — it is a number somebody meant, so the team must not be
  renumbered, but it has no integer value to raise the mark with.
- 2026-08-13 17:40 — Added the retry loop for projection lag. Waits happen during
  replay too, unlike the order saga's under-paid case: an unprojected order is
  only fixed by another consumer advancing, and back-to-back reads give it no
  chance to. Under-seated and non-patrulje are terminal, so the common
  no-op costs one read.
- 2026-08-13 17:50 — ✅ All criteria complete. 18 tests pass, `gofmt`, `go vet`
  and `go test ./...` clean on both resolution paths (`go.work` and `GOWORK=off`).
  Confirmed nothing under `cmd/` references the package, so it is inert until 059.
- 2026-08-13 17:52 — Added two tests beyond the criteria: merchandise must not
  count toward seats (four t-shirts + two seats is still two seats), and
  `order.Queries`/`product.Queries` must keep satisfying the narrow local reader
  interfaces — otherwise the wiring in 059 breaks with an error that points at
  `main.go` rather than at the cause.
- 2026-08-13 17:55 — Completed. Saga implemented and unit-tested, unwired.
  Follow-ups unchanged: 058 seeds state from existing `teamNumber`s, 059 wires it.
