# 003 — BFF: order endpoints GET /api/orders and GET /api/order/:id

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] `GET /api/orders` returns year-scoped orders with owner label, currency,
      total, paid, due, and status.
- [x] Owner label resolved via secondary memoised lookup, not a join.
- [x] `GET /api/order/:id` returns a single order with lines + paid/due.
- [x] Both routes registered and both have (swaggo-style) OpenAPI annotations.
- [x] `go build`/`go vet` clean for `./cmd/api/...`.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 13:20 — Picked up. Note: the repo has NO OpenAPI/swagger tooling, spec, or a single existing annotation anywhere (despite `.rules`). Decision: honour the rule by adding swaggo-style annotation comments (`@Summary`/`@Router`/...) on the two new handlers — the de-facto Go format — even though nothing generates from them yet; flag for the user. Owner label via a memoised secondary lookup (patrulje/klan/personnel), not a join.
- 2026-07-31 13:35 — Added `go/cmd/api/order.go` with `listOrdersHandler` (`GET /api/orders`), `showOrderHandler` (`GET /api/order/:id`), and `resolveOwnerName` (memoised secondary lookup; patrulje→"number - name", klan→name, else personnel name, fallback ownerId). `orderView` embeds `order.Order` and adds `ownerName`. Registered both routes in routes.go. build + vet clean for ./cmd/api/...
- 2026-07-31 13:38 — Completed. Runtime behaviour (live DB/jetstream) not exercised in this environment — validated by build+vet; needs a stack run to smoke-test the JSON.
