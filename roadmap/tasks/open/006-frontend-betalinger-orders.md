# 006 — Frontend: Betalinger page shows orders

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). Convert
`vue/src/views/PaymentListView.vue` from a raw-payments table to an
**order-centric** table backed by `GET /api/orders` (task 003).

- Fetch `/api/orders` (via the existing `http`/`@/helpers` client).
- `DataTable` columns: Tidspunkt (created), Ejer (owner name/number from the
  secondary-lookup label), Beløb (total), Betalt (paid), Mangler (due), Status
  tag. Status is binary **Åben / Betalt** (the API emits `OPEN`/`PAID`;
  cancelled collapses to `PAID`). Amounts are minor units — divide by 100 for
  `da-DK` currency formatting (reuse the existing `formatAmount`).
- Keep an expander row showing the order **lines** (product name, member, qty,
  unit price, line total). Optionally also show the underlying payment
  transactions (PRD open question — decide with reviewer).

Depends on: 003 (`GET /api/orders`).

## Acceptance Criteria

- [ ] Betalinger lists orders (not raw payments) for the current year.
- [ ] Columns include owner, total, paid, due, and Åben/Betalt status tag.
- [ ] Row expansion shows order line items.
- [ ] Amounts formatted in `da-DK` DKK; labels in Danish.
- [ ] `npm run lint` and `npm run test:unit` pass.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
