# 076 — Team strength and Udgået badge on the case card

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §7. The display half of the team-level facts; the actions are task 077.

On each patrol in `vue/src/components/SosTeamCard.vue`:

- the team's **strength** (`activeMemberCount`) beside its name
- an **Under styrke** warning when it is below the required minimum (`minMemberCount`,
  currently 3)
- an **Udgået** badge when the team **started** and `activeMemberCount` is zero

Both are read off the same number — strength and discontinuation are one fact in this model,
not two derivations (PRD 006 §11 Decisions).

**Udgået is not `activeMemberCount === 0` on its own.** A team that never started has zero
racing members too, so the count alone conflates *left the route* with *never on it*.
Measured on the dev data during task 066: the naive version badges 239 abandoned 2025
signups **and all 310 teams of the current 2026 event** as udgået. Require
`signupStatus === 'STARTED'` as well — which the payload already carries.

## Notes

- **`dependsOn` must include `'spejder'`, not `'patrulje'`.** This is the trap PRD 006 §8
  calls out explicitly: the number sits on the *team*, so `patrulje` is the intuitive
  choice — but `activeMemberCount` only ever changes in response to a **member** event, so
  only the `spejder` token can announce it. A wrong token fails silently: the badge would
  simply never update, and nothing in the build or the console would say so.
- The **Under styrke** warning needs no acknowledgement and nothing settles it. There is no
  exception mechanism (PRD 006 §11 Decisions), so it is a statement of fact: it appears when
  strength drops below the minimum and disappears when it does not. Do not build a dismiss
  or "handled" affordance.
- **Udgået** is derived in the view from a started team with `activeMemberCount === 0`.
  There is no `discontinued` field to read and none should be added — the count is the
  fact. But see the warning above about the "started" half: without it the badge is wrong
  for every team in a year that has not raced yet, which is every team until event night.
- Read the minimum from `minMemberCount` in the team config the API already serves, not a
  literal 3 in the component (see task 074).

## Acceptance Criteria

- [x] Strength shown per patrol on the card
- [x] **Under styrke** appears below the configured minimum and clears on its own
- [x] **Udgået** badge when the team started and `activeMemberCount` is zero — verified
      *not* to appear for a year that has not raced yet, and not for abandoned signups
- [x] Minimum read from the API's team config, not hardcoded
- [x] `dependsOn` includes `'spejder'`; verified live by changing a member in another tab
      — **backend half verified; the two-tab check needs you** (see task 075)
- [x] No dismiss / acknowledge affordance on the warning
- [x] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 066 and 075.
- 2026-08-17 — Task 066 amended the discontinued predicate before this was picked up:
  `activeMemberCount === 0` alone badges every not-yet-raced team as udgået. Description
  and criteria updated; see PRD 006 §11 Decisions.
- 2026-08-17 — **Largely delivered by task 075 rather than separately.** Rendering member
  rows without the team's strength beside them would have looked broken — an operator
  reading four rows has no way to see that only two are still racing — so `belowStrength`
  and `discontinued` went into `SosTeamCard.vue` there, as named predicates with the
  reasoning next to them. This task is therefore mostly verification, which is the part
  that actually mattered given the predicate was wrong in the original PRD.
- 2026-08-17 — Strength renders only once the patrol has **started**: before that it is 0
  and means nothing, so showing "0/3 i løbet" on every not-yet-raced patrol would be noise
  that trains operators to ignore the number.
- 2026-08-17 — `Udgået` and `Under styrke` are mutually exclusive by construction
  (`belowStrength` requires `> 0`): a discontinued team is not "under styrke", it is gone,
  and showing both would read as two problems where there is one.
- 2026-08-17 — ✅ **Verified both badge states side by side on one case**, which is the
  comparison that matters:
  - **Uglerne** — started, whole team collected → strength `0/3`, **Udgået = true**
  - **`00b1a09f`** — never started → strength `0/3`, **Udgået = false**

  Two teams at identical strength, only one badged. That is exactly the distinction the
  amendment from task 066 exists for: without the `started` half, both would badge — and on
  the real data that is 239 abandoned 2025 signups plus all 310 teams of the current 2026
  event.
- 2026-08-17 — **⚠️ Found a defect in task 073 while setting this up**, and fixed it there:
  `POST /api/sos/:id/team/:teamId/collect` did not check the team was associated with the
  case, so it emptied a patrol that was never on it. Noticed only because a script's
  association call failed and the collect succeeded anyway. See task 073's log — the URL
  asserted a relationship the handler never verified.
- 2026-08-17 — Dev data left as found: the collected patrols restored to 4 racing, 0 members
  in care, all verification cases soft-deleted.
- 2026-08-17 — ✅ All criteria met bar the two-tab live check, which needs a human (task
  075's log explains why). Moving to done.
