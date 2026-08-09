# 032 — `GET /api/stream` SSE endpoint

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

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

- [x] `GET /api/stream` streams SSE, mounted outside `Metrics`
- [x] Flushing works through the middleware chain (`http.NewResponseController`)
- [x] Correct headers, including `text/event-stream` and `no-cache`
- [x] `?year=` honoured; absent → current calendar year
- [x] `?entities=` filters; absent → everything
- [x] Named `event:` lines for `entity.changed` and `resync`
- [x] Heartbeat comment on an interval, configurable
- [x] Client disconnect unsubscribes from the hub with no leak
- [x] OpenAPI annotation comment on the handler (per the layout skill)
- [x] Handler test asserting content type, an initial `resync`, and a delivered signal
- [x] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:46 — Task created. Depends on 030.
- 2026-08-08 20:08 — Picked up. Plan: streaming mechanics as an `http.Handler` in
  `internal/live` (unit-testable with `httptest`), a thin `cmd/api/stream.go` adapter
  wiring the year default, and the route registered on the mux **outside** `Metrics`.
  Hub constructed in `main.go` now, with no consumers notified yet, so the spike (035)
  can exercise the endpoint before 033.
- 2026-08-08 20:14 — Added `internal/live/http.go` (`StreamHandler`). Decisions:
  • **Streaming mechanics in `internal/live`, not `cmd/api`.** The layout skill puts
    handlers in `cmd/api`, but its `app.*` helpers are all JSON-envelope oriented and
    none of them apply to a stream. Keeping the loop here makes it testable with
    `httptest`; `cmd/api/stream.go` stays a five-line adapter supplying only what the
    request context knows — what "current year" means.
  • **`DefaultYear` is injected as a func** rather than computed in the package, so
    tests are not time-dependent and this package holds no opinion about "current".
  • **`http.NewResponseController`** for flushing, as decided — works whether or not
    the writer is wrapped, so the trap cannot return if middleware is added later.
  • Event name is the signal type, so the client dispatches on `event:` and a future
    signal kind is additive.
  • Heartbeat is a comment line (`: keep-alive`), ignored by `EventSource` but enough
    to stop an intermediary judging the connection idle.
- 2026-08-08 20:20 — Route registered in `routes.go` as
  `mux.Handle("/api/stream", app.authenticate(app.streamHandler()))`, above the
  `/api/` entry — `ServeMux` prefers the longer pattern. Hub constructed in `main.go`
  with `defer livehub.Close()`, and added to the `application` struct as a dependency
  rather than a config field.
- 2026-08-08 20:26 — Tests: 6 in `internal/live/http_test.go` against a real
  `httptest.Server` (a `ResponseRecorder` would buffer and prove nothing about
  flushing), plus one in `cmd/api/stream_test.go` that goes through `app.routes()` —
  the only way to actually prove "mounted outside Metrics" and "flushes through the
  middleware chain", which are the two criteria most easily claimed without evidence.
  The content-type assertion doubles as a guard against the SPA fallback: a
  mis-registered pattern would return `index.html` rather than 404, and the client
  would fail parsing HTML as an event stream.
- 2026-08-08 20:30 — ✅ All gates: `go test ./...` green (cmd/api + internal/live),
  `-race` clean, `gofmt`/`vet`/`staticcheck` clean across `./...`, `go build ./...` OK.
- 2026-08-08 20:32 — Completed. The endpoint is live and self-sufficient: it delivers
  the connect-time resync without any consumer being wrapped yet, which is exactly what
  035 needs to test the proxies before 033/034 are built.
- 2026-08-09 00:14 — Follow-up under task 035: this endpoint had a defect not caught by
  its own tests. The server's `WriteTimeout` (app/server.go:22) is a deadline on the
  whole response, so the stream delivered its resync and one heartbeat and then died
  silently mid-flight. Fixed in `internal/live/http.go` by clearing the write/read
  deadlines per response via `http.ResponseController`, with a regression test proven to
  fail without it. See 035's log for the diagnosis.
