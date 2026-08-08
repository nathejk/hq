# 035 — Spike: verify SSE survives the proxies

**Status:** open
**Priority:** high
**Created:** 2026-08-08
**Picked up by:**
**Started:**
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

- [ ] Dev path verified: signals arrive promptly through Traefik → Vite → Go
- [ ] Production-shaped path verified (Traefik → Go directly, e.g. the prod image
      locally or stage)
- [ ] A heartbeat interval established that survives the real idle timeout
- [ ] Confirmed no Traefik buffering middleware on the route, and basic auth is
      transparent to the stream
- [ ] Findings written into PRD 004 (§8), replacing the open question with an answer
- [ ] If any hop buffers: the fallback decision is recorded

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:52 — Task created. Depends on 032; the last Phase 2 unknown.
