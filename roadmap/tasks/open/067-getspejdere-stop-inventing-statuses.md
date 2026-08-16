# 067 — Stop inventing member statuses in GetSpejdere

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §6, §8 and §11 Decisions. **Status is set by messages** — no SQL may
synthesise one.

`go/internal/data/member.go:42`, behind `GET /api/patrulje/:id`
(`cmd/api/patrulje.go:85`), currently fabricates a status when no `spejderstatus` row
exists:

```sql
IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), ss.status) AS status,
```

Neither `started` nor `paid` is a valid `types.MemberStatus`, and `paid` is not even in
the legacy mapping documented in `types/member.go` — so the endpoint serves values no
lifecycle helper recognises, into a field typed `types.MemberStatus`.

Serve `ss.status`, and `MemberStatusNone` where there is no row yet.

Also: `data.Spejder.CurrentTeamID` exists with a `json:"currentTeamId"` tag but is never
selected or scanned, so it is always empty. Select and scan it now that the projection
maintains it.

## Notes

- **This must land with task 065, not after it.** The moment `spejderstatus` holds rows,
  every member's `status` on an existing endpoint changes value. Removing the fallback is
  what makes that change coherent rather than a mix of real and invented values.
- **Delete the fallback, do not keep it as a default.** Left in place, a member with no
  row keeps reading `paid` while `InOurCare()` is asked about them — the worst of both.
- Check the export path too: `cmd/api/export.go:88` calls the same `GetSpejdere`.
- Harmless on screen today only because `PatruljeView.vue:116` renders a hardcoded
  `"ikke startet"` tag and ignores the value entirely. Task 080 fixes that separately.
- Note `internal/data` is a legacy path that the shared-go migration has not finished
  retiring, and `shared-go/tables/spejder/querier.go:50` has the same LEFT JOIN without
  the invented fallback. Do not "fix" this by switching the handler to the shared entity
  in this task — that is a bigger change with its own blast radius.

## Acceptance Criteria

- [ ] The `IF(... 'started', 'paid')` expression is gone from `internal/data/member.go`
- [ ] `status` serves the projected value, empty (`MemberStatusNone`) when there is no row
- [ ] `currentTeamId` selected, scanned and populated on the payload
- [ ] `GET /api/patrulje/:id` exercised against the dev stack: members of a started
      patrol read `racing`, members of an unstarted one read empty
- [ ] The Excel export still produces a file with a sane status column
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065; should ship in the same slice.
