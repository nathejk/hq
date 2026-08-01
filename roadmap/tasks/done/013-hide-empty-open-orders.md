# 013 — Hide empty (worth 0) open orders

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Open orders with a `totalAmount` of 0 (empty orders that were opened but never
had priced lines added) should not be shown in the UI. Filter them out of the
order list queries used for display: `ListByYear` (Betalinger, `GET /api/orders`)
and `ListByOwner` (Patrulje page). Both are display-only — neither is used by
the order command/saga layer — so filtering in the query is safe.

Rule: exclude rows where `status = 'open' AND totalAmount = 0`. A paid/cancelled
order is always kept (they collapse to PAID for display), and any open order
with a positive total is kept.

## Acceptance Criteria

- [x] `ListByYear` excludes `status='open' AND totalAmount=0`.
- [x] `ListByOwner` excludes `status='open' AND totalAmount=0`.
- [x] `go build`/`go vet`/`staticcheck` stay green for the order package.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 16:10 — Task created + picked up. Filtering empty open orders in the two display queries.
- 2026-07-31 16:15 — Added `AND NOT (o.status = 'open' AND o.totalAmount = 0)` to both `ListByOwner` and `ListByYear`. build/vet/staticcheck clean. Completed.
