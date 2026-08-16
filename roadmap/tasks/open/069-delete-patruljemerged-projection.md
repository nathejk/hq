# 069 — Delete the patruljemerged projection

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `go/nathejk/table/patruljemerged/` deleted
- [ ] Import and wiring removed from `cmd/api/main.go`
- [ ] Table dropped (dev; noted for stage/prod, which are cleared before deploy)
- [ ] The two unreachable readers identified in the log, not silently left undocumented
- [ ] `go build ./... && go vet ./...` clean
- [ ] API starts and replays with the projection gone

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Best done after task 066, so discontinuation is
  expressible before the old mechanism is removed.
