# 184 — Opt-in previous-years search

**Status:** doing
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

## Description

Depends on task 183. Implements PRD 014's year rule end to end.

**Searching this year is the primary function. Previous years are a deliberate act by the
operator, never something the system slides into.** The server must not widen the year scope
on its own — not on an empty result, not on an exact phone match, not ever. An operator who has
not asked for history must be able to trust that what they are reading is this year.

Backend: honour `includePreviousYears` in the query layer and handler (the parameter and its
`false` default were wired in task 181).

Frontend: **a checkbox that is visibly off**, reading "søg også tidligere år" — not a year
dropdown pre-filled with the current year. A dropdown invites the operator to change the year
without noticing, and every list in HQ is already year-scoped by the global year selector.

Cross-year results state their year **on every row** and are grouped *after* the current
year's, never interleaved. A hit from 2024 presented alongside this year's is exactly the
stale-data-that-looks-current failure the live-updates design exists to avoid.

The control belongs to the query, so it belongs in the URL alongside `q` — a shared link must
reproduce what the sender saw.

## Acceptance Criteria

- [ ] `includePreviousYears` honoured through query layer and handler
- [ ] Test: the server never widens the scope on its own, including on an empty result
- [ ] Checkbox off by default, labelled "søg også tidligere år"
- [ ] Cross-year rows labelled with their year and grouped after the current year's
- [ ] The toggle is reflected in the URL
- [ ] `go test ./...`, `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:06 — Picked up. Plan: frontend only — the two backend criteria were delivered by
  task 181, under the deliberately different name `includeOtherYears`. Add a checkbox, off by
  default, reflected in the URL alongside `q` and folded into the live-cache key (otherwise the two
  result sets collide on one entry). Extend `toRows` with the active year so cross-year rows form
  their own groups *after* every current-year group, and give them a visual treatment plus a Year
  column. First job: establish how the SPA can know the active year at all, since
  `globalstate.yearSlug` is deliberately empty for the current calendar year.
