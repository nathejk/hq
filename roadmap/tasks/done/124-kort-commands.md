# 124 — Kort commands with dirty-checking

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §8 (BFF). Depends on 121.

Write-side commands for maps: create, update (name, set, format, extents, note), delete,
replace checkpoint list, reorder.

Commands **dirty-check against the read model before publishing**, per `.rules`: a no-op
save publishes nothing and therefore emits no live signal, so an operator opening and
closing the modal does not make every other session refetch.

Two shapes worth care:

- **Extents** are a list of 0–2. Corners are normalised to a true north-west/south-east
  pair on save, whichever two corners the operator picked (PRD §5 edge cases).
- **Reorder** is one command over several maps, not N single-field updates — it is one
  operator gesture and should be one event, mirroring `sortCheckgroupsHandler`.

## Acceptance Criteria

- [x] Create / update / delete / set-checkpoints / reorder commands
- [x] No event published when nothing changed
- [x] Corner coordinates normalised to north-west + south-east on save
- [x] Reorder publishes one event for the whole set
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: mirror the set commands from task 122 — same commander, same publish
  helpers, same dirty-check discipline. Reorder is per set, following the route shape task 122
  landed on (`PUT /api/kortsaet/:id/kort`).
- 2026-09-03 — The dirty-check is **per field, not per request**: an `Updated` carrying eight fields
  when one moved would make a replay look like eight edits and would invalidate clients watching
  only one of them. So `Update` compares each field against the read model and omits what did not
  move.
- 2026-09-03 — Corner normalisation is where a subtle no-op bug would have lived, and the test
  worth having is the one that catches it: sending a rectangle back with its corners **swapped**
  must not count as an edit. Normalising before the dirty-check makes that true; normalising after
  would have published an event every time the UI round-tripped an unchanged extent.
- 2026-09-03 — Refused a **degenerate extent** (two clicks on the same latitude or longitude). It can
  only come from a mis-click, and a stored rectangle with no area draws as nothing — which an
  operator reads as "the save failed", then tries again.
- 2026-09-03 — `Create` takes only the set and the name, and deliberately does **not** check the set
  exists: replay materialises events in stream order, so a sheet may legitimately precede its set,
  and refusing here would enforce a constraint the projection does not hold.
- 2026-09-03 — `SetCheckpoints` is a **replace**, not add/remove. The UI is a set of tick-boxes, so
  "these ones" is the operator's actual intent; incremental events would let two concurrent editors
  interleave into a list neither of them chose. Ids are de-duplicated with order preserved, which
  keeps re-saving an unchanged selection a genuine no-op.
- 2026-09-03 — Did not validate that checkpoint ids exist. Task 123 already filters unresolvable ids
  on read, which also covers a checkpoint deleted *after* the save, so a check here would duplicate
  it and cost a read on every save.
- 2026-09-03 — Moving a sheet between sets is an `Update` of its `kortsaetId`, not a reorder. Keeping
  them separate means neither operation has to infer the other's intent.
- 2026-09-03 — Completed. `go build ./...`, `go vet`, `gofmt` clean; 41 tests pass in
  `nathejk/table/kort`. HTTP wiring for these is task 125.
