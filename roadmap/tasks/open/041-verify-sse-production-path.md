# 041 — Verify SSE on the production path (Traefik → Go)

**Status:** open
**Priority:** high
**Created:** 2026-08-09
**Picked up by:**
**Started:**
**Completed:**

## Description

Carried out of PRD 004 at closure (§12). It is the one part of live updates that has
never been exercised in a production-shaped environment, and the failure mode is bad:
**live updates silently not working for operators**, on a feature they will come to
rely on during an event.

What *is* verified (task 035), on the dev path Traefik → Vite → Go:

- no buffering anywhere: resync at 0.05s, heartbeats at 20.1s / 40.1s / 60.1s
- HTTPS serves over HTTP/2, so the HTTP/1.1 six-connection limit is moot
- the one real blocker was **our own code**, not a proxy: the server's `WriteTimeout`
  (30s, `app/server.go:22`) is a deadline on the whole response, so the stream
  delivered its first events and then died silently mid-flight — which reads exactly
  like proxy buffering. Fixed per-response via `http.ResponseController`; regression
  test included and proven to fail without the fix.

Production has **one hop fewer** (no Vite), so it is likely fine. That is a reasonable
expectation, not evidence.

## What to check

- Heartbeats keep arriving past Traefik's idle/responding timeout — the interval is
  `live.DefaultHeartbeat` (20s) and the point is to stay under whatever prod enforces.
- No buffering middleware in the prod Traefik chain (compression is the usual culprit;
  `X-Accel-Buffering: no` is already set but only some proxies honour it).
- **Basic auth is transparent to `EventSource`.** Cookies are sent automatically
  same-origin, so this should hold — but stage/prod have basic auth and dev does not,
  so this specific combination has never run.
- A connection survives several minutes idle, then still delivers a signal promptly.
- Reconnect after a deliberate API restart: the browser retries by itself and the
  connect resync repopulates. This overlaps the blue/green switch in PRD 005 §
  "Draining SSE on switchover".
- The SPA's `ConnectionIndicator` reflects reality rather than optimism.

## How

Cheapest credible route is the **first stage deploy**: open the SPA, leave a page open
for a few minutes, edit something from a second tab, and watch the network panel for
the stream plus the heartbeat comments. If SSE is broken there, the indicator will say
`reconnecting` or `offline` rather than needing instrumentation.

If a proxy does buffer, the fallback already exists and needs no new code: the polling
transport is a deliberate, exported implementation behind the same interface, so
`setLiveTransport(createPollingTransport())` restores correctness while the proxy is
fixed.

## Acceptance Criteria

- [ ] A stream held open through prod/stage Traefik survives longer than its idle
      timeout, with heartbeats observed
- [ ] An entity change reaches a browser through the production path
- [ ] Basic auth confirmed transparent to `EventSource`
- [ ] Reconnect-after-restart observed to recover via the connect resync
- [ ] Findings recorded — especially any timeout or middleware that had to change, so
      the next environment is not a rediscovery

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 — Created at PRD 004's closure, carrying forward the one unverified item
  from its §12. Kept as its own high-priority ticket rather than left as an unticked
  PRD box, so it cannot be lost now that the PRD is closed.
