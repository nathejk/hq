# 044 — SOS write side: commands and year-scoped subjects

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §8. The command struct that publishes SOS domain events on the current
subject convention `NATHEJK.{year}.sos.{sosId}.{event}`, built with
`github.com/jrgensen/stream/subject` — **not** the legacy `nathejk:sos.*` channel
strings.

Events: `created, headline.updated, description.updated, commented, comment.updated,
severity.specified, assigned, deleted, closed, reopened, team.associated,
team.disassociated`.

Requirements from the PRD:

- **Dirty-check every field**: a patch that changes nothing publishes nothing, and
  therefore emits no live signal. Close/reopen are idempotent for free.
- The **acting user is passed in by the handler**, not read from context inside the
  package — that is what keeps `nathejk.dk/internal/requestctx` out of the imports
  (PRD 001 §8). Recorded as `createdByUserId`; empty until the auth service exists.
- The server mints `SosID` and `SosCommentID`.

## Acceptance Criteria

- [x] `commands.Sos` publishes one event per changed field
- [x] A no-op patch publishes nothing (verified by test)
- [x] Subjects are year-scoped and match `NATHEJK.{year}.sos.{id}.{event}`
- [x] Actor arrives as an argument; no `requestctx` import in the package
- [x] `go test ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up. Plan: `commands.go` with a `PatchCommand` of pointer fields,
  dirty-checked against the querier, following `table/year/commands.go` for the publish
  idiom but taking the actor as an argument.
- 2026-08-11 — The year is read from the **stored case** rather than passed in by the
  handler. It has to be: the subject is year-scoped, and taking the year from the request
  header would let a case's later events land under a different year from its `created`
  event if an operator switched year in another tab.
- 2026-08-11 — Close/reopen are a `status` field assignment, so idempotency is the
  dirty-check rather than its own guard: closing a closed case falls out as "nothing
  changed, publish nothing". Two tests pin this.
- 2026-08-11 — Added a check that a comment being edited actually belongs to the case.
  Without it a mistyped id would amend some other case's comment, and the edit would be
  recorded on a timeline nobody involved is reading. Deliberately **not** checking that
  the caller wrote it — §11 decided any operator may edit, and there is no identity to
  check against anyway.
- 2026-08-11 — `AssociateTeam` is dirty-checked even though the projection upserts: the
  reason is the timeline, not the table. Two operators on the same call both reaching for
  the same patrol should not produce two "patrulje tilknyttet" entries.
- 2026-08-11 — Wrote 16 command tests against a recording publisher. The two that matter
  most: a patch changing nothing publishes nothing (so no live signal, so other
  operators' screens do not refetch for a no-op), and a patch publishes exactly one event
  per changed field.
- 2026-08-11 — One fake had the wrong signature — `stream.Publisher.Publish` takes a
  single message, not a variadic. Compiler caught it.
- 2026-08-11 — ✅ All criteria met: 22 tests pass in the package, `go build ./...`,
  `go vet` and `gofmt -l` clean. Moving to done.
