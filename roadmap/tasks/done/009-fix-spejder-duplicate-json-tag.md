# 009 — Fix duplicate json tag on spejder.CurrentTeamID

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

`go vet ./...` fails: `nathejk/table/spejder/table.go:32` — `InitialTeamID`
and `CurrentTeamID` both use json tag `"teamId"`. This is a hard gate for the
dev loop and CI. Rename the `CurrentTeamID` json tag to `"currentTeamId"`.

Note: because the two fields currently share a tag, Go's json encoder emits
*neither* `teamId` field for `spejder` today; after the fix `teamId`
(InitialTeamID) and `currentTeamId` will both be emitted on the `/api/spejder`
response — strictly additive.

## Acceptance Criteria

- [x] `CurrentTeamID` json tag is `"currentTeamId"`.
- [x] `go vet ./...` no longer reports the duplicate-tag error (now fully clean).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 14:50 — Task created (follow-up flagged during PRD 002 work).
- 2026-07-31 14:52 — Picked up. Renaming the CurrentTeamID json tag.
- 2026-07-31 14:55 — Changed tag to `"currentTeamId"`. `go vet ./...` now fully clean; `go build ./...` OK. Completed.
