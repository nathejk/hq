# 077 — Below-strength panel: warning and the two actions

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §6 and §7. The actions an operator takes when a patrol drops below three
racing members.

**Pre-commit warning.** Confirming `Ønsker at udgå` for a member whose departure would put
the team below the minimum warns *before* committing, naming the resulting strength
("Patruljen har kun 2 tilbage"). It changes the conversation the operator is having on the
phone. **It does not block the transition** — the member is leaving whether or not the team
is compliant.

**Below-strength panel** on the same card, whenever a team on the case is short-handed:

- **Hent hele patruljen** — every remaining `racing` member → `waiting`, one action
  (task 073)
- **Flyt de resterende** — move survivors to another patrol, picking the destination with
  the **same filtered patrol picker the card already uses to associate a team**

Plus, from the pre-commit dialog, simply proceeding.

## Notes

- **The move picker needs no endpoint.** `SosTeamCard.vue` already filters the SPA's live
  `patrulje:list` cache (min two characters, capped at ten, matching number / name / group)
  for team association — reuse that, filtered additionally on `activeMemberCount > 0`. Same
  cache key, so opening it costs no request and the list cannot be stale in one place and
  fresh in another. There is deliberately no `reassign-candidates` route.
- **Moving is per member.** Usually all survivors go to the same patrol — make that the
  convenient default — but two may go to two different ones, so the flow must allow
  repeating it with a different destination. Do not model it as one group move.
- **There is nothing to grant.** No **Tillad undtagelse** button, no reason field, no
  approval step: breaches are recorded, not handled (PRD 006 §11 Decisions). The warning
  stays for as long as the team is short-handed and needs no acknowledgement — the timeline
  below it says how the team got there.
- **Dirty-guard the move dialog.** It holds unsaved state, so it must defer incoming live
  payloads while open and say on screen that updates are paused, as `KlanListView.vue` does.
  The three-line `useLiveResource` adoption suits read-only surfaces only.
- The warning must keep showing while it is true, so an operator taking over a shift sees the
  state of play — including a team that was already below three before the case was opened.
  The warning reflects current strength, not only the transition that caused it.
- There is effectively always a valid destination (any racing patrol in the year), so the
  flow cannot dead-end for lack of candidates. What it can dead-end on is nobody in the
  field having agreed one yet — in which case the operator has not been told a destination
  and the breach simply stays visible.

## Acceptance Criteria

- [ ] Pre-commit warning on `Ønsker at udgå` names the resulting strength and does not block
- [ ] **Hent hele patruljen** calls the single collect endpoint, not N member calls
- [ ] **Flyt de resterende** reuses the existing `patrulje:list` picker, filtered on
      `activeMemberCount > 0`
- [ ] Survivors can be moved to two different patrols across two passes
- [ ] No exception / undtagelse affordance anywhere
- [ ] Move dialog defers live updates while open and says so
- [ ] Warning persists while the team is short-handed, including pre-existing breaches
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors
- [ ] Exercised end to end against the dev stack on a four-member patrol

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 072, 073, 076.
