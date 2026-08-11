# 042 — SOS domain vocabulary (types, severity, message structs)

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Types and event bodies compile in `go/nathejk/table/sos/`
- [ ] `Severity.Valid()` and `Status.Valid()` reject unknown values
- [ ] `types.SosID` reused, not redefined
- [ ] No `nathejk.dk/...` import in the package
- [ ] `go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
