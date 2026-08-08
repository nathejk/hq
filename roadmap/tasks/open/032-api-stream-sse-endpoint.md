# 032 — `GET /api/stream` SSE endpoint

**Status:** open
**Priority:** high
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

## Description

The HTTP surface over the hub (030): one SSE endpoint carrying `entity.changed` and
`resync` events for the whole SPA.

### Decisions already settled (PRD 004 §8) — implement, do not relitigate

**Mount outside the `Metrics` middleware.** `/api/` is served as
`app.Metrics(app.authenticate(router))` (`routes.go:95`), and `metricsResponseWriter`
implements `Header`/`WriteHeader`/`Write`/`Unwrap` but **not `Flush`**
(`app/middleware.go:10-38`) — so a plain `w.(http.Flusher)` assertion fails and nothing
ever reaches the browser. `http.ServeMux` prefers the longer pattern, so:

```go
mux.Handle("/api/stream", app.authenticate(streamHandler)) // no Metrics
mux.Handle("/api/", app.Metrics(app.authenticate(router)))
```

It also stops an hours-long stream being recorded as one multi-hour request, which
would skew `total_processing_time_μs`.

**Use `http.NewResponseController(w)` anyway** — one line, works wrapped or not, and
immune to someone adding middleware later, which is exactly how this trap returns.

**Year as a query parameter, not a header.** `EventSource` cannot set headers, so the
`X-YearSlug` mechanism is unavailable. A missing `year` defaults to the current
calendar year, matching `YearSlug()` (`routes.go:102`).

**Entity filter** via `?entities=sos,patrulje`.

**Heartbeat every ~20s** (an SSE comment) so Traefik's idle timeout does not kill an
idle stream.

### Also required

- Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, and
  `X-Accel-Buffering: no` (harmless, helps if a buffering proxy ever appears).
- Send `event:` names so the client can dispatch on them and a future
  `version.changed` (PRD 005) is additive.
- Clean teardown when the client disconnects (`r.Context()` done) — no leaked
  subscription.
- Note the route sits on the same mux that serves the SPA through a fallback
  filesystem returning `index.html` for unknown paths (`SpaFileSystem`,
  `routes.go:110-123`). A typo in the pattern would not 404 — it would return HTML and
  the client would fail parsing an "event stream". Worth an explicit test that the
  content type is right.

## Acceptance Criteria

- [ ] `GET /api/stream` streams SSE, mounted outside `Metrics`
- [ ] Flushing works through the middleware chain (`http.NewResponseController`)
- [ ] Correct headers, including `text/event-stream` and `no-cache`
- [ ] `?year=` honoured; absent → current calendar year
- [ ] `?entities=` filters; absent → everything
- [ ] Named `event:` lines for `entity.changed` and `resync`
- [ ] Heartbeat comment on an interval, configurable
- [ ] Client disconnect unsubscribes from the hub with no leak
- [ ] OpenAPI annotation comment on the handler (per the layout skill)
- [ ] Handler test asserting content type, an initial `resync`, and a delivered signal
- [ ] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:46 — Task created. Depends on 030.
