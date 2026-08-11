# 044 — SOS write side: commands and year-scoped subjects

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `commands.Sos` publishes one event per changed field
- [ ] A no-op patch publishes nothing (verified by test)
- [ ] Subjects are year-scoped and match `NATHEJK.{year}.sos.{id}.{event}`
- [ ] Actor arrives as an argument; no `requestctx` import in the package
- [ ] `go test ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
