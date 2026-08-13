# 057 — BFF: patrulje number-assignment saga

**Status:** open
**Priority:** high
**Created:** 2026-08-13
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `go/nathejk/table/patruljenumber/` exists with a constructor taking the
      publisher, order Queries, product Queries and the year, and returning a
      `cqrs.Consumer`.
- [ ] Implements `CaughtUp()` so it can be recognised as a
      `stream.CatchupListener`, with a compile-time assertion pinning that.
- [ ] Publishes nothing before `CaughtUp` — covered by a test that replays a
      qualifying `order.paid` and asserts zero published messages.
- [ ] Publishes `numberassigned` with `maxNumber+1` for a live, unassigned,
      eligible patrulje — asserted on both subject and body.
- [ ] Already-assigned patrulje produces no event (idempotent), covered for both
      "assigned via replayed event" and "assigned earlier in this process".
- [ ] Ineligible patrulje (seatCount < 3) produces no event, and does not retry
      to the full attempt budget.
- [ ] An order not yet projected as paid is retried, and transitions once the
      projection arrives — covered by a fake that misses the first read.
- [ ] Non-patrulje owners and other years are ignored.
- [ ] Two eligible patruljer in succession get consecutive distinct numbers.
- [ ] `go test ./...`, `go vet ./...` and `gofmt -l .` clean; not wired into the
      mux by this task.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
