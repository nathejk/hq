# 106 — Notes on the case card and the patrol page

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The half of PRD 008 that makes one trail worth having: the nødtelefon reads what the shelter wrote,
and vice versa.

- `SosTeamCard.vue` already opens the member dialog, so it inherits the notes for free once task 103
  lands. Verify it, and add the note count to the member row so an operator sees there is something
  to read before opening it.
- `PatruljeView.vue`: the roster's names open the same dialog.

**Notes stay off the SOS timeline** (PRD 008 §4, decided 2026-08-23). The card *shows* a scout's
notes because it shows that scout; nothing is copied onto the case. One text, one place it can be
edited, and no copy to diverge when it is corrected.

Do not confuse this with task 097, which is PRD 007's: that puts a machine-written summary of a
*status change* on the case timeline and still stands.

## Acceptance Criteria

- [ ] Notes readable and writable from the case card's member dialog
- [ ] Note count visible on the case card's member rows
- [ ] Patrol page roster names open the same dialog
- [ ] Nothing appended to any SOS timeline by this task
- [ ] `vue-tsc` clean

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
