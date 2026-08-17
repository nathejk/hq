# 081 — Lift-readiness check for the spejderstatus package

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] `lift_test.go` in the `spejderstatus` package, modelled on the SOS one
- [x] Fails if a `nathejk.dk/...` import is introduced (verified by adding one temporarily)
- [x] Passes against the package as built by tasks 065–072
- [x] `go test ./nathejk/table/spejderstatus/` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Best added with task 065 so it guards from the start.
- 2026-08-17 — Picked up. `lift_test.go` copied from `table/sos/` (task 054) and adapted,
  including its guard-against-the-guard: if the directory scan finds no non-test `.go`
  files the test fails rather than passing vacuously, so a rename cannot turn the check
  into a no-op that reports success.
- 2026-08-17 — Named the two likely offenders in the doc comment rather than describing the
  rule abstractly, because both are real temptations in this package specifically:
  `cmd/api/actor.go` returns a `sos.Actor` and importing `table/sos` for that one type is
  the obvious shortcut (which is why task 070 gives this package its own `Actor`), and
  `internal/requestctx` is what `table/year`, `table/checkgroup` and `table/checkpoint` all
  reach into for the acting user — precisely what makes those three awkward to move.
- 2026-08-17 — **Verified the guard actually guards.** Temporarily added
  `_ "nathejk.dk/internal/requestctx"` to `querier.go`; the test failed with the intended
  message naming the file and the import. Reverted, suite green. An untested assertion is
  not an assertion — and this one is easy to write in a way that never fires, since it
  passes when it finds nothing.
- 2026-08-17 — Added a second test the task did not ask for: `table.sql` must be present in
  the package directory and the embedded schema must be non-empty and contain the columns
  PRD 006 added. Rationale: the schema is loaded with `//go:embed`, so a lift that took the
  `.go` files and left a stale `.sql` behind would compile perfectly and fail at 3am with
  "Unknown column" — the same failure mode task 065 already hit once on the dev stack.
- 2026-08-17 — ✅ All criteria met. Moving to done.
