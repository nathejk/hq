# 128 — Checkpoint assignment UI with per-checkgroup select-all

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Checkpoints selectable per map, grouped by checkgroup, with per-group select-all
- [x] Saved via `PUT /api/kort/:id/checkpoints`
- [x] Selected map's checkpoints highlighted, others faded
- [x] "Ikke på noget kort" list for the selected set, clicking through to the map
- [x] Positionless checkpoints assignable and flagged, not blocked
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: the picker needs the checkgroups, which `KortView` already has
  loaded; passing them in as a prop keeps one fetch rather than giving the dialog its own.
- 2026-09-03 — Extracted the picker's two rules into `composables/kort.ts` as pure functions
  (`groupSelectionState`, `toggleGroupSelection`, `orderPicks`) rather than leaving them in the
  component. Tests here run on composables only, and these are exactly the parts where a subtle
  wrong answer would be invisible in a screenshot.
- 2026-09-03 — Select-all **completes** a partial checkgroup rather than inverting it. With three of
  four ticked, an operator reaching for the group header means "all of them", never "swap them";
  only a fully ticked group clears. Tested, because the inverting reading is the one a naive
  implementation lands on.
- 2026-09-03 — The group header shows a **third state** (`some`) instead of rounding to on or off. A
  half-ticked checkgroup is usually a mistake in the making — the checkgroup is revealed as a whole,
  so a sheet holding half of one is what task 133 will warn about — and it should be visible while
  it is still being made.
- 2026-09-03 — `orderPicks` sorts the selection into **checkgroup order** before sending. Order
  carries no meaning to the domain, but stability does: the API compares the submitted list against
  the stored one to decide whether anything changed, so ticking A-then-B and B-then-A must produce
  the same list or every re-save would look like an edit and emit a live signal to every session.
  Verified against the real API — sending the identical selection twice published nothing the second
  time.
- 2026-09-03 — Picked ids that are in no checkgroup are **kept**, not dropped. They can only appear if
  a checkpoint vanished from the payload mid-edit, and silently discarding them would make a save do
  something the operator did not ask for.
- 2026-09-03 — Checkpoints without a position are assignable and **flagged with a warning icon**
  rather than blocked: the sheet may well be drawn before somebody places the pin, and refusing
  would stop the operator recording what they know.
- 2026-09-03 — The unassigned list is **per set**, not overall. The two mistakes are different: a post
  missing from the crew maps is a driver who cannot find it, one missing from the patrol maps is a
  patrol that will never be sent there. An overall list could not tell an operator which they are
  looking at.
- 2026-09-03 — Field edits and tick-box edits have **separate save buttons**, because they are
  separate endpoints and separate events. One combined "Gem" would either send both (making a
  checkpoint save look like a rename to every other session) or hide which half was pending.
- 2026-09-03 — Had to move the `anyDirty` computed and its emit below every source it reads: the
  watcher runs immediately, so referencing a `const` declared later threw a TDZ error during setup.
- 2026-09-03 — Verified against the real API: assign two checkpoints, re-save the same selection (no
  event), clear to `[]`, re-assign. All correct.
- 2026-09-03 — Completed. 23 tests pass, `vite build` clean, no type errors in the feature's files.
  Still not verified in a browser — no way to drive one from here.
