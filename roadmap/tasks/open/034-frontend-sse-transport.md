# 034 — Frontend: SSE transport replacing polling

**Status:** open
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] An `EventSource`-based `LiveTransport` implementation
- [ ] Installed as the default transport; polling still available
- [ ] Dispatches on the SSE `event:` name; unknown names ignored, not fatal
- [ ] Emits `resync` on connect and on every reconnect
- [ ] State transitions: `live` on open, `reconnecting` on error-with-retry, `offline`
      when closed
- [ ] `stop()` closes the connection and leaks nothing
- [ ] Year change tears down and reconnects with the new year (via 025's sync)
- [ ] Tests with a fake `EventSource`: open, message dispatch, error/reconnect,
      unknown event name, teardown
- [ ] `npm run test:unit` green; no new `vue-tsc` errors in touched files; lint clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:50 — Task created. Depends on 032. Frontend counterpart of Phase 2.
