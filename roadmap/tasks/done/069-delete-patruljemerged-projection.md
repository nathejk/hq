# 069 — Delete the patruljemerged projection

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §4, §6 and §11 Decisions. **The merged-team concept is deprecated** —
not just its events, the concept. Teams are not merged and split; members are moved, and a
team with no active members is thereby discontinued.

Delete:

- `go/nathejk/table/patruljemerged/` (package, consumer, `table.sql`)
- its import and wiring in `cmd/api/main.go:51`, `:175`, `:266`
- the `patruljemerged` table itself

The legacy consumer subscribed to `NATHEJK:*.patrulje.*.merged` and
`NATHEJK:*.patrulje.*.splited`. **Neither event is being produced any more**, and neither
is ported: `.merged` inserted a `teamId → parentTeamId` row and `.splited` deleted it,
which is how discontinuation used to be recorded and undone. `activeMemberCount`
(task 066) gets that reversibility for free by being recomputed rather than set.

## Notes

- **Two dead readers remain and are deliberately not fixed here.**
  `table/checkgroup/handler.go:48` and `table/year/handler.go:48` both run
  `SELECT DISTINCT m.teamId FROM patruljemerged m JOIN patruljestatus s ...`. Both sit
  behind `GET /api/cgstatus`, which is **commented out** at `routes.go:91`, so both are
  unreachable. Deleting the table leaves them referencing a table that no longer exists.
  That is acceptable — they are already dead code — but note it in the log so whoever
  revisits checkgroup is not surprised. Everything checkgroup-related is explicitly out of
  scope for PRD 006 (§4).
- `patrulje.querier.GetDiscontinuedTeamIDs` (`table/patrulje/query.go:122`) returns an
  empty slice and **has no callers**. Leave it alone — it is not part of this PRD (§4), and
  removing it is a separate cleanup.
- Historical `patruljemerged` rows encode past discontinuations the new model cannot
  reconstruct, since there are no per-member move events behind them. Legacy data
  migration is out of scope: they are dropped. Confirm this is understood before deleting
  the table on stage.

## Acceptance Criteria

- [x] `go/nathejk/table/patruljemerged/` deleted
- [x] Import and wiring removed from `cmd/api/main.go`
- [x] Table dropped (dev; noted for stage/prod, which are cleared before deploy)
- [x] The two unreachable readers identified in the log, **and dealt with** rather than left
      pointing at a table that no longer exists — see log
- [x] `go build ./... && go vet ./...` clean
- [x] API starts and replays with the projection gone

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Best done after task 066, so discontinuation is
  expressible before the old mechanism is removed.
- 2026-08-17 — Picked up. Package, wiring and table all gone.
- 2026-08-17 — **Went further than "flag the dead readers", because leaving them was worse
  than the alternative.** The task proposed noting the two `SELECT ... FROM patruljemerged`
  readers and moving on. But shipping code that queries a dropped table is a landmine, so:
  - `go/nathejk/table/year/handler.go` — **deleted outright.** It turned out to be
    byte-identical to `checkgroup/handler.go` apart from the package clause, and
    `year.NewControlgroupStatusHandler` is referenced **nowhere at all**, not even from a
    comment. A dead copy of a dead file; removing it is cleanup of this deletion, not
    checkgroup work.
  - `go/nathejk/table/checkgroup/handler.go` — kept (checkgroup is out of scope, §4) but its
    `inactiveTeamIDs` closure now returns empty, with the old query preserved in a comment
    alongside the replacement predicate and a warning that the "and started" half is not
    optional. It is reachable only from the commented-out `/api/cgstatus` route, so this
    changes no behaviour — it just means whoever revives that screen finds a signpost
    instead of a broken query.
- 2026-08-17 — The dropped table held **0 rows**, which settles the open question in PRD 006
  §11 about migrating historical `patruljemerged` data: there is none to migrate. Closed
  rather than answered.
- 2026-08-17 — Three log lines mentioning `patruljemerged` after the restart looked alarming
  but were hot-reload build output from the transient moment mid-deletion ("cannot find
  module providing package …"), not the running process. Confirmed by checking the live API
  afterwards rather than trusting the grep.
- 2026-08-17 — ✅ Verified: healthcheck `available`, zero `Error consuming` in the last
  minute, and the read model still reconciles at **686 = 686** after the drop and a full
  replay. `go build`, `go vet`, `gofmt -l`, `go test ./...` all clean.
- 2026-08-17 — ✅ All criteria met. Moving to done.
