# 043 — SOS projections: sos, sos_team, sos_activity

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §8. The read side: three MySQL projections in
`go/nathejk/table/sos/`, written to shared-go's guidelines
(`shared-go/tables/signup` is the layout reference: `table.go`, `consumer.go`,
`querier.go`, `commands.go`, `repository.go`, `interfaces.go`, `table.sql`).

Tables:

- **`sos`** — `id, year, headline, description, createdAt, createdBy, status,
  severity, assigneeSectionSlug, lastActivityAt, deletedAt`. `lastActivityAt` is
  maintained on **every** event for the case, because it drives the list's "Sidst
  opdateret" column and its default sort. `deletedAt` implements the soft delete; every
  read path filters it out.
- **`sos_team`** — case↔patrol association (legacy `sosassoc`), idempotent insert.
- **`sos_activity`** — `sosId, activityId, seq/createdAt, type, actorUserId, value,
  status, comment, refActivityId`. The whole timeline. `refActivityId` is what makes a
  comment edit append rather than overwrite. The `type` column and payload must
  accommodate PRD 006's member entry types without a schema change.

Queries needed: list by year grouped open/closed ordered by `lastActivityAt` desc; one
case with timeline + associated teams; cases by team (for task 048).

## Notes

- `CREATE TABLE IF NOT EXISTS` never alters an existing table, so get the shape right
  up front; in dev the table must be dropped to pick up a column added later.
- Depends on task 042 for the event bodies.

## Acceptance Criteria

- [ ] Three tables created on startup via `//go:embed table.sql`
- [ ] Consumer handles every SOS subject and is idempotent on replay
- [ ] `lastActivityAt` advances on every event for the case
- [ ] Soft-deleted cases are excluded from list and single-case reads
- [ ] Queries: list, single-with-timeline, by-team
- [ ] No `nathejk.dk/...` import in the package
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
