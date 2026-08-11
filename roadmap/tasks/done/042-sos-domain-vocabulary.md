# 042 — SOS domain vocabulary (types, severity, message structs)

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001**. Foundation for every other SOS task: the ids, the severity type and
the event bodies the SOS events are encoded with.

Per PRD 001 §8 (amended 2026-08-11) these live **inside `go/nathejk/table/sos/`**, not
in shared-go, so no cross-repo release blocks implementation. They are lifted with the
package later (task 055). `types.SosID` already exists in shared-go and must be reused
rather than redefined.

Needed:

- `SosCommentID` (string-ish id type, same shape as shared-go's other ids)
- `Severity` with `green` | `yellow` | `red` and a `Valid()` helper
- `Status` with `open` | `closed`
- Event bodies for: `created`, `headline.updated`, `description.updated`, `commented`,
  `comment.updated`, `severity.specified`, `assigned`, `deleted`, `closed`, `reopened`,
  `team.associated`, `team.disassociated`, plus the assignable-section toggle used by
  task 045.

## Constraints

- **No imports from `nathejk.dk/...`** — this is what keeps the later lift a file move
  (task 054 asserts it).
- Depend only on `shared-go/types` and `shared-go/messages` for existing vocabulary
  (`types.SosID`, `types.TeamID`, `types.YearSlug`, `types.Slug`, `messages.Metadata`).

## Acceptance Criteria

- [x] Types and event bodies compile in `go/nathejk/table/sos/`
- [x] `Severity.Valid()` and `Status.Valid()` reject unknown values
- [x] `types.SosID` reused, not redefined
- [x] No `nathejk.dk/...` import in the package
- [x] `go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up. Plan: `types.go` for ids/enums/actor, `messages.go` for the
  event bodies, a table test for the validators. Reference for style:
  `shared-go/tables/signup` and `shared-go/messages/team.go`.
- 2026-08-11 — Named the comment id `sos.CommentID` rather than `SosCommentID`: inside
  package `sos` the latter stutters, and it lifts to shared-go under the PRD's name when
  the package moves (task 055). Same for `NewSosID`, which is only a constructor —
  `types.SosID` itself stays shared-go's.
- 2026-08-11 — Decided **not** to make `ActivityType` an exhaustive enum. PRD 006 appends
  member transitions to this timeline, and an exhaustive switch is exactly what would
  make that a breaking change; the projection and the SPA are both required to tolerate
  unknown types instead.
- 2026-08-11 — Event bodies carry `sosId` even though it is already in the subject.
  Costs a few bytes; buys a body that still makes sense when a human reads it out of a
  log or a dead-letter queue.
- 2026-08-11 — ✅ All criteria met: `go build ./...`, `go vet`, `go test` and `gofmt -l`
  all clean; three tests cover the validators and the id constructors. Moving to done.
