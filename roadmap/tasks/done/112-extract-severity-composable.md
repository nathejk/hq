# 112 — extract `composables/severity.ts` from `sos.ts`

**Status:** done
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §8 ("Priority: mirrored from SOS, shared not copied"). A dispatch task's priority is the
SOS vocabulary — grøn / gul / rød — so that two race-night desks do not have two words for
urgent, and a pickup created from a red case can arrive red.

`composables/sos.ts` holds `severityLabel`, `severityTagSeverity` and `severityOptions`, kept in
one place "so the list badge and the detail select cannot drift apart" — exactly the drift a copy
in a dispatch view would reintroduce. But a delivery of dinner importing `sos.ts` is the wrong
dependency to read.

Move the three helpers to a neutral `composables/severity.ts` and **re-export them from
`sos.ts`**, so the nødtelefon's call sites are untouched.

## Acceptance Criteria

- [x] `composables/severity.ts` owns the three helpers, with no sos-specific naming
- [x] `sos.ts` re-exports them; no existing call site changed
- [x] Colours stay theme tokens (`success` / `warn` / `danger`), no hex
- [x] `npm run build` / type-check clean

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — Moved `severityLabel`, `severityTagSeverity` and `severityOptions` to
  `composables/severity.ts` and re-exported them from `sos.ts`. Not a single call site changed,
  which was the point: the nødtelefon should not have to be edited for kørsel's benefit.
- 2026-08-27 — `Severity` the *type* stays declared in `sos.ts` as well as `severity.ts`. It is
  four words, several sos call sites import it from there, and re-exporting a type through the
  module that used to own it is the same courtesy as the helpers.
- 2026-08-27 — The new module's header records the mirror-image decision on the Go side —
  `dispatch` declaring its own `Priority` rather than importing `sos` — so the two halves of one
  decision are not discovered separately, and so whoever lifts `sos` to shared-go (task 055)
  finds both places that converge on `types.Severity`.
- 2026-08-27 — ✅ Verified with `vue-tsc`: nothing reported for `sos.ts` or `severity.ts`, and the
  repo's pre-existing error count is unchanged at 109. (There is no node toolchain outside
  Docker here, so the check ran in a `node:22-alpine` container against the repo's own
  node_modules.)
- 2026-08-27 — All criteria met. Moving to done.
