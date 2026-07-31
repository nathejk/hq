# 007 — Frontend: Patrulje Betalinger section shows orders

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). In
`vue/src/views/PatruljeView.vue`, the "Betalinger" `DataTable` currently binds
to `:value="payments"`. Switch it to the team's **orders** from the
`/patrulje/:id` payload (task 004).

- Consume `orders` from the `/patrulje/:id` response (add to the `load()`
  mapping next to `payments`).
- `DataTable` columns: Tidspunkt, Beløb (total), Betalt (paid), Mangler (due),
  Status tag (Åben/Betalt). Reuse `getSeverity`/`formatAmount` patterns.
- Optionally make rows expandable to the order line items (consistent with
  task 006).

Depends on: 004 (team orders in the patrulje payload).

## Acceptance Criteria

- [ ] Patrulje "Betalinger" section lists the team's orders, not raw payments.
- [ ] Columns include total, paid, due, and Åben/Betalt status tag.
- [ ] Amounts formatted in `da-DK` DKK; labels in Danish.
- [ ] `npm run lint` and `npm run test:unit` pass.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
