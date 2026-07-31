# 004 — BFF: include team orders in showPatruljeHandler

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `showPatruljeHandler` includes the team's orders in its response.
- [ ] Klan handling decided and implemented or explicitly deferred (logged).
- [ ] `go test ./...`, `go vet ./...`, `staticcheck ./...` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
