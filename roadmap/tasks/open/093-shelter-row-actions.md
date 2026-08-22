# 093 — Row actions: modtaget and handover

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Buttons visible in the row per section as above
- [ ] Accepting from *Afventer afhentning* requires a confirm
- [ ] **Genforenet** disabled with a tooltip for a patrol with no active members
- [ ] No undo action anywhere on the screen
- [ ] Success relies on the live refetch, not local mutation
- [ ] Failures produce a toast naming what failed
- [ ] Double-press produces no error
- [ ] **Carried from task 092:** verified live in two browser tabs — an acceptance in one tab
      moves the scout between sections in the other. Could not be done in 092 because nothing
      could yet *cause* a status change; this task is the first point at which it is possible.

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
