# 012 — Remove obsolete signup handlers

**Status:** done
**Priority:** low
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Follow-up to task 011. The two handlers preserved with `//lint:ignore U1000`
directives — `sendMobilepaySmsHandler` and `confirmSignupHandler` in
`go/cmd/api/signup.go` — are confirmed obsolete. Remove them (and any imports
that become unused).

## Acceptance Criteria

- [x] Both handlers removed.
- [x] No now-orphaned imports left behind (build clean; other handlers still use `log`/`fmt`/`types`/etc.).
- [x] `go build`, `go vet`, `go test ./...`, and `go tool staticcheck ./...` stay green (staticcheck 0).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 16:00 — Task created after user confirmation that both handlers are obsolete.
- 2026-07-31 16:05 — Removed `sendMobilepaySmsHandler` and `confirmSignupHandler` (and their `//lint:ignore` directives) from `cmd/api/signup.go`. No imports orphaned. build/vet/test clean; staticcheck ./... = 0. Completed.
