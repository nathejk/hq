# 081 — Lift-readiness check for the spejderstatus package

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §8 (risks).

Add a test asserting that `go/nathejk/table/spejderstatus/` imports nothing from
`nathejk.dk/...`, so the package can later be lifted to `shared-go/tables/spejderstatus/` as
a file move rather than a rewrite.

`go/nathejk/table/sos/lift_test.go` already does exactly this for the SOS package (PRD 001,
task 054). Copy the approach.

## Notes

- **Why this needs a test rather than a review note:** a single convenience import turns the
  lift from a file move into a rewrite, and **nothing in the build will complain**. The rot is
  silent, which is the only reason it is worth a test at all.
- The likeliest offender is the acting user: `app.actor` returns `sos.Actor`, and importing
  `nathejk.dk/nathejk/table/sos` for that one type is the obvious shortcut. Task 070 exists to
  remove the temptation; this test is what keeps it removed.
- Second likeliest: `requestctx`. The handler resolves the actor and passes it in — the domain
  must not reach into request context itself. `table/sos/types.go` documents this as the rule
  that makes the package movable.
- Cheap to run, so it belongs with the normal test suite rather than a CI-only check.

## Acceptance Criteria

- [ ] `lift_test.go` in the `spejderstatus` package, modelled on the SOS one
- [ ] Fails if a `nathejk.dk/...` import is introduced (verified by adding one temporarily)
- [ ] Passes against the package as built by tasks 065–072
- [ ] `go test ./nathejk/table/spejderstatus/` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Best added with task 065 so it guards from the start.
