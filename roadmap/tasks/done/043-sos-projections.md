# 043 — SOS projections: sos, sos_team, sos_activity

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

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

- [x] Three tables created on startup via `//go:embed table.sql`
- [x] Consumer handles every SOS subject and is idempotent on replay
- [x] `lastActivityAt` advances on every event for the case
- [x] Soft-deleted cases are excluded from list and single-case reads
- [x] Queries: list, single-with-timeline, by-team
- [x] No `nathejk.dk/...` import in the package
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up. Plan: `table.sql` + `table.go` + `consumer.go` + `querier.go` +
  `filter.go`, following `table/checkgroup` for the goqu consumer style and
  `table/order` for multi-write handlers.
- 2026-08-11 — **Timeline entries are keyed by the event's stream sequence**
  (`msg.Sequence()`), not a generated id. This is the decision the whole projection
  rests on: the stream is replayed on every API restart, so a generated key would
  duplicate the entire timeline each time, and a sequence also gives the timeline a
  total order that agrees with the log. Verified by a test that handles the same
  message twice.
- 2026-08-11 — Checked that a three-statement `table.sql` is safe: `DB_DSN` carries
  `multiStatements=true` (`docker-compose.yml:65`) and `table/order` already ships a
  two-statement schema whose second table exists in the running database.
- 2026-08-11 — **Timestamps stored as UTC DATETIME.** The driver reads DATETIME back as
  UTC (`parseTime=true`, default loc), so the API emits a Z-offset timestamp and the
  browser converts to the operator's clock. Storing local wall-clock would have made
  every entry two hours wrong in summer — in a log whose only job is establishing when
  things happened.
- 2026-08-11 — `GetByID` uses three queries rather than one join: a join multiplies the
  case row by timeline entries × teams, and the timeline is unbounded where the case is
  not. It is one case opened by one operator, so the round trips are free.
- 2026-08-11 — Wrote `consumer_test.go` against a recording writer (the repo's existing
  style: assert on emitted SQL). Six tests: create writes case + entry, **every** event
  advances `lastActivityAt` and appends an entry, replay is idempotent, association is
  idempotent, delete is soft, and an unknown subject is a logged no-op rather than an
  error — the last one matters because PRD 006 adds subjects to this domain.
- 2026-08-11 — Three test assertions were wrong, not the code: goqu's mysql dialect
  emits `INSERT IGNORE ... ON DUPLICATE KEY UPDATE` rather than `INSERT INTO`, which
  `table/checkgroup` already relies on in production. Loosened the assertions to the
  table name plus the conflict clause.
- 2026-08-11 — Deferred the `commander` to task 044 rather than stubbing it, so
  `table.go` compiles with only the read side; `New` already takes the publisher so the
  wiring in `cmd/api` will not change twice. Also moved the assignable-section table to
  task 045 where it belongs.
- 2026-08-11 — ✅ All criteria met: `go build ./...`, `go vet`, `gofmt -l` and
  `go test ./nathejk/table/sos/` all clean. Moving to done.
