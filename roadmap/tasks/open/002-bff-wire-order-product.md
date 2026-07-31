# 002 — BFF: wire order + product into main.go and data.Models

**Status:** open
**Priority:** high
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `product` and `order` are constructed in `main.go`.
- [ ] Both are registered with the `xstream.Mux` consumer set.
- [ ] Order read API is reachable from handlers via `data.Models`.
- [ ] Product catalogue is seeded (or seeding path confirmed) for the year.
- [ ] `go test ./...`, `go vet ./...`, `staticcheck ./...`, `go build ./...` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
