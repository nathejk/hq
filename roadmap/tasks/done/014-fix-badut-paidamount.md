# 014 — Fix badut paidAmount so paid gøglere aren't hidden

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Some gøglere are missing from the `/badut` list. Root cause: the frontend
`BadutListView.vue` shows only personnel with `paidAmount > 0`, but the backend
`personnel/query.go` `GetAll` computes `paidAmount` with the pre-order-layer
assumption `payment.orderForeignKey = personnel.userId`. With the order layer,
payments reference an **order id**, so gøglere who paid via an order compute
`paidAmount = 0` and get filtered out.

Same class of bug as task 005 (patrulje list). Fix: resolve `paidAmount`
through orders as well — sum `reserved`/`received` payments whose
`orderForeignKey` is EITHER the userId (legacy) OR an order owned by that user
(`orders.ownerId = personnel.userId`). The two branches are mutually exclusive
per payment row, so no double-count, and it keeps working through the data
transition.

## Acceptance Criteria

- [x] `personnel/query.go` `GetAll` `paidAmount` resolves payments via orders (legacy-tolerant `OR`).
- [x] Gøglere who paid via an order now compute `paidAmount > 0` (so the client `paidAmount > 0` filter keeps them).
- [x] `go build`/`go vet`/`staticcheck` stay green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 16:25 — Task created + picked up. Diagnosed: client filters paidAmount>0; backend paidAmount used legacy orderForeignKey=userId join.
- 2026-07-31 16:30 — Updated the `paidAmount` subquery to `WHERE status IN (...) AND (orderForeignKey = personnel.userId OR orderForeignKey IN (SELECT orderId FROM orders WHERE ownerId = personnel.userId))`. build/vet/staticcheck clean. Note: not exercised against a live DB here. Completed.
