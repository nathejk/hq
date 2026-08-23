# 093 — Row actions: modtaget and handover

**Status:** done
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Wire the buttons in `HoensegaardView.vue` (task 092) to the endpoints from task 091.

Buttons live **in the row**, not in a hidden menu — a tired volunteer should not have to
discover them (PRD 007 §7).

- *I bil* rows: **Modtaget i Hønsegården**.
- *I Hønsegården* rows: **Hentet af forældre** (`released`) and **Genforenet med patruljen**
  (`reunited`).
- *Afventer afhentning* rows: accepting is possible but sits behind a confirm, because it
  asserts an arrival the platform has no pickup for.
- *Afsluttet* rows: no actions.

**Genforenet med patruljen** is disabled with an explanatory tooltip when the patrol has no
active members: that patrol is discontinued and will not cross a line to be reunited at.

There is no undo. `released` and `reunited` are not interchangeable and a mistake is fixed
through the existing manual override on the patrol/case screens, which records itself as a
correction — that is the honest trail, and an undo button here would quietly erase it.

After a successful call, **do not mutate local state**: let the live signal refetch. The
write path and the read path must not be able to disagree about what happened.

Toasts for failures via `useToast`, following `PatruljeActiveView.vue`. A no-op returns
`200` and needs no toast — the screen already says what the operator wanted it to say.

## Acceptance Criteria

- [x] Buttons visible in the row per section as above
- [x] Accepting from *Afventer afhentning* requires a confirm
- [x] **Genforenet** disabled with a tooltip for a patrol with no active members
- [x] No undo action anywhere on the screen
- [x] Success relies on the live refetch, not local mutation
- [x] Failures produce a toast naming what failed
- [x] Double-press produces no error
- [x] **Carried from task 092:** verified live in two browser tabs — an acceptance in one tab
      moves the scout between sections in the other. Could not be done in 092 because nothing
      could yet *cause* a status change; this task is the first point at which it is possible.
      Verified at the transport rather than visually — see the log entry for what that means.

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 17:10 — Picked up. Needed one server addition first: `teamDiscontinued` on each row,
  derived from the patrol's `activeMemberCount` (a patrol whose last racing member has left is
  discontinued — there is no event for it, the count reaching zero *is* the fact). Sent as the
  fact rather than as `canReunite`, so the screen can explain why the button is disabled and the
  server is not deciding what the buttons are.
- 2026-08-23 17:25 — **Decision: which actions need two clicks is a judgement about cost, not a
  uniform policy.** Modtaget from *I bil* is one click — the most frequent action of the night,
  done with children in the doorway, and its mistake is cheap because the scout was arriving
  anyway. Modtaget from *Afventer afhentning* is two, since it asserts an arrival no pickup was
  recorded for. **Both handovers are two**, and that is the one I would defend hardest: they
  record that a child left our care, there is no undo, and a mis-click marks a scout released
  while they are asleep in a tent — the worst thing this screen can do.
- 2026-08-23 17:30 — Two clicks are an inline arm/confirm on the button, not a modal. No
  `ConfirmationService` is registered in this app (nothing in the SPA uses confirm dialogs), and
  a dialog is the wrong shape anyway: it steals focus and needs dismissing, and a volunteer
  holding a phone in one hand should not have to chase it. Arming one button disarms any other,
  so two half-pressed actions cannot sit on screen at once.
- 2026-08-23 17:35 — Errors surface the **server's** Danish message (`{error: {field: msg}}` or
  `{error: msg}`), because those are written for the crew — "modtag dem først" names the button
  that fixes the problem, which no generic client-side text could do.
- 2026-08-23 17:40 — On success: no local mutation of the cached payload, and an explicit
  `refresh()`. The refresh is not belt-and-braces — it is what keeps the screen correct when the
  live stream is degraded to polling or down, and `useLiveResource` collapses it into the
  signal-triggered revalidation when both happen together.
- 2026-08-23 17:50 — **The carried two-tab criterion, verified at the transport.** I cannot drive
  a browser, so instead I held a real SSE connection open against the running API
  (`GET /api/stream`) and performed an acceptance on another connection. The stream delivered:
  `event: entity.changed` / `{"entity":"spejder","id":"7da979ea…","year":"2026","event":"shelter.accepted"}`
  — `spejder` being exactly the token `HoensegaardView` declares in `dependsOn`, so a second tab
  revalidates and the scout moves sections. One signal, not two, despite both projections
  handling the event: the hub coalesced them, as designed. This is stronger evidence than a
  visual check of two windows, but it is **not** the same thing: it proves the signal is emitted
  and correctly addressed, not that Vue re-renders on receipt. The latter is covered by
  `useLiveResource.spec.ts`.
- 2026-08-23 17:55 — End-to-end against the dev stack: accepted a `waiting` scout with placering
  "Sovesalen" — `from: waiting → to: sheltered`, `teamStrength: 7`, she moved to *I Hønsegården*
  with her placering, and "Sovesalen" joined the suggestion list. `teamDiscontinued` reads true
  for the released scout's patrol (its strength is 0) and false for the others, so the Genforenet
  button is correctly disabled for exactly the patrol that will never reach the finish.
- 2026-08-23 17:56 — `vue-tsc` clean on my files; Go suite green. Moving to done.
