# 114 — task dialog with the place picker

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §7. `components/DispatchTaskDialog.vue` — create and edit a task.

Fields: kind (`pickup` / `transport` / `collection` / `delivery`, defaulting the places
differently), pick-up and drop-off place, description, priority (grøn/gul/rød via task 112),
space needs in words, `tidligst` and `skal leveres`.

**Place picker**: one control offering checkpoints, loks and HQ as groups and accepting free
text — free text is the normal case for "på Slangerupvej ved skovbrynet", not a fallback.
`composables/personnelTree.ts`'s grouped picker is the model.

Almost everything is optional: the mitigation for the desk-discipline risk (§8) is that the
written path is the fastest path.

## Acceptance Criteria

- [ ] Dialog creates and edits tasks against `POST`/`PATCH /api/dispatch/task`
- [ ] Kind selection defaults the places sensibly (delivery from HQ, collection to HQ, …)
- [ ] Place picker offers grouped checkpoints/loks/HQ and accepts arbitrary text
- [ ] Priority uses the shared severity vocabulary
- [ ] Only kind and a description are required
- [ ] Danish labels; day-and-time entry for `tidligst` / `skal leveres`

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
