# 067 — Stop inventing member statuses in GetSpejdere

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] The `IF(... 'started', 'paid')` expression is gone from `internal/data/member.go`
- [x] `status` serves the projected value, empty (`MemberStatusNone`) when there is no row
- [x] `currentTeamId` selected, scanned and populated on the payload
- [x] `GET /api/patrulje/:id` exercised against the dev stack: members of a started
      patrol read `racing`, members of an unstarted one read empty
- [x] The Excel export still produces a file with a sane status column
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065; should ship in the same slice.
- 2026-08-17 — Picked up. Fallback deleted; `status` is now `COALESCE(ss.status, '')` and
  `currentTeamId` is `COALESCE(ss.currentTeamId, s.teamId)` — falling back to the signup
  team rather than to empty, because a member with no status row is still with their own
  patrol, and an empty team would make them invisible to every per-team query.
- 2026-08-17 — Kept the old expression in a doc comment on `GetSpejdere` rather than just
  deleting it. It is worth a reader's time *why* it was wrong: the values were not
  `MemberStatus` at all, so `Valid()`, `CanFinish()` and `InOurCare()` all returned false
  for them while the field was typed `types.MemberStatus` — and the `'started'` branch
  keyed on `patruljestatus.startedUts`, which is hardcoded to 1 on *signup*, so it did not
  even mean started.
- 2026-08-17 — **⚠️ Found that task 066's "exposed on the single-team payload" landed in a
  path nobody calls.** `GET /api/patrulje/:id` serves its team from
  `internal/data.TeamModel.GetPatrulje`, not from `table/patrulje`'s `GetByID` — so
  `activeMemberCount` was absent from the detail response despite the column being
  correct and the other querier being updated. This is precisely the failure mode PRD 006
  §8 warned about for `GetDiscontinuedTeamIDs` ("confirm which path is live at runtime
  before implementing, otherwise the fix lands in the unused one") — and I walked into it
  anyway, one task later. Added `activeMemberCount` **and** `signupStatus` to
  `data.Patrulje` and its query; the latter so a caller can tell a discontinued team from
  a never-started one without a second request.
- 2026-08-17 — ✅ Verified on the dev stack, both directions:
  - a `STARTED` 2025 patrol → `activeMemberCount: 4`, `signupStatus: 'STARTED'`, every
    member `racing`
  - a real 2026 patrol → `activeMemberCount: 0`, `signupStatus: ''`, members `''`
    (previously would have read `'paid'`), and it is **udgået under the naive rule but not
    under the corrected one** — the amendment from task 066 demonstrated against live data
    rather than argued.
- 2026-08-17 — ✅ All criteria met. Full `go build`, `go vet`, `gofmt -l`, `go test ./...`
  clean. Moving to done.
