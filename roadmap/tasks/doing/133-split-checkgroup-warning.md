# 133 — Warn when no single map covers a whole checkgroup

**Status:** doing
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:**

## Description

PRD 010 §6, §8 ("A checkgroup must fit on one map"). Depends on 128.

A checkgroup is revealed as a whole, so if no single sheet in a set covers all of its
checkpoints, a patrol is shown checkpoints it holds no map for. That is a printing mistake
with a race-day cost, and it is cheap to detect.

Two things about the test that are easy to get wrong:

- It is **existential, not partitioning**: *some* map in the set must contain the whole
  group. Two overlapping sheets that both contain it are fine — overlap is deliberate, so a
  partitioning test would fire constantly and be ignored.
- It is about **map membership, never geometry**. A checkgroup's checkpoints may legitimately
  sit in two different areas of the same sheet — that is what a double-sided A3's two extents
  are for — so comparing positions would false-alarm on every one of them.

A **warning, never a block**: a half-entered set trips it constantly, and a save that
refuses to complete during data entry is worse than a visible warning.

## Acceptance Criteria

- [ ] Warning per set, naming the checkgroup and the maps its checkpoints are spread across
- [ ] Satisfied when any one map contains the whole checkgroup, even if others partly do
- [ ] Based on map membership, not coordinates
- [ ] Never blocks a save
- [ ] Offending checkpoints highlighted on the map
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. `someMapContainsAll` already exists and is tested from task 126, with the
  existential semantics pinned. What remains is the presentation: which set, which checkgroup, which
  sheets it is spread across.
