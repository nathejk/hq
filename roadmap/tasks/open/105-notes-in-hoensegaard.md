# 105 — Notes in Hønsegården

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Name opens the dialog in all three sections
- [ ] Noter column with count and snippet; empty for scouts with none
- [ ] Writing a note updates the row's badge without a reload
- [ ] The dialog defers list updates while the note form is dirty, and says so
- [ ] `vue-tsc` clean; verified against the running stack

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
