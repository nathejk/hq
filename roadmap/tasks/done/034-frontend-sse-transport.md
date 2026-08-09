# 034 — Frontend: SSE transport replacing polling

**Status:** done
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

Implement the `LiveTransport` interface (023) over `EventSource` and install it in
place of the polling transport. No page changes — that is the point of the seam.

### What it must do

- Connect to `/api/stream?year=…&entities=…` and dispatch on the SSE `event:` name, so
  a future `version.changed` (PRD 005) is additive rather than a format change.
- Report `live` when connected, `reconnecting` while retrying, `offline` when it has
  given up. The polling transport stays available as a fallback and keeps reporting
  `polling`.
- **Revalidate on every (re)connect.** Signals may have been missed while
  disconnected, so a connect emits `resync` — the same path polling and hub overflow
  use.
- Rely on `EventSource`'s built-in reconnection rather than hand-rolling backoff. This
  is the whole reason SSE was chosen over a websocket: the legacy `dims` channel
  hand-rolled backoff by re-entering `commit('dims/initialize')` and still could not
  recover missed messages.
- Jitter anything we *do* schedule ourselves: after a deploy every client reconnects at
  once and each then revalidates.
- Auth: cookies are sent automatically same-origin, so nothing to add. A 401 must
  surface rather than spin invisibly — note this path cannot be exercised in dev (no
  auth there), only against stage.

### Notes

- `setLiveTransport()` already exists for the swap; keep polling importable so a
  fallback remains possible if the spike (035) finds a proxy problem.
- Do not let the transport touch the cache directly — it publishes onto the bus, as
  polling does.

## Acceptance Criteria

- [x] An `EventSource`-based `LiveTransport` implementation
- [x] Installed as the default transport; polling still available
- [x] Dispatches on the SSE `event:` name; unknown names ignored, not fatal
- [x] Emits `resync` on connect and on every reconnect
- [x] State transitions: `live` on open, `reconnecting` on error-with-retry, `offline`
      when closed
- [x] `stop()` closes the connection and leaks nothing
- [x] Year change tears down and reconnects with the new year (via 025's sync)
- [x] Tests with a fake `EventSource`: open, message dispatch, error/reconnect,
      unknown event name, teardown
- [x] `npm run test:unit` green; no new `vue-tsc` errors in touched files; lint clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:50 — Task created. Depends on 032. Frontend counterpart of Phase 2.
- 2026-08-09 00:44 — Picked up. Plan: `plugins/live/sse.ts` implementing `LiveTransport`
  over `EventSource`, with the constructor injectable so it can be tested in the node
  environment (which has no `EventSource`). Installed as the default when the global
  exists, polling as the fallback.
- 2026-08-09 00:50 — Added `plugins/live/sse.ts`. Decisions:
  • **The EventSource constructor is injectable.** Tests run in a node environment with
    no `EventSource`, and adding jsdom just to test a transport would be a heavy way to
    get a fake. An `EventSourceLike` interface plus a `create` factory keeps the tests
    honest and the production path plain.
  • **`reconnecting` until `onopen`, never `live` optimistically.** Reporting live before
    the server has answered is exactly the dishonesty the indicator exists to prevent.
  • **`CLOSED` distinguishes given-up from retrying.** `EventSource` retries itself and
    sits in `CONNECTING` while it does, so only `CLOSED` becomes `offline` — the one
    state an operator must act on.
  • **The year is always sent, even when empty.** Empty means "current year", and
    letting the server apply its default would make a stream/REST year mismatch
    invisible — the failure 025 exists to prevent.
  • **A malformed frame is swallowed, not fatal**, and a signal naming no entity is
    dropped: it would invalidate nothing while looking like it worked. The next signal or
    reconnect resync recovers.
  • A deliberate restart (year change) reports `connect`, not `reconnect` — it is a fresh
    subscription rather than recovery from a dropped one.
- 2026-08-09 00:54 — Installed as the default in `plugins/live/index.ts` when
  `EventSource` exists, polling otherwise. Polling stays exported rather than becoming
  dead code: it is the documented fallback if a proxy ever buffers, which is the whole
  reason the transport seam exists.
- 2026-08-09 00:58 — ✅ 12 new tests (40 total across the frontend), `vue-tsc` 107 vs the
  109 baseline with none in my files, eslint clean apart from the pre-existing dead
  `setNav`, and `npm run build-only` succeeds — which also confirms no import cycle
  between `index.ts` and `sse.ts`.
- 2026-08-09 01:00 — Completed. **PRD 004 Phase 2 is functionally complete**: signals
  flow from a projection through the hub and the endpoint into the SPA's cache. Only
  035's production-path verification remains outstanding.
