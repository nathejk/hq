# 138 — Lift the kort message types to `nathejk/shared-go`

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §8 ("How the maps leave HQ"), and the correction note at the top of that PRD.

The kort events are published on the stream but their Go types live inside this repo
(`go/nathejk/table/kort/messages.go` and `kortsaet_messages.go`). **A consumer cannot decode events
it has no types for**, so this blocks the hej-app entirely — it is the only remaining thing between
PRD 010 and the feature being useful outside HQ.

Starting local was right while the shape settled, and the shape has now been exercised end to end.
Move the message types (not the projection) to `nathejk/shared-go/messages`:

- `Created`, `Updated`, `Deleted`, `Sorted` (sheets)
- `SetCreated`, `SetUpdated`, `SetDeleted`, `SetsSorted` (sets)
- `Format`, `Extent`, and the `KortID` / `KortsaetID` types

They were written for this move: no HQ-only types, only local declarations and `shared-go/types`.
Name them for a shared namespace on the way (`NathejkKortCreated`, … matching the existing
`NathejkCheckpointUpdated` convention) rather than keeping the bare local names.

The projection stays here for now. Nothing else wants HQ's read model, and moving it would be a
larger change with no caller (tasks 055 and 083 are the same kind of lift, still open).

**Do not change the wire shapes while moving them.** Two properties in particular are relied on by
`roadmap/api/kort-events.md`, which is already circulated:

- a sheet's `Updated` is a **patch** — pointer fields, absent means unchanged, and a pointer to an
  empty slice means "cleared";
- a set's `SetUpdated` is a **whole record**, so an absent `teamType` means "has none".

## Acceptance Criteria

- [ ] Message types in `nathejk/shared-go/messages`, named for the shared namespace
- [ ] `nathejk/table/kort` imports them; no duplicate local definitions left behind
- [ ] Wire shapes byte-identical — a replay of existing events still materialises correctly
- [ ] `go build ./...`, `go vet ./...`, and the kort tests pass
- [ ] `roadmap/api/kort-events.md` updated with the shared-go import path, and its task-138
      prerequisite note removed
- [ ] hej-app team told it is unblocked

## Progress Log

- 2026-09-03 — Task created. Promoted from "eventually" to blocking by the correction to PRD 010:
  cross-service communication is over the stream, so shared types are the integration point rather
  than a tidy-up.
