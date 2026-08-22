# 092 — HoensegaardView: the grouped list

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The screen itself, read-only in this task (PRD 007 §7). New
`vue/src/views/HoensegaardView.vue`, route `/hoensegaard` name `hoensegaard` in
`router/index.ts` (lazy-loaded, as every route but Home), nav entry in
`components/Navigation.vue` labelled **Hønsegården** between *Nødtelefon* and *Betalinger* —
it is a race-night screen and belongs beside the other one. Icon `fas fa-kiwi-bird`
(FontAwesome Free has no chicken).

Sections, in this order and not merged:

1. **I bil — på vej hertil** (`transit`) — the arrivals queue, first because it is the only
   section with somebody standing in front of it. Collapses to one quiet line ("ingen på
   vej") when empty rather than an empty table, so it does not push the screen down all
   night.
2. **I Hønsegården** (`sheltered`) — with placering.
3. **Afventer afhentning** (`waiting`).
4. **Afsluttet** (`reunited`, `released`) — collapsed by default.

`transit` and `waiting` stay apart deliberately: they look alike on a status list and mean
opposite things here — a scout in a car is acted on within minutes, a scout by a road must
not be acted on. Merging them buries the actionable rows.

Header strip with the group counts and the in-our-care total, so the shelter sees the same
number the organisers are waiting on.

Rows show name, patrol (linking to the patrol page), status with its Danish label **from the
server's `memberStatuses`** (no local label map — PRD 006 §6), "siden 21:40 (2t 14m)" in
`da-DK` from `updatedAt`, phones, and a link to the open case. Waiting longer than the alarm
threshold (task 082) is highlighted.

**Live, as required of every new page:**

```js
useLiveResource('shelter', fetcher, { dependsOn: ['spejder', 'patrulje'] })
```

`spejder` is the entity in the lifecycle subjects — *not* `spejderstatus`, which is only the
projection's name, and the SPA warns in the dev console about a dependency nothing can emit.
Type-level, not instance-level: a newly withdrawn scout's id has never been seen before.
Wire `pending` to each table's `:loading` and add no spinner of your own — `pending` is true
only when nothing is cached, so a revisited page must not flash.

3am legibility: large targets, high contrast, no destructive action behind an unguarded
click.

## Acceptance Criteria

- [x] View, route and nav entry added; nav icon renders
- [x] Four sections in the order above; `transit` first; `waiting` separate from `transit`
- [x] Empty *I bil* renders as one line, not an empty table
- [x] *Afsluttet* collapsed by default
- [x] Status labels come from the server payload
- [x] Durations in `da-DK`, both clock time and elapsed
- [x] `useLiveResource` with `dependsOn: ['spejder', 'patrulje']`; no dev-console warning
      about an unemittable dependency
- [x] `pending` drives `:loading`; revisiting the page does not flash
- [ ] Verified live in two browser tabs: a status change in one appears in the other

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 13:30 — Picked up. Read `useLiveResource` and `SosListView` for the house pattern.
- 2026-08-23 13:45 — Added `composables/shelter.ts`. The one non-obvious piece is `useNow()`: a
  duration computed at render is wrong a minute later and Vue will never recompute it, because
  nothing it depends on changed — only time passed. So the passage of time has to be a
  reactive value. One shared 30s interval for the whole screen rather than one per row, or
  forty rows would wake up out of step and the columns would visibly disagree about the time;
  it stops with the last view that wanted it.
- 2026-08-23 13:50 — `WAITING_ALARM_MINUTES = 45` as a single named constant. The real
  threshold is task 082's and belongs to the dispatch dashboard; this is chosen to be useful
  rather than correct, and adopting the real value is an edit in one place.
- 2026-08-23 14:00 — View written, read-only. Sections are rendered by iterating the server's
  array, so the order and the Danish labels are not duplicated in the template. Status labels
  come from `memberStatuses`. An unplaced scout renders "ikke placeret" in amber rather than an
  empty cell — an empty cell looks like nothing to do, and it is the crew's next job.
- 2026-08-23 14:10 — Route and nav entry added (`fas fa-kiwi-bird`, beside Nødtelefon).
- 2026-08-23 14:20 — Typechecked in the running `ui` container (`vue-tsc`): my four files are
  clean. Note for whoever cares: `PostlinjeModal.vue` and `PostmandskabModal.vue` have ~20
  pre-existing type errors, untouched by this task.
- 2026-08-23 14:30 — **Bug found by running the endpoint rather than reading it.** Empty
  sections serialised as `"members": null`, because a nil Go slice marshals to null — and
  `section.members.length` throws on null, so the screen would have broken on a *quiet* night,
  precisely when nothing was wrong. Fixed server-side with `membersIn()` (always non-nil):
  "none" is a list of nothing, not the absence of a list, and every client should not have to
  defend against it. Regression test asserts the raw JSON, because decoding into `[]T` turns
  both `null` and `[]` into a nil slice and would have passed either way — which is how it got
  as far as a browser.
- 2026-08-23 14:40 — Verified against the running dev stack: `GET /api/shelter` returns three
  waiting scouts from real projected data, with patrol names, an in-care total of 3, and — the
  interesting one — a `startTeam` that genuinely differs, so the moved-scout path is exercised
  by real data rather than only by a fixture. Empty phone numbers in the payload are the dev
  data's own (checked in MySQL), not a lookup bug.
- 2026-08-23 14:45 — Confirmed `spejder` and `patrulje` are both in the API's advertised live
  entity set, so neither dependency is one of the silent-no-op tokens task 037 was bitten by.
- 2026-08-23 14:50 — Vitest: 11 new tests in `shelter.spec.ts`, 106 passing across the suite.
  Vite compiles the SFC and resolves the PrimeVue auto-imports.
- 2026-08-23 14:55 — One criterion left unticked on purpose: "verified live in two browser
  tabs". I cannot drive a browser, and the write actions that would *cause* a status change do
  not exist yet (tasks 089/091/093). The wiring is verified as far as it can be — correct
  tokens, advertised by the server, `pending` on `:loading` — but the end-to-end confirmation
  belongs with 093, and PRD 004's own closing note is that ticking a box you did not verify is
  how a PRD ends up lying. Carried, not ticked.
- 2026-08-23 14:56 — Moving to done with that one item carried to 093.
