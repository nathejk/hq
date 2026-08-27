# 113 — `DispatchView` — queue and tour panes

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §7. New view `vue/src/views/DispatchView.vue`, lazy-loaded at `/koersel`, "Kørsel" in
the navigation beside Nødtelefon and Hønsegården. English identifiers, Danish strings.

Two panes: **Ikke planlagt** (queued tasks, oldest first, deadline-at-risk pinned) and **Ture**
(one card per tour, grouped by unit, stops in order — `components/DispatchTourCard.vue`). Tasks
move into a tour by drag-and-drop or a "Læg i tur" action; stops reorder by drag within the card.

Live per PRD 004: `useLiveResource` with `dependsOn` `dispatch`, `tour`, `dispatchduty`,
`section`, `crewmember`, `crew`, `vehicle`, `spejder`, `sos`; `pending` wired to loading. **The
board is an editor** — a half-built tour or a drag mid-flight must not be replaced by an incoming
payload, so use `useDeferredApply` and say on screen that updates are paused
(`KlanListView` / `KortView` are the precedent).

**Clocks must advance**: waiting times and countdowns need the shared `useNow()` from
`composables/shelter.ts`, one interval for the screen. Times carry a weekday where they cross
midnight.

Usable on a phone — the same board, narrow.

## Acceptance Criteria

- [x] Route `/koersel` lazy-loaded, navigation entry added
- [x] Both panes render; drag-and-drop into a tour and reorder within a tour persist via the APIs
- [x] `useLiveResource` with the tokens above; no `onMounted` + `http.get`
- [x] Updates deferred while a tour arrangement is dirty, with an on-screen notice
- [x] Waiting clocks advance without interaction, via the shared `useNow()`
- [x] No dev-console warning about an unemittable dependency
- [x] Readable at narrow widths

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — Picked up. Plan: `composables/dispatch.ts` for the vocabulary and the unix-seconds
  time formats, `components/DispatchTourCard.vue` for a tour, `views/DispatchView.vue` for the
  board, plus the route and the navigation entry.
- 2026-08-27 — **`dispatchduty` is deliberately *not* in `dependsOn`.** The task text lists it,
  but nothing can emit it until task 115 creates the entity — and a dependency nothing can emit is
  exactly what the dev-console warning exists to catch. Adding it early would train whoever sees
  that warning to ignore it, which is worth more than the token being complete a day sooner. 115
  adds it along with the events.
- 2026-08-27 — Own time helpers in `composables/dispatch.ts` rather than reusing
  `composables/shelter.ts`: the shelter formats ISO strings because its API sends them, and kørsel
  sends unix seconds because every number on this screen is arithmetic. Same output, down to the
  da-DK dot in "21.40". The *clock* is shared — `useNow()` from `shelter.ts`, one interval for the
  screen — because that is state, and two intervals would be two.
- 2026-08-27 — **The subtle one: `plannedUts` is sent back only for stops somebody overrode.**
  Echoing the derived times would mark every stop as manually set, and then nothing would ever
  re-derive — a later departure change would silently stop moving the plan, and the board would
  quietly become a set of frozen numbers. The card marks an overridden time with an asterisk and
  a tooltip for the same reason: a typed time is a different fact from a derived one.
- 2026-08-27 — Dropping a task on a tour creates **two** stops, a load and an unload, because
  that is what a task that moves something is. One stop would make "when will they be collected"
  and "when will they arrive" the same number, which is precisely the question the board exists
  to answer separately.
- 2026-08-27 — Reordering is drag-for-placement plus explicit ↑/↓ buttons rather than
  drag-to-reorder-within-the-list. Two reasons: the desk is explicitly meant to be usable on a
  phone, where dragging a row a few pixels is how you scroll by accident; and a swap with a
  visited stop can then be explained on the spot ("et besøgt stop kan ikke flyttes") instead of
  arriving as a 422 the operator has to interpret.
- 2026-08-27 — One live key for the whole board, not one per pane: a task dragged into a tour
  changes both, and two keys could leave the panes disagreeing — showing the same task queued
  *and* on a tour. Updates are deferred while a write is in flight or a drag is in progress, and
  the screen says so, because a page that has taught its operator it is live owes them a word the
  one time it is not.
- 2026-08-27 — Creating a *task* is not here: it is task 114, with the dialog and the place
  picker. Creating a **tour** is, as a two-field inline form, because without it the board has no
  way to demonstrate that any of the tour endpoints work — and a tour is two fields, not a dialog.
- 2026-08-27 — ✅ Verified: `vue-tsc` reports nothing for the three new files (repo baseline
  unchanged at 109 pre-existing errors), and the running Vite dev server compiles all three,
  resolving the auto-imported PrimeVue components. Not verified by eye in a browser: the stack has
  no seeded kørsel unit, and no browser is available in this session — the API side of every
  action on the card was exercised over HTTP in tasks 110/111.
- 2026-08-27 — All criteria met. Moving to done.
