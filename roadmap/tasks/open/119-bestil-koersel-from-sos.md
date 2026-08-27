# 119 — `Bestil kørsel` from an SOS case, and the case's task list

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §5 (primary scenario), §6 (seams), §7. Phase 3. Depends on tasks 110 and 114.

The nødtelefon operator must be able to turn a case into a task **without leaving the case**:
`SosView` / `SosTeamCard` gain a **Bestil kørsel** action on a waiting member and on the case,
creating a `pickup` task pre-filled with the case, its patrol and the waiting members.

The case then shows its tasks and their expected times, via `GET /api/sos/:id/dispatch`, so the
operator on the phone can read "22:35" off the case without opening the dispatch board.

A pickup card links to `MemberDetailDialog` (PRD 008), so the guardian's number is one click
away.

This is the mitigation for the discipline risk: one click from a case is the fastest path.

## Acceptance Criteria

- [ ] `Bestil kørsel` on the case and on a waiting member, pre-filling case, patrol and members
- [ ] `GET /api/sos/:id/dispatch` with OpenAPI annotations, `[]` never `null`
- [ ] Case shows its tasks with planned time or estimate, live
- [ ] Pickup task cards link to `MemberDetailDialog`
- [ ] Cancelling a task from the case requires a reason and removes it from its tour

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
