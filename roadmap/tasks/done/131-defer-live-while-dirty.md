# 131 — Defer live payloads while the settings modal is dirty

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §6 (non-functional). Depends on 127. `.rules` → Live updates: *"A page holding
unsaved state must not be updated underneath the operator."*

`KortView` already defers incoming payloads while marker dragging is in progress
(`applyDeferred`, `syncMapIfDeferred`). The settings dialog holds unsaved state too, so it
must join the **same** mechanism rather than adding a second, parallel one — two competing
defer flags is how one of them ends up wrong.

While the dialog is open and dirty, incoming payloads are held and applied when the edit
ends, and the UI says updates are paused. Discarding an operator's half-entered map because
someone else renamed a checkpoint is far worse than briefly showing older data.

## Acceptance Criteria

- [x] Live payloads deferred while the dialog is dirty, applied on close/save
- [x] Reuses the existing defer mechanism, not a second one
- [x] Visible "opdateringer sat på pause" indication while deferred
- [x] Clean dialog (nothing edited) does **not** defer
- [ ] Verified with two browser sessions editing at once — **not done**, no way to drive a browser
      from here. The mechanism is the tested `useDeferredApply`, and the trim trap below was found
      and fixed by reasoning plus an API check.
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: `composables/useDeferredApply.ts` already exists and is the house
  mechanism — but `KortView` still has its own `applyDeferred` flag from before it landed. Migrating
  the view onto the shared composable is the way to satisfy "not a second mechanism", rather than
  adding a third condition to the home-rolled one.
- 2026-09-03 — Migrated `KortView` off its own `applyDeferred` flag onto `useDeferredApply`, and
  folded **map readiness into the pause condition** rather than keeping the `syncMap()` call in
  `onMounted`. That removes the second code path entirely: a warm cache is held because the map does
  not exist yet, and applied the moment it does. Both `syncMapIfDeferred()` call sites in the
  edit-mode exits are gone — leaving edit mode flips the condition, and the composable cannot forget
  to ask.
- 2026-09-03 — **Found a bug the task description did not anticipate, and it was the important one.**
  Deferring the Leaflet markers would not have protected the dialog at all: the dialog reads the
  sheets straight from the live cache, and its three buffers each had their own
  `watch(selected, …)`. So any incoming payload re-ran the load and **wiped the field under the
  cursor** — triggered by another operator renaming any sheet, or by this operator's own save of a
  different sheet. Replaced all three watchers with one `loadBuffers` behind the same
  `useDeferredApply`, guarded by `anyDirty`.
- 2026-09-03 — Selection changes flow through that same deferred apply, which is safe only because
  `select()` already refuses to change the selection while dirty — so a selection change is a clean
  moment by construction. Worth stating, because it would otherwise look like selection could be
  swallowed while paused.
- 2026-09-03 — **Second trap, from the fix itself:** with unsaved state now pausing live updates, any
  field the server normalises but the client does not would leave the draft permanently unequal to
  the saved row — a successful save, then "ugemte ændringer" forever, and a **frozen map** as a
  consequence. Confirmed against the API that it trims names (`"  Kort 1  "` → `"Kort 1"`), so the
  client now trims before both comparing and sending, for sheet names and set names. Client-side
  normalisation has to match the server's for exactly the fields it touches.
- 2026-09-03 — Added the „Opdateringer sat på pause — der er ugemte ændringer” banner, driven by
  `updatesWaiting` from the composable. A page that has taught its operator to trust it is current
  owes them a word the one time it deliberately is not — and it doubles as the visible cue for why a
  colleague's change has not appeared.
- 2026-09-03 — Completed. 169 composable tests pass (the suite, not just this feature), `vite build`
  clean, no type errors in the feature's files.
