# 124 — Kort commands with dirty-checking

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Create / update / delete / set-checkpoints / reorder commands
- [ ] No event published when nothing changed
- [ ] Corner coordinates normalised to north-west + south-east on save
- [ ] Reorder publishes one event for the whole set
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
