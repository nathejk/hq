# 075 — Member rows in SosTeamCard

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §7. The operator's main surface.

`vue/src/components/SosTeamCard.vue` ships with each associated patrol's identity and
contact only — its header comment says so explicitly and reserves the space below for this
work. PRD 001 deliberately shipped **no member rows**, because a list of names with nothing
next to them reads as a broken feature. This task introduces them together with the status
and actions that give them a purpose.

Each member row shows:

- name and contact
- **current status** with its timestamp and, where known, who accepted them
- row actions that depend on status:
  - `racing` → **Ønsker at udgå** (→ `waiting`)
  - `waiting` → **Fortsætter selv** (→ `racing`), as a normal, prominent action — *not*
    buried in an override menu. A scout getting their breath back is an ordinary outcome
    and saves a car being sent.
  - **`transit` onwards the row is read-only.** It reflects what the car and shelter have
    recorded and offers no button to advance or reverse them.
- secondary: **Flyt til anden patrulje**
- members `waiting` past the threshold highlighted

**No status override here.** Corrections live on the patrol page (task 084), because a
correction is not part of the call the operator is on — see PRD 006 §7.

## Notes

- **Status labels come from the backend**, never hardcoded in the view (PRD 006 §6).
- Use PrimeVue overlay/popover for the secondary menus, not `b-popover`.
- **Live updates:** the card's resource depends on `['sos:{id}', 'sos', 'spejder']`. The
  member token is `spejder` — the *event subject's* entity — **not** `spejderstatus`, which
  is a projection name and can never appear in a signal. PRD 004 §12 records this as the
  recurring silent defect; use the SPA's dev-only dependency validation while building.
- **Optimistic writes** for the member actions: an operator on the phone must never wait for
  a round trip. Use `composables/optimisticWrite.ts`. **But not for the resume action** — the
  server may legitimately reject it ("allerede hentet"), so show it pending and let the
  server answer.
- Wire `pending` to `:loading`; do not add a spinner. `pending` is true only when nothing is
  cached, so a revisited page must not flash.
- The correction path is deliberately elsewhere: it exists to record a reality another
  interface failed to write down, and putting it on a different screen from the live-call
  actions is a stronger separation than a visually-distinct button beside them. Its
  frequency is a tracked metric (PRD 006 §9).
- Until the car and shelter interfaces ship, that correction path is the only way a member
  leaves `waiting`, so it will be used more than it eventually should be. Do not design it
  away — and do not smuggle it back onto this card to save a click.

## Acceptance Criteria

- [ ] Member rows render per associated patrol, with status, timestamp and acceptor
- [ ] `Ønsker at udgå` on `racing` rows; `Fortsætter selv` prominent on `waiting` rows
- [ ] Rows in `transit`, `sheltered`, `reunited`, `released` offer no transition buttons
- [ ] `Flyt til anden patrulje` present and visibly secondary; **no override on this card**
- [ ] Status labels sourced from the backend
- [ ] `dependsOn: ['sos:{id}', 'sos', 'spejder']`, no dev-console dependency warnings
- [ ] Optimistic on all actions except resume
- [ ] Members `waiting` past the threshold visually flagged
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors
- [ ] Verified in two browser tabs: one operator's change appears in the other

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 067 and 072.
- 2026-08-17 — The status override was removed from this card by the decisions of
  2026-08-17: corrections belong on the patrol page (task 084). This card keeps only the
  actions that belong to a call in progress.
