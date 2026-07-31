# 004 — BFF: include team orders in showPatruljeHandler

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). The Patrulje detail
payload (`showPatruljeHandler` in `go/cmd/api/patrulje.go`) currently returns
raw `payments` for the team. Add the team's **orders** to the JSON envelope so
the frontend (task 007) can show orders instead.

- Fetch orders via `order.Queries.ListByOwner(year, "patrulje", teamId)`.
- Add them to the response envelope (e.g. `"orders": orders`) alongside (or
  instead of) `payments`.
- Decide whether `showKlanHandler` (`go/cmd/api/klan.go`) should get the same
  treatment for klan orders — the PRD leaves owner scope open; if klan orders
  are in scope, mirror the change with `ListByOwner(year, "klan", teamId)`.

Depends on: 002 (wiring + `data.Models`).

## Acceptance Criteria

- [x] `showPatruljeHandler` includes the team's orders in its response
      (`"orders"`, via `Order.ListByOwner(year, patrulje, teamId)`; `payments`
      kept for now so nothing breaks before task 007).
- [x] Klan handling decided: **deferred** — there is no klan *detail* view yet
      (only `KlanListView`), and Betalinger owner-scope is still open, so klan
      orders have no frontend home. Logged; trivially mirrored later.
- [x] `go build`/`go vet` clean for `./cmd/api/...`.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 13:45 — Picked up.
- 2026-07-31 13:50 — Added `Order.ListByOwner(year, TeamTypePatrulje, teamId)` to `showPatruljeHandler` and included `"orders"` in the envelope (kept `"payments"` too). Klan deferred (no klan detail view; owner-scope open). build + vet clean for ./cmd/api/... Completed.
