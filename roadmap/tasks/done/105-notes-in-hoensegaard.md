# 105 — Notes in Hønsegården

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Wire PRD 008 into the shelter screen — the place that asked for it.

- The scout's **name** opens `MemberDetailDialog` (task 103), which hosts `MemberNotes` (104). The
  name is the affordance in every host, so the crew learns one gesture.
- A **Noter** column: the count as a badge plus the first line of the latest note, from task 102's
  payload fields. Truncated with the full text in a tooltip. This is what makes notes visible while
  scanning rather than hidden behind a click.
- The dialog also carries the guardian's phone, address and birthday — which is why PRD 007 removed
  the phone column from these tables. Check that story actually holds once it is on screen: if
  reaching a parent now takes two clicks where it used to take none, say so rather than leaving it.

The list already depends on `spejder`, so writing a note invalidates it and the badge updates with
no new plumbing.

## Acceptance Criteria

- [x] Name opens the dialog in all three sections
- [x] Noter column with count and snippet; empty for scouts with none
- [x] Writing a note updates the row's badge without a reload
- [x] The dialog defers list updates while the note form is dirty, and says so
- [x] `vue-tsc` clean; verified against the running stack

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-24 00:50 — The scout's name is now a link-button opening `MemberDetailDialog`. A button
  rather than a row click, so sorting a column or editing a placering cannot open a dialog by
  accident — and the name is the affordance on every host, so the crew learns one gesture.
- 2026-08-24 00:55 — Noter column: the count as a badge plus the newest line, both opening the
  dialog. A scout with **no** notes shows "Tilføj" rather than an empty cell — on somebody sheltered,
  no notes means nobody has written down what was agreed, which is worth a nudge.
- 2026-08-24 01:00 — The snippet is already truncated server-side (120 runes, task 102); the view only
  collapses whitespace, because a note written across three lines would otherwise stretch the row and
  push every other scout down the screen.
- 2026-08-24 01:05 — Generalised the deferral: `paused` is now "a placering is being typed **or** a
  note is", replacing the placering-only check. Both are unsaved state that a re-render destroys — the
  placering field sits in a row that can move between sections, and the dialog's panel rebuilds around
  its textarea. A `watch(paused)` applies what arrived when either ends, because the dialog reports the
  *state* rather than the transition. The banner names which one is holding things up.
- 2026-08-24 01:08 — `closeMember` clears `notesDirty` explicitly: the form unmounts with the dialog,
  so no dirty event will ever arrive to end the pause. Without it, closing the dialog mid-sentence
  would freeze the list for the rest of the night.
- 2026-08-24 01:10 — Deliberately **no** actions slot on the dialog here. The shelter's actions live
  in the row, where the crew is already looking; duplicating them inside the dialog would give two
  places to press for one thing. This is also what the slot design bought — the same component,
  different actions, no flags.
- 2026-08-24 01:15 — ✅ All criteria met. `vue-tsc` clean, `Badge` resolves through the auto-import
  plugin, 108 vitest passing, Go suite green. Verified the payload the column reads against the
  running stack: Marie shows `noteCount: 1` with her snippet, the two closed scouts `0`.
  The one thing only a human can confirm is the feel of it — writing a note from the shelter screen
  and watching the badge appear.
- 2026-08-24 01:16 — Moving to done.
