# 005 — BFF: fix patrulje list paidAmount to resolve payments via orders

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] `patrulje` list `paidAmount` is computed through orders (payments joined
      to orders owned by the team), not solely `orderForeignKey == teamId`.
- [x] Derived signup/paid status (pay/paid/semipaid) still correct — logic
      unchanged, fed by the corrected `paidAmount`.
- [x] No crash / double-count on legacy team-keyed payment rows (kept as an
      `OR` branch; a payment row matches at most one branch).
- [x] `go build`/`go vet` clean for the patrulje package.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 13:55 — Picked up. Plan: change the `paidAmount` subquery in `patrulje/query.go` `GetAll` to sum payments where `orderForeignKey` is EITHER the teamId (legacy) OR an order owned by the team (new). The two branches are mutually exclusive per payment row, so no double-count — and it keeps working through the data transition.
- 2026-07-31 14:00 — Rewrote the subquery accordingly (`WHERE status IN (...) AND (orderForeignKey = p.teamId OR orderForeignKey IN (SELECT orderId FROM orders WHERE ownerType='patrulje' AND ownerId=p.teamId))`). build + vet clean. Note: SQL not exercised against a live DB in this environment. Completed.
