# 095 — Placering combobox with self-defining suggestions

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The zones scouts are kept in **are not known until race start**, so they are neither
configured nor hardcoded (PRD 007 §6). The placering field is an editable combobox whose
suggestions are the distinct placeringer already recorded this year — `placements` in the
`/api/shelter` envelope (task 087's `DistinctPlacements`), most-used first — with free text
still accepted.

The first scout into a tent is typed; every one after that is a pick. That is what stops
"Telt 4", "telt4" and "t4" becoming three places, with no zone entity, no admin screen and
no setup step on the night.

**Unsaved state must never be overwritten.** This is an editor, so while it is dirty,
incoming live payloads are deferred and applied when the edit ends, and the screen says
updates are paused — exactly as `KlanListView.vue` and `KortView.vue` do. A crew member
typing a tent number at 3am must not have the field yanked out from under them because
somebody else's scout changed status.

Max 64 characters, trimmed, matching the server's validation.

A typo becomes a suggestion — accepted: ordering by use count keeps the real zone at the top
and the mistake at the bottom, and correcting the affected scout's placering removes it. No
rename tool in v1.

## Acceptance Criteria

- [ ] Editable combobox; suggestions from the payload, most-used first
- [ ] Free text accepted and submitted unchanged (bar trimming)
- [ ] Dirty state defers incoming payloads and applies them when the edit ends
- [ ] The screen states that updates are paused while editing
- [ ] 64-character cap, trimmed
- [ ] Manually verified: typing a new zone makes it a suggestion for the next scout

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
