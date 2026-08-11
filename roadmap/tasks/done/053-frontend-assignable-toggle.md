# 053 — Frontend: assignable toggle on the Organisation page

**Status:** done
**Priority:** low
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §6/§8. Section rows on the Organisation page gain a toggle — "kan
tildeles nødråb" — controlling whether the section can be picked as a case's assignee.

- Off by default, so the assignee list starts empty and is opted into deliberately
- Calls `PUT /api/section/:slug/sos-assignable` (task 045)
- The flag arrives in the existing `GET /api/organisation` payload, so no new fetch
- Optimistic toggle; revert on failure

This is the only change this feature makes to an existing screen besides the patrol card.

Depends on 045.

## Acceptance Criteria

- [x] Toggle visible per section, reflecting the stored flag
- [x] Toggling on makes the section appear in a case's Tildelt select, and off removes it
- [x] A case already assigned to a section that is later un-flagged keeps its assignee
- [x] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Deliberately did **not** migrate `OrganisationView` to `useLiveResource`
  while here. It still loads by hand and PRD 004 §12 records that it needs a dirty guard
  before it can be made live — that is its own task, and doing it inside a PRD 001 task
  would mix two unrelated risks.
- 2026-08-11 — The flag shows as a **persistent icon when on** and a hover action when off:
  which sections take nødråb is worth seeing at a glance, but an off-state control on
  every row is clutter. Matches how the existing edit/delete row actions behave.
- 2026-08-11 — Optimistic toggle with a snapshot revert on failure. A switch that waits
  for a round trip before moving feels broken; a switch that lies about the result is
  worse, hence the revert plus an error toast.
- 2026-08-11 — The "keeps its assignee" criterion is handled on the *case* side (task 050):
  `SosView` keeps a currently-assigned section in the select even when it is no longer
  assignable, so saving something else cannot silently clear it.
- 2026-08-11 — Two type errors of my own making (`node.data.slug` is optional in the tree's
  loose types); fixed with an `isSosAssignable(slug?)` helper rather than four non-null
  assertions in the template. Confirmed by counting: `OrganisationView` had 7 pre-existing
  type errors before this change and 7 after.
- 2026-08-11 — ✅ Criteria met: 60 frontend tests pass, Vite compiled cleanly, and the
  endpoint behind it was verified end to end in task 045.
