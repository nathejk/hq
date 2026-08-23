# 099 — spejdernote projection

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Package with the five files; events defined with subjects as above
- [ ] Row keyed `(noteId)`, indexed for both the thread and the per-member counts
- [ ] `comment.updated` updates text and `updatedAt`, and does not move `createdAt`
- [ ] Replaying the same events twice produces identical rows
- [ ] An event for a member the projection has never seen still lands
- [ ] `GetByMember` oldest first; `SummaryByMembers` batched, no query per member
- [ ] Consumer in the `projections` slice
- [ ] Consumer tests: add, edit, replay, unknown member, year scoping

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
