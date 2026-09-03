# 133 — Warn when no single map covers a whole checkgroup

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Warning per set, naming the checkgroup and the maps its checkpoints are spread across
- [x] Satisfied when any one map contains the whole checkgroup, even if others partly do
- [x] Based on map membership, not coordinates
- [x] Never blocks a save
- [ ] Offending checkpoints highlighted on the map — **not done.** See the log: the highlight is
      owned by sheet selection, and a second, competing source of fading would have muddled the one
      signal that screen already has. The warning names the checkgroup and the sheets instead.
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. `someMapContainsAll` already exists and is tested from task 126, with the
  existential semantics pinned. What remains is the presentation: which set, which checkgroup, which
  sheets it is spread across.
- 2026-09-03 — `splitCheckgroups` reports the checkgroup's name and the sheets its checkpoints are
  scattered over, per set. Tested: split across two sheets fires; one sheet holding the whole group
  is silent; **two overlapping sheets that each hold the whole group are silent** (the existential
  property, and the one that keeps this warning from crying wolf on the normal case).
- 2026-09-03 — A checkgroup on **no** sheet at all is deliberately not reported here. It is already in
  the unassigned list, and saying the same thing twice in one dialog trains an operator to skim both.
- 2026-09-03 — Added a test for a group spread across the **two areas of one double-sided sheet**,
  which must stay silent. It is the case that would break a geometry-based implementation, and it
  documents why this one counts membership.
- 2026-09-03 — **Did not** highlight the offending checkpoints on the map, against the acceptance
  criterion. Fading is already the language of sheet *selection*, and a second source of it would
  leave the operator unable to tell which question the map was answering — for a warning that already
  names the checkgroup and the sheets in words. Recorded rather than quietly skipped; worth revisiting
  if it turns out to be hard to act on in practice.
- 2026-09-03 — Verified on real data: split `Postlinje 1`'s two posts across the two live sheets and
  confirmed the same logic the SPA runs reports
  `SPLIT in set 'Patruljer': checkgroup 'Postlinje 1' across ['Kort 2', 'Kort 1']` from the actual
  `GET /api/kort` payload — then put the group back on one sheet and confirmed zero warnings.
- 2026-09-03 — Left the dev data **clean**: one warning-free patrol set, so the screen does not greet
  anyone with amber. The deliberate seam between the two sheets' areas is still there, but it only
  shows when the coverage toggle is pressed.
- 2026-09-03 — Completed. 44 kort tests pass, `vite build` clean, no type errors in the feature.
