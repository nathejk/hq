# 054 — Lift-readiness check: no nathejk.dk imports in the sos package

**Status:** open
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §8. `go/nathejk/table/sos/` is written to shared-go's guidelines so the
eventual lift (task 055) is a file move rather than a rewrite. The single check that
decides which of those it will be is whether the package imports anything from
`nathejk.dk/...` — and **nothing in the build complains if it does**, so the discipline
rots silently.

Note the local precedent runs the other way: `table/year/commands.go:28`,
`table/checkgroup/commands.go:54` and `table/checkpoint/command.go:31` all reach for
`nathejk.dk/internal/requestctx` directly. The sos package must not; the actor is passed
in by the handler instead.

Make it mechanical rather than a habit: a Go test in the package (or a small script run in
CI) that reads the package's imports and fails on any `nathejk.dk/` prefix.

## Acceptance Criteria

- [ ] An automated check fails when a `nathejk.dk/...` import is added to the sos package
- [ ] The check passes on the current package
- [ ] Proven to fail: add such an import temporarily and confirm it is caught

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
