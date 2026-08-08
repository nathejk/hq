# 026 — Connection state exposed and displayed

**Status:** open
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 004: *"A dispatch desk that looks live but is frozen is worse than one that admits
it."* The transport must publish its state and the UI must show it. Depends on 023.

States: `live` · `reconnecting` · `polling` · `offline`.

Note what this is deliberately **not**: there is no "data may be stale" or "still
replaying" state. The API does not serve until its projections are fully caught up
(PRD 005), so a connected client is talking to a caught-up API by construction. The
only thing the operator ever needs to distinguish is **connected** from
**unavailable** — never *fresh* from *stale*. Aggregate figures such as PRD 001's
in-our-care count are therefore either correct or absent, never plausible-but-wrong.

While the transport is polling (Phase 1), the honest state is `polling`, not `live` —
the UI should not claim real-time behaviour the transport does not have.

### Notes

- Small and unobtrusive when healthy; clearly degraded when not. Not a modal, and
  nothing that steals focus mid-call.
- PrimeVue components are auto-imported; Aura preset; Danish UI text.
- Keep the indicator dumb: it renders state, it does not own reconnection policy.

## Acceptance Criteria

- [ ] Transport exposes a reactive connection state with the four values
- [ ] State is readable through a composable, not by importing the transport
- [ ] A small indicator component renders it, with Danish labels
- [ ] The indicator is visible on every page (placed in the app shell / navigation)
- [ ] Polling transport reports `polling`, never `live`
- [ ] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:51 — Task created. Depends on 023.
