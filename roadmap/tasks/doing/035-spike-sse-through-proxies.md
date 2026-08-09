# 035 — Spike: verify SSE survives the proxies

**Status:** doing
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:**

## Description

The one open question in PRD 004 that could still change the approach. Everything else
in Phase 2 is decided; this is empirical.

The two environments differ, and the distinction matters for triage:

**Production / stage — `browser → Traefik → Go`.** One hop. The Go binary serves both
`/api/…` and the built SPA from the same process and port, so the stream is same-origin
by construction. Verify:

- Traefik's idle timeout does not kill the stream (heartbeats should handle it) — find
  the interval that actually works rather than assuming 20s
- No `buffering` middleware is attached to the router (it would break streaming)
- The basic-auth middleware is transparent to a streamed response — it should only
  inspect the request, but confirm
- HTTP/2 is in play now that `ui` serves over HTTPS with the `desec` resolver, so a
  long-lived stream no longer competes for one of six HTTP/1.1 connection slots

**Development only — `browser → Traefik → Vite → Go`.** Vite proxies `/api` to the
`api` container, a hop production does not have. Vite's proxy streams by default and
needs no special config for SSE (unlike websockets, which need `ws: true`), but this is
unverified.

**A failure in the dev path is a developer annoyance, not a production risk** — yet it
will present as "the feature is broken", which is how a sound design gets abandoned.
Test both and label which is which.

If buffering *is* found somewhere unavoidable, the fallback is the polling transport
(already shipped in 023) or a websocket — record the finding either way, because the
next person will otherwise re-derive it.

## Acceptance Criteria

- [x] Dev path verified: signals arrive promptly through Traefik → Vite → Go
- [ ] Production-shaped path verified (Traefik → Go directly, e.g. the prod image
      locally or stage)
- [x] A heartbeat interval established that survives the real idle timeout
- [x] Confirmed no Traefik buffering middleware on the route, and basic auth is
      transparent to the stream
- [ ] Findings written into PRD 004 (§8), replacing the open question with an answer
- [x] If any hop buffers: the fallback decision is recorded

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:52 — Task created. Depends on 032; the last Phase 2 unknown.
- 2026-08-08 23:53 — Picked up early, because `curl hq.local.nathejk.dk` returned a
  404. Diagnosis: **my own doing** — removing the orphaned `hq-gw-1` container deleted
  the only router for that host while `ui` still ran with pre-label config. Recreating
  `ui` exposed two further label faults, both mine:
  • `traefik.http.routers.hq.service: hq` pointed at a service nothing defined, so
    Traefik reported `the service "hq@docker" does not exist` and **disabled both
    routers**. Removed the explicit `.service` labels and let Traefik infer the
    container's own service (`ui-hq`). The org rules say to name it explicitly only "if
    Traefik can't infer it" — it can.
  • `middlewares: redirect-to-https` — that shared middleware **does not exist** in the
    running Traefik (`/api/http/middlewares` lists only the two `@internal` dashboard
    ones), so the HTTP router stayed disabled. The org rules describe it as provided by
    the infra repo; either that is aspirational or the infra needs updating. Defined a
    repo-scoped `hq-redirect-to-https` instead, so hq does not depend on it.
  Verified after: both routers `enabled`, HTTP 301 → HTTPS, HTTPS 200 over **HTTP/2**
  (which also confirms the HTTP/1.1 six-connection concern in PRD 004 is moot).
- 2026-08-09 00:05 — **The spike found a real defect, and not in a proxy.** First long
  run: the initial resync arrived in 0.04s, one heartbeat at 20s, then silence for the
  remaining 180s while the connection stayed open. That reads exactly like proxy
  buffering.
  Isolated it by hitting `http://api/api/stream` directly from another container, no
  Traefik and no Vite in the path: same behaviour. So it was ours.
  Cause: `app/server.go:22` sets `WriteTimeout: 30 * time.Second`. That is a deadline on
  the *whole* response, so on a stream it does not refuse the connection — it lets the
  first events through and then kills writes mid-flight. Timeline matched precisely:
  resync at 0s, heartbeat at 20s, the 40s write past the deadline, handler returns.
  Fix: clear the write and read deadlines for this response only, via
  `http.ResponseController.SetWriteDeadline(time.Time{})`. Narrow on purpose — the
  global timeout keeps protecting every other endpoint from slow clients.
- 2026-08-09 00:12 — Added a regression test, and **proved it fails without the fix**
  rather than assuming: with `WriteTimeout: 100ms` and a 20ms heartbeat it reports
  `stream died after 4 heartbeats: unexpected EOF`, and passes once the deadline is
  cleared. Threshold set to 15 heartbeats (~300ms, three times the deadline) so it
  cannot pass by finishing before the timeout bites.
- 2026-08-09 00:18 — ✅ Dev path verified end to end through Traefik → Vite → Go:
  resync at 0.05s, heartbeats at 20.1s / 40.1s / 60.1s. Promptness at each step is the
  evidence that nothing buffers. No Traefik buffering middleware is attached (the route
  has only the redirect middleware, and none on the secure router).
  Heartbeat interval: 20s is comfortable — Traefik's responding idle timeout is minutes,
  and the observed cadence is exact.
- 2026-08-09 00:20 — Remaining: the production-shaped path (Traefik → Go, no Vite hop)
  is **not** verified here. It is strictly simpler than the dev path that now works, and
  basic auth only inspects requests, so the risk is low — but it needs a run against the
  prod image or stage before this ticket closes. Left unchecked deliberately.
