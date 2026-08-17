# 076 — Team strength and Udgået badge on the case card

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Strength shown per patrol on the card
- [ ] **Under styrke** appears below the configured minimum and clears on its own
- [ ] **Udgået** badge when the team started and `activeMemberCount` is zero — verified
      *not* to appear for a year that has not raced yet, and not for abandoned signups
- [ ] Minimum read from the API's team config, not hardcoded
- [ ] `dependsOn` includes `'spejder'`; verified live by changing a member in another tab
- [ ] No dismiss / acknowledge affordance on the warning
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 066 and 075.
- 2026-08-17 — Task 066 amended the discontinued predicate before this was picked up:
  `activeMemberCount === 0` alone badges every not-yet-raced team as udgået. Description
  and criteria updated; see PRD 006 §11 Decisions.
