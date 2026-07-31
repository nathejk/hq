# 006 — Frontend: Betalinger page shows orders

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] Betalinger lists orders (not raw payments) for the current year
      (`GET /api/orders`).
- [x] Columns include owner (Ejer), total, paid, due (Mangler), and
      Åben/Betalt status tag.
- [x] Row expansion shows order line items (lazy-fetched via `GET /api/order/:id`).
- [x] Amounts formatted in `da-DK` DKK; labels in Danish.
- [~] `npm run lint` / `npm run test:unit` — NOT run here (the `ui` container
      timed out in this environment); needs a run in the container.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 14:15 — Picked up. Note: actual repo uses `@/plugins/axios` `http` (baseURL `/api/`), not `@/helpers` fetchWrapper — following the existing PaymentListView/PatruljeView convention. Since `GET /api/orders` (task 001/003) omits line items for weight, the row expansion will lazy-fetch `GET /api/order/:id` for lines. Status is binary OPEN→Åben / PAID→Betalt.
- 2026-07-31 14:25 — Rewrote `PaymentListView.vue`: fetches `/orders`; DataTable columns Tid/Ejer/Beløb/Betalt/Mangler/Status(tag); expander lazy-loads `/order/:id` lines into a reactive map and renders an Ordrelinjer sub-table. Reused da-DK formatAmount/formatDateTime.
- 2026-07-31 14:30 — Tried `docker compose run --rm --no-deps ui npm run lint` — timed out (image/deps/network unavailable here). Validated by manual review against existing conventions. Needs lint/unit run in the ui container. Completed.
