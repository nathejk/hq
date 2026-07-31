# 011 — Clean up pre-existing staticcheck findings

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] `go tool staticcheck ./...` is clean (0 findings).
- [x] No behavioural regressions from the ctx-related fixes (ctx now correctly
      threaded into the queries; equivalent behaviour, better cancellation).
- [x] `go build`/`go vet`/`go test ./...` stay green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 15:20 — Task created as a follow-up to task 010 (enabling the staticcheck gate surfaced 27 pre-existing findings).
- 2026-07-31 15:30 — Picked up. Plan: ST1005 → lowercase error strings; U1000 → remove dead funcs/types/fields; SA4006/SA4009 → actually thread `ctx` into the queries (real, if benign, bugs); SA4010 → fix discarded append. Re-run `go tool staticcheck ./...` to confirm clean.
- 2026-07-31 15:50 — Fixes applied:
  - ST1005: lowercased error strings in ctrlgrpcmd.go, commands/checkgroup.go (x2), table/checkgroup/commands.go, table/checkpersonnel/commands.go.
  - SA4006: threaded ctx via QueryRowContext/ScanStructsContext in checkgroup, checkpoint, year queries and payment GetAll.
  - SA4009: commands/checkgroup.go Update now derives from the incoming ctx (`requestctx.WithValue(ctx, ...)`) instead of `context.Background()`.
  - SA4010: removed dead `klanRecipients` construction + unused klans fetch and `klan` import in cmd/api/mail.go (response behaviour unchanged).
  - U1000: removed unused field `scan/consumer.go` `c`; removed unused types `response`/`scans`/`cgCount` in checkgroup/handler.go and year/handler.go; removed the duplicated unused `calculateMetadata` (+ now-unused `math` import) from checkpersonnel/checkpoint/klan/lok/year filter.go; and added `//lint:ignore U1000` (with reason) to the two intentional-but-unwired signup handlers rather than deleting real logic.
- 2026-07-31 15:55 — `go tool staticcheck ./...` = 0 findings; `go vet ./...`, `go test ./...`, gofmt all clean. Completed.
