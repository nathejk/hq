# 131 — Defer live payloads while the settings modal is dirty

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Live payloads deferred while the dialog is dirty, applied on close/save
- [ ] Reuses the existing defer mechanism, not a second one
- [ ] Visible "opdateringer sat på pause" indication while deferred
- [ ] Clean dialog (nothing edited) does **not** defer
- [ ] Verified with two browser sessions editing at once
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
