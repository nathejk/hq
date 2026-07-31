# 011 — Clean up pre-existing staticcheck findings

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

## Description

Task 010 added `staticcheck` as a `go.mod` tool directive and it is a dev-loop
gate (`go tool staticcheck ./...`). staticcheck reports **27 pre-existing
findings** across the codebase (none in the order/product/payments work). Until
these are resolved (or explicitly ignored), the staticcheck gate is red, which
per the dev-loop contract prevents the binary from restarting on change.

Findings (from `go tool staticcheck ./...`):

- `cmd/api/ctrlgrpcmd.go:38` ST1005 (capitalized error string)
- `cmd/api/mail.go:36` SA4010 (append result never used)
- `cmd/api/signup.go:78,113` U1000 (unused handlers `sendMobilepaySmsHandler`, `confirmSignupHandler`)
- `nathejk/commands/checkgroup.go:82` SA4009 (ctx overwritten before use), `:84,:102` ST1005
- `nathejk/table/checkgroup/commands.go:130` ST1005
- `nathejk/table/checkgroup/handler.go:13,101,103` U1000 (unused types)
- `nathejk/table/checkgroup/query.go:76,95` SA4006 (ctx value never used)
- `nathejk/table/checkpersonnel/commands.go:51` ST1005
- `nathejk/table/{checkpersonnel,checkpoint,klan,lok,year}/filter.go` U1000 (`calculateMetadata` unused — duplicated across packages)
- `nathejk/table/checkpoint/query.go:26`, `nathejk/table/payment/queries.go:25`, `nathejk/table/year/query.go:23` SA4006 (ctx value never used)
- `nathejk/table/scan/consumer.go:14` U1000 (unused field `c`)
- `nathejk/table/year/handler.go:13,101,103` U1000 (unused types)

Note: the `SA4006`/`SA4009` ctx findings may indicate real (if benign) bugs —
review those rather than blindly silencing. The `U1000`/`ST1005` ones are
cosmetic/dead-code and safe to fix or remove.

## Acceptance Criteria

- [ ] `go tool staticcheck ./...` is clean (or documented, intentional
      `//lint:ignore` directives where a finding is a deliberate false positive).
- [ ] No behavioural regressions from the ctx-related fixes.
- [ ] `go build`/`go vet`/`go test ./...` stay green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 15:20 — Task created as a follow-up to task 010 (enabling the staticcheck gate surfaced 27 pre-existing findings).
