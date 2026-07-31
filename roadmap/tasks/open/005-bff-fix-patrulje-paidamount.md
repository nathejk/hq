# 005 — BFF: fix patrulje list paidAmount to resolve payments via orders

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). In
`go/nathejk/table/patrulje/query.go`, `GetAll` derives `paidAmount` with a
subquery that joins `payment.orderForeignKey = p.teamId`. With the order layer,
`payment.orderForeignKey` now references an **order id**, not a team id, so this
computation is wrong.

Correct it to resolve payments through orders — sum `reserved`/`received`
payments whose `orderForeignKey` matches an order whose `ownerId = p.teamId`
(and `ownerType = 'patrulje'`, year-scoped) — or reuse the order read model's
paid-amount logic. Keep the derived signup/paid status semantics in `GetAll`
(pay / paid / semipaid) working against the corrected total.

Watch out: the payment table still holds legacy rows keyed by team id (PRD open
question); make sure the corrected query does not double-count or crash on such
rows.

Depends on: 002 (order tables available); relates to the order paid-amount
subquery in `go/nathejk/table/order/querier.go`.

## Acceptance Criteria

- [ ] `patrulje` list `paidAmount` is computed through orders, not
      `orderForeignKey == teamId`.
- [ ] Derived signup/paid status (pay/paid/semipaid) still correct.
- [ ] No crash / double-count on legacy team-keyed payment rows.
- [ ] `go test ./...`, `go vet ./...`, `staticcheck ./...` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
