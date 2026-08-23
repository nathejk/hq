# 102 — Note summary on the shelter rows

**Status:** done
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] `noteCount`, `latestNote`, `latestNoteAt` on each shelter row
- [x] One batched query; no per-member lookups
- [x] Snippet truncated server-side, on a rune boundary
- [x] Members with no notes report `0` and an empty snippet, not null
- [x] Handler tests, including a member whose notes belong to somebody else not leaking in

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 22:50 — Rows now carry `noteCount`, `latestNote` and `latestNoteAt`, from one batched
  `SummaryByMembers` call beside the placering lookup. No per-row queries.
- 2026-08-23 22:55 — Snippet capped at 120 runes — about a line and a half, enough to recognise
  "Ringet til mor 01.20. Hun henter kl. 06…" without opening anything, which is the whole point of
  putting it on the row. Truncated on the **server**: a note may be 2000 characters, and a
  forty-scout screen should not carry 80KB of prose to render forty snippets.
- 2026-08-23 23:00 — `truncateRunes` cuts on runes, not bytes, and the test says why: cutting UTF-8
  mid-character yields ï¿½, and a snippet ending in a replacement glyph reads as corrupted data
  rather than as an abbreviation. Danish makes that likely rather than theoretical — æ, ø and å are
  two bytes each — so the test truncates a string of æ.
- 2026-08-23 23:05 — ✅ All criteria met. 4 new handler tests, including the one that would be a real
  incident: a note about one child appearing on another child's row. `cmd/api` green; full Go suite
  green.
- 2026-08-23 23:10 — Verified against the running stack: Marie shows `noter: 1` with the snippet and
  timestamp of the note written in task 101, and the two closed scouts show `0` with an empty snippet
  rather than null.
- 2026-08-23 23:11 — Moving to done.
