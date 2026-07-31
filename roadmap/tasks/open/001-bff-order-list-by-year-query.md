# 001 — BFF: add year-wide order list query

**Status:** open
**Priority:** high
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `order.Queries` gains a year-wide list method (with a year filter).
- [ ] Implementation reuses existing column list + scan/paid/due logic.
- [ ] Results are year-scoped and ordered newest-first.
- [ ] `go test ./...`, `go vet ./...`, and `staticcheck ./...` stay green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
