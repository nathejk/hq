# 008 — Confirm shared-go version pin includes order/payment messages

**Status:** done
**Priority:** low
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

From PRD 002 (`roadmap/prd/002-order-based-payments.md`). The order feature
relies on `github.com/nathejk/shared-go/messages` (`order.go`, `payment.go`).
In dev these resolve via `go.work` from the sibling `../shared-go` checkout, but
prod/CI build with `GOWORK=off` against the version pinned in `go/go.mod`.

Verify the pinned shared-go version in `go/go.mod` actually contains the order
message structs (`NathejkOrderCreated`, `NathejkOrderLinesChanged`,
`NathejkOrderPaid`, `NathejkOrderCancelled`) and the payment fields
(`orderForeignKey`, `orderType`). If not, cut/release a shared-go version and
bump the `go.mod` require so the Docker/CI build compiles without the workspace.

## Acceptance Criteria

- [x] Pinned shared-go version in `go/go.mod` includes the order + payment
      messages used by the order projection.
- [x] A `GOWORK=off` build (as in CI/Docker) compiles the order code.
- [x] No bump needed — pin already sufficient (documented below).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
- 2026-07-31 14:10 — Verified. Pinned version is `github.com/nathejk/shared-go v0.0.0-20260702212401-45a8aa6c7daf`. Its module cache dir contains `messages/order.go` (NathejkOrderCreated/LinesChanged/Cancelled/Paid) and `messages/payment.go` with `OrderForeignKey`/`OrderType`. `GOWORK=off go build ./nathejk/table/order/...` succeeds against the pin (repo has no go.work anyway). No bump required. Completed.
