# 121 — `nathejk/table/kort` projection and local message types

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §8 (BFF, Data / storage). Phase 1 — the foundation every other task builds on.

Add `go/nathejk/table/kort/` following the shape of `go/nathejk/table/checkpoint/`:
`table.go`, `consumer.go`, `query.go`, `command.go`, `table.sql`, `filter.go`.

The `kort` table:

| column | notes |
|---|---|
| `id`, `year`, `version` | as every other projection |
| `kortsaetId` | the set it belongs to (task 122) |
| `name`, `format`, `note`, `sortOrder` | `format` in `a4`/`a3`/`skitse`/`andet` |
| `checkpointIds` | JSON array — deliberately **not** a join table, see PRD §8 |
| `extents` | JSON array of 0–2 `{northWest, southEast}` lat/lng objects |

Event message types start **local to this repo**, not in `nathejk/shared-go` — they move
independently of the projection once the hej-app needs them (PRD §8). Carry no HQ-only
types so that move stays a package move.

Consumes `NATHEJK.*.kort.*.created|updated|deleted`. The delete cascades from checkpoint
and checkgroup are task 123.

Register the consumer in the `projections` slice in `cmd/api/main.go` — a consumer added
to the mux outside that slice emits no live signal at all (`.rules` → Live updates).

## Acceptance Criteria

- [x] `kort` table created from an embedded `table.sql`
- [x] Consumer handles created/updated/deleted
- [x] Local message types, free of HQ-only types
- [x] Registered in the `projections` slice in `main.go`
- [x] `go build ./...` and `go vet ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: model on `nathejk/table/spejdernote` (the most recent projection
  shape — local `messages.go`, `table.go`, `consumer.go`, `querier.go`, `table.sql`,
  `schemaMigrations`), and on `dispatch` for the JSON-array-in-TEXT precedent
  (`dispatch_task.memberIds`). Kort events only in this task; the `kortsaet` table and the
  reorder event are 122, the delete cascades 123, commands 124.
- 2026-09-03 — `Updated` carries **pointers to slices** for `checkpointIds` and `extents`, not
  plain slices. The distinction is load-bearing: `nil` means "this event does not mention
  checkpoints" and leaves the column alone, while a pointer to an empty slice means "this sheet
  now has none" — a real edit an operator will make, and one a plain nil slice could not express
  at all. Tested both ways.
- 2026-09-03 — The `created` upsert names only `kortsaetId` and `name` in its update list, so a
  replay cannot undo the format, note, extents and checkpoints that later events put on the row.
  Replay order is therefore irrelevant, which is the only way this table rebuilds safely. Guarded
  with a test that fails if anyone adds the JSON columns to that list.
- 2026-09-03 — Empty JSON columns are written as `[]`, never `null`, and seeded at create time
  rather than left to the column default. The hej-app parses this, and `null` would put a
  special case in every reader. `scanKort` also tolerates malformed JSON by yielding an empty
  list: a map whose extents will not parse is still a map with a name and a checkpoint list, and
  failing the read would take the whole settings screen down over one row — and make that row
  unfixable from the UI.
- 2026-09-03 — Chose `kort` as **its own subject and live token**, unlike spejdernote which rides
  on the scout's subject to reuse an existing token. A map is not a fact about another entity:
  nothing else's screen needs to refetch when a sheet is renamed.
- 2026-09-03 — Added `Kort.ContainsAll`, the primitive task 133's warning needs, with the
  existential semantics spelled out in a test — *some* map must hold the whole checkgroup, so two
  overlapping sheets that both hold it are fine. Wrote it here because the same reasoning that
  makes the JSON column correct (no checkpoint→maps query) is what makes this a method on the row
  rather than SQL.
- 2026-09-03 — `schemaMigrations` present and empty from the start, per `.rules`: task 122 adds
  the set table and this table already declares `kortsaetId`, so the window where a later column
  goes silently missing opens immediately.
- 2026-09-03 — Completed. `go build ./...`, `go vet`, `gofmt -l` clean; 9 tests pass in
  `nathejk/table/kort`. Registered in `projections` so `live.NotifyAll` signals it and `kort`
  appears in the advertised token set for task 126 to depend on.
