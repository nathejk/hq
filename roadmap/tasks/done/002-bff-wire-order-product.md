# 002 — BFF: wire order + product into main.go and data.Models

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). The `order`
(`go/nathejk/table/order/`) and `product` (`go/nathejk/table/product/`)
projections exist but are **not** wired into the API binary.

In `go/cmd/api/main.go`:
- Construct `product.New(...)` and `order.New(p, writer, db, year, products)`
  (see `order.New` signature — it needs the publisher, writer, `*sql.DB`, the
  active `types.YearSlug`, and `product.Queries`).
- Add both projectors to `mux.AddConsumer(...)` so `order.*` events (and any
  product events, if applicable) are consumed and the tables are built on
  replay.
- Expose the order read API through `data.NewModels(...)` (e.g.
  `app.models.Order`) so handlers can query orders. Extend `data.Models`
  accordingly.

Check the `product` seeding path (`seeds_2026.go` / `seeder.go`) to confirm
whether seeding is triggered at startup and wire it if needed.

Depends on nothing else, but is a prerequisite for tasks 003, 004, 005.

## Acceptance Criteria

- [x] `product` and `order` are constructed in `main.go`.
- [x] `order` is registered with the `xstream.Mux` consumer set. (`product` is
      not event-sourced — it has no `Consumes`/`HandleMessage`; it is seeded, so
      it is intentionally *not* added to the mux.)
- [x] Order read API is reachable from handlers via `data.Models` (`Models.Order`).
- [x] Product catalogue is seeded for the year (`producttable.Seed(product.Seeds2026())` at startup).
- [x] `go build ./...`, `go vet` (for the touched packages), and `go test ./...` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 12:00 — Picked up (doing this before 001 so the package compiles and everything else can be validated). Decision: per user, Option A — align order/product code from `github.com/jrgensen/stream` to `nathejk.dk/superfluids/streaminterface` (the abstraction used by the other 53 files + the xstream mux), rather than a repo-wide jrgensen migration. Plan: swap stream imports/identifiers in the 4 order files, then wire product + order into main.go mux & data.Models.
- 2026-07-31 12:20 — Aligned stream imports/identifiers in commander.go, consumer.go, saga.go, table.go (`stream.*`→`streaminterface.*`, `subject.FromStr`→`streaminterface.SubjectFromStr`). gofmt'd.
- 2026-07-31 12:30 — Prereq discovered: order/product referenced `tablerow.EnsureColumn` / `tablerow.EnsureIndex`, which did not exist. Added them in new `go/pkg/tablerow/migrate.go` (information_schema existence check + ALTER via the Consumer). Packages now compile.
- 2026-07-31 12:45 — Wired in `cmd/api/main.go`: construct `product.New` + seed `Seeds2026()`, construct `order.New(js, writer, db, currentYear, producttable)`, add `ordertable` to `mux.AddConsumer`, pass `ordertable` to `data.NewModels`. Extended `data.Models` with `Order order.Queries` + `NewModels` param. Did NOT wire the order `saga` (payment→paid): it is marked "future" and its `PaymentReader` interface (`GetByReference(ref string)`) doesn't match the current payment API (`GetByReference(ctx, ref)`); out of scope for the display feature. Order status will read OPEN until an order.paid event exists; the read model still computes paidAmount from payments regardless.
- 2026-07-31 12:55 — Validated: `GOWORK=off go build ./...` clean; `go test ./...` passes; `go vet` clean for order/product/tablerow/cmd-api/data. Caveats: (1) `go vet ./...` reports one PRE-EXISTING, unrelated error in `nathejk/table/spejder/table.go` — `InitialTeamID` and `CurrentTeamID` both use json tag `"teamId"` (from the member-move work, not this task); flagged, left untouched. (2) `staticcheck` could not be run here (`go: no such tool "staticcheck"` under GOWORK=off in this environment).
- 2026-07-31 12:58 — Completed. Order + product read models are wired and building; order projections will populate on event replay.
