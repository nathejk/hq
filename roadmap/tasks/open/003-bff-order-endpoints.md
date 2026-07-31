# 003 — BFF: order endpoints GET /api/orders and GET /api/order/:id

**Status:** open
**Priority:** high
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). Add read endpoints so
the frontend can list and inspect orders.

New file `go/cmd/api/order.go`:
- `listOrdersHandler` (`GET /api/orders`) — orders for `app.YearSlug(r)` using
  the year-wide list query from task 001. Enrich each row with a human-readable
  owner label via a **secondary lookup** (not a SQL join): group orders by
  `ownerType`, then batch-fetch names/numbers from the corresponding read model
  (patrulje/klan/personnel/…) through `app.models`. Keep the `order` querier
  free of cross-table joins on `ownerId`.
- `showOrderHandler` (`GET /api/order/:id`) — reuse the existing
  `order.Queries.GetByID` (already returns lines + paid/due amounts).

Register both routes in `go/cmd/api/routes.go`. Use `app.WriteJSON` /
`app.ReadJSON` / `app.ServerErrorResponse` / `app.NotFoundResponse` — do not
hand-roll JSON or `http.Error`.

**Every endpoint must have OpenAPI annotations** (repo `.rules`).

Owner scope for the secondary lookup depends on an open question in the PRD
(teams only vs. also personnel) — default to patrulje/klan and personnel if a
personnel read model is readily available; otherwise note the gap.

Depends on: 001 (list query), 002 (wiring + `data.Models`).

## Acceptance Criteria

- [ ] `GET /api/orders` returns year-scoped orders with owner label, currency,
      total, paid, due, and status.
- [ ] Owner label resolved via secondary batched lookup, not a join.
- [ ] `GET /api/order/:id` returns a single order with lines + paid/due.
- [ ] Both routes registered and both have OpenAPI annotations.
- [ ] `go test ./...`, `go vet ./...`, `staticcheck ./...` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
