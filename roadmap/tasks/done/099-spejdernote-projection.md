# 099 — spejdernote projection

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The read model behind PRD 008: prose notes attached to a scout.

New local package `go/nathejk/table/spejdernote/` (`table.go`, `table.sql`, `messages.go`,
`consumer.go`, `querier.go`), with the events it owns:

- `NATHEJK.{year}.spejder.{memberId}.commented` — `{noteId, memberId, note, actor}`
- `NATHEJK.{year}.spejder.{memberId}.comment.updated` — same shape

Its own package rather than a table on `spejderstatus` (which lifts to shared-go verbatim, task
083) or on `sos` (notes are not case-scoped — that is the point of PRD 008 §4).

Row keyed by `noteId`, holding `memberId`, `year`, `note`, `actorUserId`, `createdAt`,
`updatedAt`. **The projection holds the current text; the event stream holds history** — an edit
updates the row, and every version stays in JetStream, so showing an edit history later is a UI
decision rather than a migration.

Queries needed by later tasks:
- `GetByMember(ctx, year, memberID) ([]Note, error)` — the thread, **oldest first**
- `SummaryByMembers(ctx, year, ids) (map[MemberID]Summary, error)` — count plus the latest note's
  text, one grouped query, for the row summaries in task 102

Include the `schemaMigrations` hook from the start, as `shelter` does: `CREATE TABLE IF NOT
EXISTS` never alters an existing table, so a column added later is silently absent from every
database that has already booted.

**Wire the consumer into the `projections` slice in `cmd/api/main.go`.** Outside that slice it
emits no live signal. The entity token is `spejder`, already advertised, so no client needs a new
dependency.

## Acceptance Criteria

- [x] Package with the five files; events defined with subjects as above
- [x] Row keyed `(noteId)`, indexed for both the thread and the per-member counts
- [x] `comment.updated` updates text and `updatedAt`, and does not move `createdAt`
- [x] Replaying the same events twice produces identical rows
- [x] An event for a member the projection has never seen still lands
- [x] `GetByMember` oldest first; `SummaryByMembers` batched, no query per member
- [x] Consumer in the `projections` slice
- [x] Consumer tests: add, edit, replay, unknown member, year scoping

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 20:15 — Package written: `table.sql`, `table.go`, `messages.go`, `consumer.go`,
  `querier.go`. `commands.go` deliberately absent — it is task 100 — so `New` takes no publisher
  yet; taking one now and leaving it unused would be a field nobody could explain.
- 2026-08-23 20:25 — **The bug this table could most easily have had, found while writing the
  upsert.** Replay re-delivers the original note on every boot, *after* the correction that came
  later. With `updatedAt` in the insert's update list, a corrected note's precedence would depend
  on statement ordering rather than on the data — and with `note` alone it would have been worse.
  `createdAt` and `updatedAt` are both excluded from the replayed insert, so the correction's own
  event is what decides the text. Pinned by
  `TestReplayingTheOriginalDoesNotUndoACorrection`, which is the test this package exists to pass.
- 2026-08-23 20:30 — A correction's UPDATE is scoped to noteId **and** memberId **and** year, not
  just the id. The command will check ownership too (task 100), but doing it here means an event
  published by anything else cannot reach another member's note either.
- 2026-08-23 20:40 — **Deleted my own first version of `SummaryByMembers`.** I had written the
  latest-note-per-member as `SUBSTRING_INDEX(MAX(CONCAT(DATE_FORMAT(…), noteId, 0x1f, note)), 0x1f,
  -1)` — the greatest-n-per-group trick. It works, it is unreadable, and it silently depends on the
  timestamp format sorting lexicographically. Replaced with one ordered query and a fold in Go:
  the shelter lists a few dozen scouts with a handful of notes each, so it is a few dozen rows and
  it is obvious. The comment above it says so, including what was tried, so nobody re-derives the
  clever version.
- 2026-08-23 20:45 — `Edited()` is a method on `Note` rather than a field the client compares,
  because the timestamps are equal by construction on a fresh note and every consumer would
  otherwise reimplement that comparison — including, eventually, one using `!=` on a `time.Time`.
- 2026-08-23 20:50 — ✅ All criteria met. 13 tests, all green; full `go test ./...` green.
  Verified against the running stack: the hot-reloaded API created the `spejdernote` table (columns
  confirmed in MySQL), boot is clean, and the advertised live entity set is **unchanged** — exactly
  as PRD 008 §6 predicted, because the events are `spejder` subjects, so no client needs a new
  dependency and the shelter list already invalidates on them.
  End-to-end proof of an actual note has to wait for tasks 100/101: nothing can publish one yet.
- 2026-08-23 20:51 — Moving to done.
