# 102 — Note summary on the shelter rows

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

Notes have to be discoverable **without opening anything**, or nobody finds the one scout with
instructions among forty rows (PRD 008 §6). `GET /api/shelter` therefore gains, per member:

- `noteCount` — how many notes exist
- `latestNote` — the most recent note's text, truncated server-side (first ~120 characters), with
  `latestNoteAt`

From `spejdernote.SummaryByMembers` (task 099): one grouped query batched over the members already
fetched, in the same shape as the placering lookup. **No query per row** — this is a page the crew
keeps open all night.

Truncated on the server rather than in the view, so the payload does not carry 2000 characters per
row for a one-line snippet. The full text arrives with the thread when somebody opens the scout.

This is also the deliberate answer to "expandable rows" as a host (PRD 008 §7): it gives the
context benefit — notes visible while scanning — without a second editing UI.

## Acceptance Criteria

- [ ] `noteCount`, `latestNote`, `latestNoteAt` on each shelter row
- [ ] One batched query; no per-member lookups
- [ ] Snippet truncated server-side, on a rune boundary
- [ ] Members with no notes report `0` and an empty snippet, not null
- [ ] Handler tests, including a member whose notes belong to somebody else not leaking in

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
