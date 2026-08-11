# 054 — Lift-readiness check: no nathejk.dk imports in the sos package

**Status:** done
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

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

- [x] An automated check fails when a `nathejk.dk/...` import is added to the sos package
- [x] The check passes on the current package
- [x] Proven to fail: add such an import temporarily and confirm it is caught

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Implemented as a Go test (`lift_test.go`) that parses the package's own
  non-test files with `go/parser` in `ImportsOnly` mode and fails on any `nathejk.dk/`
  prefix. A test rather than a CI script, so it runs for anyone who types `go test ./...`
  — including the person who just added the import.
- 2026-08-11 — Test files are exempt: they are not lifted, so they may import whatever they
  need. Only the shipped files have to stay clean.
- 2026-08-11 — Added a guard against the guard: if it finds no non-test `.go` files it
  fails rather than passing, so a rename or move that left it pointing at an empty
  directory cannot be reported as success. That is the failure mode of every
  "check nothing is wrong" test.
- 2026-08-11 — The error message says what to do instead — pass it as an argument or
  declare a port — because the tempting fix (an allowlist) defeats the point.
- 2026-08-11 — ✅ **Proven, not assumed:** added
  `import _ "nathejk.dk/internal/requestctx"` in a scratch file, watched the test fail
  with the file name and import path, then removed it and watched the package pass again.
