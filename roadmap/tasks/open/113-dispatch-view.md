# 113 — `DispatchView` — queue and tour panes

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Route `/koersel` lazy-loaded, navigation entry added
- [ ] Both panes render; drag-and-drop into a tour and reorder within a tour persist via the APIs
- [ ] `useLiveResource` with the tokens above; no `onMounted` + `http.get`
- [ ] Updates deferred while a tour arrangement is dirty, with an on-screen notice
- [ ] Waiting clocks advance without interaction, via the shared `useNow()`
- [ ] No dev-console warning about an unemittable dependency
- [ ] Readable at narrow widths

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
