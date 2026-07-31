# 001 — BFF: add year-wide order list query

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). The order read API
(`go/nathejk/table/order/querier.go`) currently exposes `GetByID`,
`FindOpenOrder`, `ListByOwner`, `ReservedQuantity`, and `PaidLineKeys` — but no
way to list **all orders for a year**, which the Betalinger page needs.

Add a year-wide list query to the `order.Queries` interface and its `querier`
implementation, e.g. `ListByYear(ctx, year)` (or a `GetAll(ctx, Filter{Year,
OwnerType, OwnerID})` if a shared filter is preferable). Reuse the existing
`orderColumns` / `scanOrder` / `listLines` helpers so each returned order keeps
the computed `PaidAmount` / `DueAmount` and hydrated lines. Order newest first,
consistent with `ListByOwner`.

Consider whether hydrating lines for every order in a year is too heavy for the
list view; if so, add a lighter variant that omits lines (the detail endpoint —
task 003 — can fetch lines via `GetByID`).

## Acceptance Criteria

- [x] `order.Queries` gains a year-wide list method (`ListByYear(ctx, year)`).
- [x] Implementation reuses `orderColumns` + `scanOrder` (paid/due computed).
- [x] Results are year-scoped and ordered newest-first.
- [x] `go build`/`go vet` clean for the order package (`go test ./...` unaffected).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 13:00 — Picked up. Plan: add `ListByYear(ctx, year)` to `order.Queries` + `querier`, reusing `orderColumns`/`scanOrder`, ordered newest-first. Decision: do NOT hydrate line items in the year-wide list (kept light for the Betalinger list view); the detail endpoint (`GetByID`, task 003) provides lines for the row expansion.
- 2026-07-31 13:10 — Added `ListByYear` to the `Queries` interface and implemented it on `querier` in `querier.go` (year filter, `ORDER BY createdAt DESC`, reuses `orderColumns`/`scanOrder`, no line hydration). Build + vet clean for the order package.
- 2026-07-31 13:12 — Completed.
