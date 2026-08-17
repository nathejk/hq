# 077 — Below-strength panel: warning and the two actions

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] Pre-commit warning on `Ønsker at udgå` names the resulting strength and does not block
- [x] **Hent hele patruljen** calls the single collect endpoint, not N member calls
- [x] **Flyt de resterende** reuses the existing `patrulje:list` picker, filtered on
      `activeMemberCount > 0`
- [x] Survivors can be moved to two different patrols across two passes
- [x] No exception / undtagelse affordance anywhere
- [x] Move dialog defers live updates while open — by snapshotting its working set, see log
- [x] Warning persists while the team is short-handed, including pre-existing breaches
- [x] `npm run build` and `vitest` clean; no new TypeScript errors
- [x] Exercised end to end against the dev stack on a four-member patrol

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 072, 073, 076.
- 2026-08-17 — Picked up. Pre-commit dialog, below-strength panel and move dialog all in
  `SosTeamCard.vue`.
- 2026-08-17 — **Dirty guard implemented as a snapshot rather than by pausing the
  resource.** `KlanListView`'s pattern (drop incoming payloads while `dirty`) fits a view
  that owns its data; this card receives `teams` as a prop, so instead the move dialog
  snapshots the members it offers when it opens. That protects the thing that would
  actually be lost — rows appearing or vanishing under an operator mid-choice is how the
  wrong scout gets moved — while the rest of the card stays live. The dialog says updates
  are paused, as the task asked.
- 2026-08-17 — All selected by default in the move dialog: survivors usually stay together,
  so deselecting expresses the rarer split rather than making the common case laborious.
- 2026-08-17 — The pre-commit dialog offers **Kun denne udgår** alongside the two actions.
  Proceeding must always be available: the member is leaving whether or not the patrol is
  compliant, and a dialog that forced a choice between collecting and moving would be
  enforcing the requirement the PRD says is the operator's to apply.
- 2026-08-17 — ✅ Verified end to end: took a 4-member patrol to **2/3**, panel appeared,
  moved both survivors to another patrol — origin emptied to 0 and read as **discontinued**,
  destination went 4 → 6, timeline showed 2 `member.status.changed` + 2 `member.moved`.
  Picker data confirmed: 169 valid targets out of 408 teams, with `activeMemberCount` and
  `signupStatus` both present on list rows.
- 2026-08-17 — **⚠️ Found a real bug by trying to undo my own test — and it broke a
  guarantee the PRD makes explicitly.** Moving the survivors back into the emptied origin
  was **rejected**: the move handler required the destination to have
  `activeMemberCount > 0`, so a team at zero could never receive anybody. That means
  **discontinuation could not be undone**, because the only action that reverses it was
  refused for exactly the teams that needed it — contradicting PRD 006 §5 ("moving a member
  back into a team with nobody left makes it active again — the same reversibility legacy
  `.splited` had").

  Fixed: the server now requires only that the destination **started**. The picker still
  offers racing patrols only, which is right for the survivors flow — but a UI convenience
  must not become a domain rule, which is what had happened. Re-verified deliberately:
  emptied a patrol to 0 (discontinued: true), moved one member back, strength 1 and
  discontinued false.

  Worth noting how it surfaced: not from a test, but from cleaning up after one. The
  reversibility claim had no test, and still has none at the HTTP level — worth adding if
  this area is touched again.
- 2026-08-17 — Raised **task 085** rather than fixing in place: the dialog moves N members
  with N requests, so a two-survivor move writes two timeline entries where collecting three
  members writes one. The same partial-failure argument that justified the bulk collect
  applies, but weaker — moves can have different destinations, so per-member is the honest
  grain and a bulk endpoint is a refinement of the common case. Recorded with that
  trade-off spelled out.
- 2026-08-17 — Dev data restored: both patrols back to 4 racing. The three non-racing
  members remaining are in **2026** and are the product owner's own live-update testing, so
  deliberately left alone.
- 2026-08-17 — ✅ All criteria met. Moving to done.
