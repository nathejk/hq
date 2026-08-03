# 017 — Ordrehistorik summary (order count + line-item counts by size)

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Add a summary block at the top of the Ordrehistorik page
(`vue/src/views/PaymentListView.vue`) showing:

- how many orders are currently **shown** (i.e. after the Type/Status filters), and
- a list of the **line items** across those orders with their counts, where
  **merchandise (t-shirts) is counted per size**.

Backend prerequisite: `GET /api/orders` (`order.ListByYear`) deliberately omitted
line items to keep the year-wide list light (task 001). The summary needs them,
so `ListByYear` now hydrates lines — but via a **single** extra query that fetches
all of the year's `order_line` rows joined to `orders` and groups them by
`orderId`, rather than the per-order N+1 fetch that `ListByOwner` uses.

Size comes from the order line's `attributes` map (e.g. `attributes.size`), which
is where the chosen product variant is recorded.

## Acceptance Criteria

- [x] `ListByYear` returns orders with their line items (single grouped query via
      new `linesForYear`, no N+1).
- [x] Summary shows the count of orders currently shown (respects filters, via a
      `shownOrders` computed that also drives the table).
- [x] Summary lists line items with counts; sized merchandise (t-shirts) split
      per size (label `Produkt (SIZE)`), summing `quantity`.
- [x] `go build`/`go vet`/`staticcheck` green.
- [~] `npm run lint` / `test:unit` — NOT run here (ui container unavailable).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 17:15 — Task created + picked up. Reverses the task-001 "no lines in year list" decision, but with a grouped single query so it stays cheap.
- 2026-07-31 17:25 — Backend: added `linesForYear` (one query joining `order_line` to `orders` for the year, grouped by orderId) and hydrated `ListByYear` from it; updated the `Queries` doc comment. build/vet/staticcheck clean.
- 2026-07-31 17:35 — Frontend: added `shownOrders` computed (applies Type/Status filters) used as the DataTable value AND the summary source so they can't disagree; added `lineSummary` computed aggregating `quantity` per `productName` + size (size read from line `attributes.size`/`tshirtSize`); added the summary block above the table ("N ordrer vist" + wrapped label:count list). Also dropped the now-redundant lazy per-row line fetch (`linesByOrder`/`onRowExpand`) since lines ship with the list, and added a "Størrelse" column to the expansion. Completed.
