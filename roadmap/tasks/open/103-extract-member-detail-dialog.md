# 103 — Extract MemberDetailDialog and make it live

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

Prerequisite for PRD 008's "same info everywhere", and worth doing on its own.

The member detail modal is not a component: it is a `detail` ref, a `detailData` ref, a
`loadDetail()` and a `<Dialog>` (around line 725) inside the 956-line `SosTeamCard.vue`. Extract it
to `vue/src/components/MemberDetailDialog.vue`, taking `memberId` and `teamId` (the endpoint needs
the team for a member the lifecycle has never touched).

**And make it live.** `loadDetail()` is a plain `http.get` into a local ref, so two crew members
looking at the same scout cannot see each other's work — which is disqualifying for a shared note
trail. Convert to `useLiveResource` keyed `member:{id}`, depending on the **instance**
(`spejder:{id}`) plus the types the payload joins in.

**Behaviour-preserving.** This component carries the nødtelefon's collection and move flows and is
its most sensitive screen. Nothing about the case card's behaviour may change: the modal's status
badge and action buttons read the row behind it (`detailMember`) precisely so they cannot disagree
with it, and that must survive the move.

Do this on its own, before any note work, so a refactor of the nødtelefon and a new feature are
never in the same diff.

## Acceptance Criteria

- [ ] `MemberDetailDialog.vue` exists, taking `memberId` and `teamId`
- [ ] `SosTeamCard.vue` uses it; its own behaviour unchanged, including the status badge and the
      waiting/resume buttons reading the live row
- [ ] Loads via `useLiveResource`, `dependsOn` the member instance and the joined types
- [ ] `pending` drives the modal's loading state; reopening a scout does not flash
- [ ] `SosTeamCard.vue` is meaningfully shorter
- [ ] `vue-tsc` clean; manual check of the case card's member flows

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
