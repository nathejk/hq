# 026 — Connection state exposed and displayed

**Status:** done
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

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

- [x] Transport exposes a reactive connection state with the four values
- [x] State is readable through a composable, not by importing the transport
- [x] A small indicator component renders it, with Danish labels
- [x] The indicator is visible on every page (placed in the app shell / navigation)
- [x] Polling transport reports `polling`, never `live`
- [x] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:51 — Task created. Depends on 023.
- 2026-08-08 17:55 — Picked up. Plan: `useConnectionState()` composable over the
  transport's reactive state, plus a small `ConnectionState.vue` indicator placed in
  `App.vue` so it is on every page without touching Navigation (which has pre-existing
  type errors I would rather not disturb).
- 2026-08-08 18:00 — Added `composables/useConnectionState.ts`: wraps the transport's
  reactive state and owns the Danish labels/descriptions, plus `isHealthy` /
  `isDisconnected` so components do not compare string literals. Views never import
  the transport, which is what lets Phase 2 swap polling for SSE with no UI change.
- 2026-08-08 18:06 — Added `components/ConnectionIndicator.vue` and placed it in
  `App.vue`'s header, so it is present on every route including the fullbleed map view.
  Chose the app shell over `Navigation.vue` deliberately: Navigation carries three
  pre-existing type errors and editing it would mix unrelated risk into this ticket.
  Details: `role="status"` + an `aria-label` carrying both label and explanation; the
  dot pulses only when disconnected, and that animation is disabled under
  `prefers-reduced-motion` since colour already carries the meaning; the text label
  hides under 640px while the tooltip keeps it available.
- 2026-08-08 18:12 — Test failure exposed a real gap rather than a test bug:
  `createPollingTransport` and `emitSignal` were never re-exported from
  `plugins/live/index.ts` — only the types were. Phase 2 (constructing an SSE
  transport) and any test that installs a fake would both have hit this. Now exported.
- 2026-08-08 18:16 — ✅ 21 tests pass (5 new, driving a fake transport through all four
  states). `vue-tsc` 107, none in my files. Lint: 0 errors; the 3 warnings in `App.vue`
  (`RouterLink`, `HelloWorld`, `toast` unused) are pre-existing and left alone.
- 2026-08-08 18:18 — Completed. Next: 027 (optimistic writes), the last Phase 1 ticket.
