# 122 — `kortsaet` entity with optional `teamType`

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] `kortsaet` table with nullable `teamType`, no uniqueness constraint on it
- [x] Consumer handles `NATHEJK.*.kortsaet.*.created|updated|deleted`
- [x] Commands for create / update / delete / reorder, dirty-checked
- [x] Delete refused while the set holds maps, with a clear error
- [x] `POST /api/kortsaet`, `PUT /api/kortsaet/:id`, `PUT /api/kortsaet` (reorder),
      `DELETE /api/kortsaet/:id` — all with OpenAPI annotations
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: `kortsaet` lives in the same package as `kort` (one consumer, two
  tables, as `dispatch` does) since the two are always read together. Reorder follows the
  `NATHEJK.{year}.checkgroups.sorted` precedent — a collection-level subject with no id, which
  `live.Signal` already renders as "something of this type changed".
- 2026-09-03 — **Changed the PRD's endpoint table.** `PUT /api/kortsaet/sorted` cannot exist:
  httprouter panics at startup on a static segment beside a wildcard at the same level, so
  registering it next to `/api/kortsaet/:id` stops the *whole API* booting — not a misroute, not a
  404. Verified with a throwaway test before designing around it. `/api/checkgroups/sorted`
  escapes this only because `checkgroup`/`checkgroups` are two different segments, and Danish
  offers no such escape: "kort" and "kortsæt" are their own plurals, and inventing "kortsaets" to
  placate a router is worse than the alternative. Reordering is therefore `PUT` on the collection
  (`PUT /api/kortsaet`), and a map's order will live under its set (`PUT /api/kortsaet/:id/kort`)
  — which is also where handout order is actually meaningful. PRD §8 updated with the reasoning,
  because task 125's routes had the same latent panic.
- 2026-09-03 — Added `cmd/api/kort_routes_test.go`. A startup panic has no runtime symptom and no
  response to inspect, so this is the one thing here that genuinely needed a test rather than a
  comment. It also pins the consequence: a set *named* "sorted" stays an ordinary set reachable by
  id, which the rejected design could not express.
- 2026-09-03 — `SetUpdated` carries the **whole record**, the opposite of the sheet's patch-shaped
  `Updated`, and deliberately: a set has two editable fields and its editor always submits both,
  while a patch would make "clear the team type" and "do not touch the team type" the same nil
  pointer. Telling them apart would need a second boolean about one value, or a pointer to a
  pointer — both of which look fine and then silently refuse to let an operator un-mark the
  spejder set. Tested that clearing works.
- 2026-09-03 — `teamType` validated against `types.TeamTypes.Exists` rather than a local switch, so
  the vocabulary has one definition. My first attempt hardcoded `TeamTypeStaff`, which does not
  exist in this version of shared-go (it is `TeamTypeCrew`) — the build caught it, which is
  precisely the argument for not restating the list.
- 2026-09-03 — `teamType` is stored NULL, never `""`: NULL is the meaningful ordinary value (the
  crew set), while an empty string would be a matchable value inviting a caller to filter on it.
  An incoming `"teamType": ""` from an empty form select is normalised to nil.
- 2026-09-03 — Reorder applies as one `CASE` statement rather than a row-per-update loop: N
  statements would let a reader see an order that never existed on screen — two rows sharing a
  sortOrder, or a gap. Ids not named keep their position, so a per-set drag need not restate every
  other set.
- 2026-09-03 — Set deletion refuses rather than cascades, and the sheet count travels with the
  error so the operator is told what is in the way. Also left the consumer's `kortsaet.deleted`
  case free of any cascade — a cascade there would silently make the refusal pointless.
- 2026-09-03 — Completed. `go build ./...`, `go vet ./...`, `gofmt` clean; 20 tests pass across
  `nathejk/table/kort` and `cmd/api`. Wired into `data.Models.Kort` and `commands.KortSet`.
