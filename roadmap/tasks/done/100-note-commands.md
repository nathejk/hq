# 100 — Note commands

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] Both commands implemented; ids minted server-side
- [x] Empty and over-long notes refused with distinct errors
- [x] Length counted in runes
- [x] `UpdateComment` refuses a note that belongs to another member
- [x] Unchanged text publishes nothing
- [x] Command tests for each rule, following `shelter/commands_test.go`

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 21:00 — Picked up with 101, since commands are only demonstrable through endpoints.
  `New` gains the publisher that 099 deliberately left out; wired into `commands.Commands` as
  `Note` and `data.Models` as `Note`.
- 2026-08-23 21:10 — `ErrWrongMember` is its own error rather than a not-found. Telling an operator
  "no such note" about a note plainly on their screen would send them hunting a bug that is not
  there; the two cases are a client bug and a wrong-screen edit, and both deserve saying.
- 2026-08-23 21:15 — Decision: **no duplicate detection on `Comment`.** Two identical notes are a
  legitimate record — "ringet til mor, intet svar" twice is two facts an hour apart — and
  suppressing the second would lose the more interesting one. `UpdateComment` *does* dirty-check,
  because a resubmitted edit is not a new fact.
- 2026-08-23 21:20 — Length in runes, as `shelter.MaxPlacementLength` learned: a note about "må sove
  i hallen — bange for edderkopper" is shorter than `len()` thinks.
- 2026-08-23 21:25 — Documented in `Commands` why editing is **not** restricted to the author: with
  every actor anonymous, an ownership check would compare two empty strings and permit everything,
  which is worse than no check because it would look like one. Flagged as a decision to revisit the
  day login lands, rather than a property to inherit silently.
- 2026-08-23 21:40 — ✅ All criteria met. 12 command tests; full Go suite green. Verified live — see
  task 101's log for the end-to-end trace.
- 2026-08-23 21:41 — Moving to done.
