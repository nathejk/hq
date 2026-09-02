# 128 — Checkpoint assignment UI with per-checkgroup select-all

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §6, §7. Depends on 127. **This is the valuable half of the feature** — the
checkpoint-to-map relation is what the hej-app needs; extents are cosmetic by comparison.

In the settings dialog, assign checkpoints to the selected map. Grouped by checkgroup,
matching the existing context-menu grouping in `KortView.vue`, with per-group select-all —
that is what keeps entry to minutes rather than an hour (PRD §8, data entry burden).

Selecting a map highlights exactly its checkpoints on the map and fades the rest, so a
mistake is visible rather than merely saved.

A checkpoint may belong to any number of maps, including several in one set — adjacent
sheets overlap by design, so this is never flagged.

Also list checkpoints belonging to **no** map in the selected set, so one cannot be
forgotten silently. A checkpoint with no position can still be assigned; it just cannot be
drawn.

## Acceptance Criteria

- [ ] Checkpoints selectable per map, grouped by checkgroup, with per-group select-all
- [ ] Saved via `PUT /api/kort/:id/checkpoints`
- [ ] Selected map's checkpoints highlighted, others faded
- [ ] "Ikke på noget kort" list for the selected set, clicking through to the map
- [ ] Positionless checkpoints assignable and flagged, not blocked
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
