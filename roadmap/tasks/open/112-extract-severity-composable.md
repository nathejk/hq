# 112 — extract `composables/severity.ts` from `sos.ts`

**Status:** open
**Priority:** medium
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `composables/severity.ts` owns the three helpers, with no sos-specific naming
- [ ] `sos.ts` re-exports them; no existing call site changed
- [ ] Colours stay theme tokens (`success` / `warn` / `danger`), no hex
- [ ] `npm run build` / type-check clean

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
