# 007 — Frontend: Patrulje Betalinger section shows orders

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] Patrulje "Betalinger" section lists the team's orders, not raw payments.
- [x] Columns include total, paid, due (Mangler), and Åben/Betalt status tag.
- [x] Amounts formatted in `da-DK` DKK; labels in Danish.
- [~] `npm run lint` / `npm run test:unit` — NOT run here (ui container
      unavailable in this environment); needs a run in the container.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 14:35 — Picked up.
- 2026-07-31 14:45 — Updated `PatruljeView.vue`: added `orders` ref (from `/patrulje/:id` `orders`), removed now-unused `payments` ref, added da-DK `formatAmount`/`formatDateTime` and `statusLabel`/`statusSeverity`. Replaced the Betalinger DataTable to show orders (Tidspunkt/Beløb/Betalt/Mangler/Status tag). `getSeverity` retained (still used by the members table). Manual review only — ui container unavailable here. Completed.
