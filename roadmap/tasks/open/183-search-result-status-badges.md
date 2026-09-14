# 183 — Status badges on search result rows

**Status:** open
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Depends on task 182. Renders the status that task 179 joined in.

**Status is shown on every row, not only on the unusual ones.** A badge that appears only
when something is wrong makes its absence carry meaning the operator has to know to read;
showing "racing" as plainly as "udmeldt" means a row can always be taken at face value.

Use the existing severity vocabulary in `vue/src/composables/severity.ts` so a departed person
does not look like a current one. The distinction that matters operationally is not
current-vs-departed but *in our care right now* (`sheltered`, `transit`, `waiting`) versus
*somebody else's charge* (`released`, `reunited`) versus *never started* (`deleted` →
"udmeldt").

`deleted` and lifecycle status are separate fields and must stay distinguishable: "udmeldt"
(never started) is not "released" (went home during the night). A scout can only be one of
them, but the UI must not render them identically.

An unknown status — a spejder with no `spejderstatus` row, ordinary for early signups and old
years — renders as unknown. Do not dress it up as `registered`.

Danish wording throughout. Note the open question in PRD 014 §11: a klan member's
`signupStatus` of `PAY` reads badly and is a fact about the *team*, not the person. Pick
something defensible and log the choice rather than blocking on it.

## Acceptance Criteria

- [ ] Every result row shows a status, including ordinary ones
- [ ] Severity styling from `composables/severity.ts`; departed ≠ current visually
- [ ] "udmeldt" distinguishable from "released"
- [ ] Unknown status rendered as unknown
- [ ] Danish labels; wording choice for team-level statuses logged
- [ ] Component test asserting a departed row is styled differently from a racing one
- [ ] `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
