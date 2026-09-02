# 122 — `kortsaet` entity with optional `teamType`

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §6, §8 ("Sets are an entity, because `teamType` lives on them"). Depends on task 121.

A map set is its own entity, not a string on each map, because `teamType` is a property of
the set as a whole. Table `kortsaet`: `id`, `year`, `version`, `name`, `teamType`
(**nullable**), `sortOrder`.

`teamType` reuses `TeamType` from `nathejk/shared-go` (`patrulje`, `klan`). Read it as
*"this set is specifically for this team type"*, **not** *"only this team type uses it"* —
an unmarked set is the general one, and klaner draw from it. Consequences that must hold:

- Nullable and stays nullable — the crew set has no team type.
- **Not unique.** Several sets may carry the same `teamType`; it is a filter yielding
  candidate maps, not a key. Do not add a constraint.
- Filtering by `klan` will usually return nothing, and that is not an error.

Sets live in the same projection package as `kort` (one consumer, two tables) since they
are always read together.

Deleting a set is **refused while it still holds maps** — a mis-click must not cost a
season's definitions.

## Acceptance Criteria

- [ ] `kortsaet` table with nullable `teamType`, no uniqueness constraint on it
- [ ] Consumer handles `NATHEJK.*.kortsaet.*.created|updated|deleted`
- [ ] Commands for create / update / delete / reorder, dirty-checked
- [ ] Delete refused while the set holds maps, with a clear error
- [ ] `POST /api/kortsaet`, `PUT /api/kortsaet/:id`, `PUT /api/kortsaet/sorted`,
      `DELETE /api/kortsaet/:id` — all with OpenAPI annotations
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
