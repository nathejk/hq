# 100 — Note commands

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The write side of PRD 008, on a `spejdernote` commander:

- `Comment(ctx, actor, year, memberID, note) (NoteID, error)` — mints the id server-side, as
  `sos.NewCommentID` does, so a client cannot collide with one it has not seen.
- `UpdateComment(ctx, actor, year, memberID, noteID, note) error`

Rules, following `sos`'s comment commands:

- Trimmed; an empty note is refused with its own error rather than published.
- Capped at 2000 characters, counted in **runes** — a Danish note must not be refused for its æ's,
  the same trap as `shelter.MaxPlacementLength`.
- `UpdateComment` must verify **the note belongs to the member named in the request**. Without it
  a client could amend any note by id, which is the check `sos.UpdateComment` makes for the same
  reason.
- Unchanged text publishes nothing, so a re-submitted edit does not put a second version in the
  stream.

No `sosId` anywhere: notes are member-scoped and case-free by design (PRD 008 §4).

Note on authorisation: editing is not restricted to the note's author, because there is no
identity to restrict it with (PRD 001 §6). That is accepted for now and should be revisited when
login lands — leave a comment saying so, rather than letting a future reader assume it was
considered and wanted.

## Acceptance Criteria

- [ ] Both commands implemented; ids minted server-side
- [ ] Empty and over-long notes refused with distinct errors
- [ ] Length counted in runes
- [ ] `UpdateComment` refuses a note that belongs to another member
- [ ] Unchanged text publishes nothing
- [ ] Command tests for each rule, following `shelter/commands_test.go`

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
