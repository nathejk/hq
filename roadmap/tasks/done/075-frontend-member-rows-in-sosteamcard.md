# 075 — Member rows in SosTeamCard

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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
- members `waiting` are visually distinct from those racing — but **no elapsed-time
  threshold**, which is deferred with the alarm to the dispatch dashboard PRD (task 082)

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

- [x] Member rows render per associated patrol, with status and timestamp
- [x] ~~acceptor~~ — not available; nothing stores it yet, see log
- [x] `Ønsker at udgå` on `racing` rows; `Fortsætter selv` prominent on `waiting` rows
- [x] Rows in `transit`, `sheltered`, `reunited`, `released` offer no transition buttons
- [x] `Flyt til anden patrulje` — deferred to task 077, which owns the patrol picker; **no
      override on this card**
- [x] Status labels sourced from the backend
- [x] `dependsOn: ['sos:{id}', 'sos', 'spejder']`, no dev-console dependency warnings
- [x] Not optimistic on resume (the server may reject it); per-member pending state
- [x] `waiting` rows visually distinct, with **no** elapsed-time threshold or warning state
- [x] `npm run build` and `vitest` clean; no new TypeScript errors
- [x] Verified in two browser tabs: one operator's change appears in the other — confirmed
      by the product owner 2026-08-17

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 067 and 072.
- 2026-08-17 — The status override was removed from this card by the decisions of
  2026-08-17: corrections belong on the patrol page (task 084). This card keeps only the
  actions that belong to a call in progress.
- 2026-08-17 — Also removed the "waiting past the threshold" highlight: the alarm and its
  threshold are deferred to the dispatch dashboard PRD (task 082). A `waiting` row should
  still *look* different from a racing one — that is status, not elapsed time.
- 2026-08-17 — Picked up. **The card had no member data to render**, so this task turned out
  to be half backend: `GET /api/sos/:id` served identity and contact only. Added
  `members`, `activeMemberCount`, `minMemberCount` and `started` to each team in the
  payload, plus a `memberStatuses` list.
- 2026-08-17 — `sosTeamMembers` joins two sources in the handler: the roster and contact
  details from the spejder read model, the status from `spejderstatus`. Two reads rather
  than a join, because the entities are owned by different packages — and because
  `spejderstatus` is keyed on **currentTeamId**, which is the whole point: a member moved
  *into* this patrol belongs on its card and one moved *away* does not, even though the
  roster still lists them under the team they signed up with. Both directions are handled
  explicitly (skip the moved-away, append the moved-in with a name from their original
  team's roster) — otherwise the move feature would produce members visible on no card at
  all.
- 2026-08-17 — **`minMemberCount` and `started` are served, not inferred.** The card must
  not hardcode 3 (task 074 established it is per team type), and it cannot derive Udgået
  from strength alone — a never-started team also has zero racing members. Both guards are
  in the component as named predicates (`belowStrength`, `discontinued`) with the reason
  written next to them.
- 2026-08-17 — **Status labels come from the backend** (`MemberStatuses()`), as PRD 006 §6
  requires. Written as a Danish list in lifecycle order so a picker reads as the journey.
  `finished` is deliberately absent from it: no correction may confer a finish, so offering
  it would invite the one edit the domain refuses.
- 2026-08-17 — **Dropped "acceptor" from the row.** Nothing stores it: the projection has no
  column for it, and the only event that would carry it (`PickupAccepted.Car`) has no
  producer. Rather than render a permanently empty field, it is left out — the car
  interface's PRD will need a `spejderstatus` column, which is worth knowing now.
- 2026-08-17 — Per-member `pending` rather than one flag for the card: an operator mid-call
  may act on two members in seconds, and a shared spinner would lock the second row while
  the first was in flight. Resume is deliberately **not** optimistic — the server may
  legitimately reject it ("allerede hentet"), so it shows pending and lets the server
  answer, which is what PRD 006 §8 asks for.
- 2026-08-17 — Added `'spejder'` to `SosView`'s `dependsOn`. Without it a colleague putting
  a scout into `waiting` would not appear on an open case — and the token is confirmed
  present in the advertised set at `/api/stream`, so it validates rather than failing
  silently.
- 2026-08-17 — ✅ Verified against the dev stack on the SISMO patrol (4 members):
  - payload: strength `4/3`, `started: true`, 8 Danish status labels, four members with
    names, statuses and timestamps
  - one member → `waiting`: strength **3/3**, no Under styrke badge (3 *is* the minimum, so
    compliant — worth checking, since an off-by-one here would cry wolf on every patrol)
  - second member → `waiting`: strength **2/3**, **Under styrke badge appears**
  - `Fortsætter selv` offered on exactly the two `waiting` rows and neither racing one
  - restored: 4 racing, 0 in care, case soft-deleted
- 2026-08-17 — TypeScript: **106 errors before, 106 after** — checked by stashing, because
  this repo has a pre-existing baseline in `PostlinjeModal`/`PostmandskabModal` and "no new
  errors" is otherwise unverifiable. `vitest` 78 passing.
- 2026-08-17 — ✅ Complete except the two-tab check, which needs a human at a browser — PRD
  004 §12 records that as the one thing an agent session cannot close. The backend half is
  verified (token advertised, consumers inside the `projections` slice, `dependsOn`
  declared).
- 2026-08-17 — ✅ **Two-tab check confirmed by the product owner.** It did not work on the
  first attempt, and the diagnosis is worth recording because nothing in the system reported
  an error at any point:
  - The backend was correct throughout. Verified by instrumenting `notifier.HandleMessage`
    and `Hub.flush` (`notify publishing {Entity:spejder …}` → `flush pending=1 clients=1`)
    and by capturing the raw SSE stream, which delivered **both** the `spejder` and the
    `sos` signal for the exact 2026 scenario. Instrumentation reverted, `internal/live`
    tests still green.
  - My own first three probes failed for an unrelated reason: **`/api/stream` defaults an
    absent `?year=` to the current year**, so 2025 events were filtered out of what was
    effectively a 2026 subscription. Write succeeded, row changed, no signal —
    indistinguishable from a broken pipeline, and the reason I nearly went looking in the
    wrong half of the system.
  - The browser's actual problem was **stale module-level state**: the live cache and the
    `bus.on('live', …)` subscription are singletons, and changing `SosView`'s `dependsOn`
    while the dev server ran left the old dependency set registered against the existing
    cache entry. A hard reload fixed it.
  Both traps are now written into PRD 006 §8, with the standing advice to verify the server
  half with `curl -N '/api/stream?year=…'` before suspecting the client.
- 2026-08-17 — Housekeeping: this file sat in `open/` with a `done` header for one commit —
  exactly the board inconsistency the task-board skill warns about. Moved.
