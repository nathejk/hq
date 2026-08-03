# 016 — Ordrehistorik: title, Type column, inline filters

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Changes to the Betalinger page (`vue/src/views/PaymentListView.vue`):

1. Title changes from "Betalinger" to **"Ordrehistorik"**.
2. New column **"Type"** immediately after "Ejer", showing one of
   **Patrulje / Klan / Gøgler / Crew / Andet**, derived from the order's
   `ownerType` (`patrulje`/`klan`/`gøgler`/`crew`, anything else → Andet).
3. **Inline (row) filters** on the **Type** and **Status** columns.
4. Status filter defaults to **"Betalt"**.

Implementation note: rows are decorated on load with `typeLabel` and
`statusLabel` so both columns can be filtered with a simple EQUALS match against
Danish labels (this also makes "Andet" filterable, which a raw `ownerType`
EQUALS match could not do cleanly).

## Acceptance Criteria

- [x] Page title is "Ordrehistorik".
- [x] "Type" column present directly after "Ejer" with the 5 Danish labels.
- [x] Type and Status columns have inline row filters (`filterDisplay="row"`,
      `:showFilterMenu="false"`, `Select` with `showClear`).
- [x] Status filter is pre-set to "Betalt" on load.
- [~] `npm run lint` / `test:unit` — NOT run here (ui container unavailable).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 16:55 — Task created + picked up.
- 2026-07-31 17:05 — Rewrote `PaymentListView.vue`: title → "Ordrehistorik"; rows decorated on load with `typeLabel` (from `ownerType`) and `statusLabel` (from OPEN/PAID); added Type column after Ejer; `filterDisplay="row"` with `Select` filters on `typeLabel` and `statusLabel`; `filters` defaults `statusLabel` to "Betalt". Status Tag now uses `statusLabel`. Empty/loading text updated to "ordrer". Completed.
