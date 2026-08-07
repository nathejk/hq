# 022 — Remove dead /api/payments endpoint

**Status:** done
**Priority:** medium
**Created:** 2026-08-07
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-08-07
**Completed:** 2026-08-07

## Description

`GET /api/payments` (and its `listPaymentsHandler`) is dead: the Betalinger /
Ordrehistorik page now reads `/api/orders` (tasks 006/016/017), and no other
frontend code calls it. It was also the only caller of the year-wide
`payment.Filter{Year: ...}` read, which the shared-go `payment` entity cannot
express — so removing it is a prerequisite for adopting shared `payment`
(task 021 deferred it for exactly this reason).

Removed:
- `go/cmd/api/payment.go` (the file contained only this handler)
- its route registration in `go/cmd/api/routes.go`

## Acceptance Criteria

- [x] `/api/payments` route and handler removed.
- [x] No remaining year-wide `payment.Filter{Year: ...}` usage.
- [x] `go build`/`go vet`/`go test`/`staticcheck`/`gofmt` all green.

## Notes / follow-up

The last remaining `payment.Filter` user is `cmd/api/patrulje.go:89`
(`Filter{TeamIDs: [teamId]}`), feeding a `payments` field on `/patrulje/:id`
that the frontend also no longer reads (task 007 switched that section to
orders). Shared payment *can* express this as `GetAll(teamID)`, so it is not a
blocker — but dropping it would remove the last Filter dependency entirely.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-07 11:20 — Verified `/api/payments` is unused by the frontend, then removed the route + handler file. build/vet/test/staticcheck/gofmt all clean. Completed.
