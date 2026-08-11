# 053 — Frontend: assignable toggle on the Organisation page

**Status:** open
**Priority:** low
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Toggle visible per section, reflecting the stored flag
- [ ] Toggling on makes the section appear in a case's Tildelt select, and off removes it
- [ ] A case already assigned to a section that is later un-flagged keeps its assignee
- [ ] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
