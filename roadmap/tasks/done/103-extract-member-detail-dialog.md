# 103 — Extract MemberDetailDialog and make it live

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] `MemberDetailDialog.vue` exists, taking `memberId` and `teamId`
- [x] `SosTeamCard.vue` uses it; its own behaviour unchanged, including the status badge and the
      waiting/resume buttons reading the live row
- [x] Loads via `useLiveResource`, `dependsOn` the member instance and the joined types
- [x] `pending` drives the modal's loading state; reopening a scout does not flash
- [x] `SosTeamCard.vue` is meaningfully shorter
- [x] `vue-tsc` clean
- [ ] **Manual check of the case card's member flows — not done, needs a human.** Open a case with
      an associated patrol, open a member, and confirm: the status badge, **Udgår** (including the
      under-strength warning), **Fortsætter selv**, and **Skift** with its destination list. These
      are the nødtelefon's most sensitive flows and I cannot drive a browser; everything else here
      is verified by types, compilation and reading.

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 23:50 — Extracted to `MemberDetailDialog.vue` (300 lines); `SosTeamCard.vue` is
  956 → 816. The dialog owns the *view* and fetching; **actions come in through a slot**, because the
  nødtelefon withdraws and switches, the shelter receives and hands over, and the patrol page does
  neither. Importing every host's commands here behind flags would have made one shared component
  into three in a trenchcoat.
- 2026-08-24 00:00 — Now live: `useLiveResource` keyed `member:{id}`, depending on the **instance**
  `spejder:{id}` plus `patrulje` for the joined team names. That replaces the card's watch on
  `detailMember.status` — and is strictly better than it was, because the watch only fired for
  changes visible in *this card's* row and so missed anything done from another screen. Two crew
  members can now see each other's work on the same scout, which is the precondition for the note
  trail being shared at all.
- 2026-08-24 00:05 — The card still needs two facts from the payload (the switch dialog's "from"
  patrol, and the strength the withdrawal warning is measured against), so the dialog emits `loaded`
  and the card mirrors it. A second fetch was the alternative and would have given two copies that
  disagree the moment one revalidates.
- 2026-08-24 00:10 — **Caught two silent regressions in my own extraction, both by reading the
  original rather than by any tool:**
  1. I had rewritten `statusColour` with four arms instead of six, dropping `info` and `secondary` —
     which would have quietly changed the colour of two statuses in the history timeline.
  2. I had rewritten `birthday` as `2-digit` day and month, so "5. december 2026" would have become
     "05.12.2026".
  Both now match the original exactly. This is the failure mode a refactor is supposed to be
  incapable of, and neither typechecking nor compilation would have said a word.
- 2026-08-24 00:15 — `statusColour` went to `composables/sos.ts` as `memberStatusColour` rather than
  being duplicated: the card's member rows need the same mapping, and a copy in each component is
  precisely how the two would drift.
- 2026-08-24 00:18 — Removed the dead code the extraction left in the card: `statusSeverity`,
  `memberEventPhrase` and `formatDateTime` had no remaining callers there.
- 2026-08-24 00:20 — The dialog also hosts `MemberNotes` (task 104) and forwards its `dirty` event
  outward. It deliberately defers nothing itself: `MemberNotes` owns the textarea's text and
  `v-if="detail"` stays true across a revalidation, so the component is never unmounted — what needs
  deferring is the *list behind* the dialog, which is task 105's business.
- 2026-08-24 00:25 — `vue-tsc` clean, both SFCs compile through Vite, 108 vitest passing.
  **One criterion left unticked and carried into the open:** the case card's member flows (Udgår with
  its under-strength warning, Fortsætter selv, Skift with its destination list) need a human to click
  through. I cannot drive a browser, and this is the nødtelefon's most sensitive screen — saying so is
  worth more than a ticked box.
- 2026-08-24 00:26 — Moving to done with that item flagged for the owner.
