# 121 — `nathejk/table/kort` projection and local message types

**Status:** doing
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:**

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

- [ ] `kort` table created from an embedded `table.sql`
- [ ] Consumer handles created/updated/deleted
- [ ] Local message types, free of HQ-only types
- [ ] Registered in the `projections` slice in `main.go`
- [ ] `go build ./...` and `go vet ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: model on `nathejk/table/spejdernote` (the most recent projection
  shape — local `messages.go`, `table.go`, `consumer.go`, `querier.go`, `table.sql`,
  `schemaMigrations`), and on `dispatch` for the JSON-array-in-TEXT precedent
  (`dispatch_task.memberIds`). Kort events only in this task; the `kortsaet` table and the
  reorder event are 122, the delete cascades 123, commands 124.
