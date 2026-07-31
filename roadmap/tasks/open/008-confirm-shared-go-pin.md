# 008 — Confirm shared-go version pin includes order/payment messages

**Status:** open
**Priority:** low
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Pinned shared-go version in `go/go.mod` includes the order + payment
      messages used by the order projection.
- [ ] A `GOWORK=off` build (as in CI/Docker) compiles the order code.
- [ ] If a bump was needed, `go.mod`/`go.sum` updated and noted here.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 00:00 — Task created from PRD 002.
