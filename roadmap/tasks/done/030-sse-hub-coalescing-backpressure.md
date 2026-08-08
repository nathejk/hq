# 030 — SSE hub: coalescing, bounded buffers, overflow → resync

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

The fan-out core, in `go/internal/live/`. Holds connected clients and broadcasts
signals to them. No HTTP here — that is 032 — so this stays testable without a server.

### The decisions it exists to implement

**Backpressure is the one that bites silently.** A slow or sleeping client (laptop
lid closed, tab backgrounded) must not block the producer, and both obvious options
are wrong:

- *Unbounded buffer per client* → a sleeping tab becomes a memory leak on the API.
- *Bounded buffer that drops* → the client silently misses an invalidation and shows
  stale data indefinitely, with no error anywhere. **Worse than having no live updates
  at all**, because the UI still looks live.

So: bounded buffer, and on overflow **collapse rather than drop** — replace the
backlog with a single `resync` ("revalidate everything you hold"). The client already
runs that path on reconnect, so overflow degrades into well-tested behaviour instead
of a new one. A slow client gets coarser updates; never wrong ones.

**Coalesce once, centrally, then fan out.** Several consumers handle the same message
(`patrulje`, `patruljestatus` and `spejderstatus` all consume `patrulje.*.started`),
so a naive notifier emits duplicates. Debounce per `(entity, id)` over a short window
(~50–100ms), which also smooths mass operations like a whole-team collection.

Note replay is *not* a burst source: the boot gate (PRD 005) means no client is
connected while a build replays, so the whole history passes with nobody listening.
Coalescing is for live bursts only.

**Bound writes in time.** A write to a dead connection must not pin a goroutine
forever; the caller supplies a context and the hub respects it.

### Filtering

Clients may subscribe to a subset (`?entities=sos,patrulje`) and to one year. A signal
for another year or an unsubscribed entity must not reach them — cheap, and it keeps
a checkpoint-scan rush off screens that do not care.

## Acceptance Criteria

- [x] Hub with subscribe/unsubscribe returning a channel of signals per client
- [x] Signals coalesce per `(entity, id)` within a configurable window
- [x] A `resync` is delivered on subscribe, so a fresh client revalidates
- [x] Per-client buffer is bounded; on overflow the backlog collapses to one `resync`
      and **no signal is silently dropped without one**
- [x] Slow client cannot block the producer or other clients
- [x] Unsubscribed entities and other years are filtered out
- [x] Unsubscribing releases everything; no goroutine or channel leak
- [x] Writes respect a cancelled context
- [x] Tests cover: coalescing, overflow → resync, filtering, slow client isolation,
      unsubscribe cleanup, concurrent publish (`go test -race`)
- [x] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:42 — Task created. Depends on 029 for the signal type.
- 2026-08-08 19:18 — Picked up. Plan: one mutex; `Publish` debounces into a pending
  map keyed by `Signal.Key()`, a timer flushes and fans out; per-client buffered
  channel with non-blocking sends so a slow client can never block a publisher;
  `Subscribe(ctx, filter)` auto-cleans on context cancel. Tests with a short window,
  plus `-race`.
- 2026-08-08 19:30 — Added `internal/live/hub.go`. Design decisions:
  • **Every send is non-blocking, so broadcasting can run under the lock.** That is
    what makes "a slow client cannot block the publisher or another client" true by
    construction rather than by careful sequencing. `Publish` is called from projection
    consumers, whose progress must never depend on how promptly a browser reads a
    socket.
  • **Overflow collapses instead of dropping.** On a full buffer the backlog is drained
    and replaced with a single `resync`. Dropping would leave a client silently stale
    with no error anywhere — worse than no live updates, since the UI still looks live.
    Collapsing reuses the reconnect path the client already runs.
  • **`resync` bypasses coalescing**, and bypasses the filter on the way out: "you are
    out of date" is true regardless of what the client subscribed to, and delaying it by
    a window helps nobody.
  • **Close happens under the same mutex as sending**, which is what makes closing a
    client channel safe — no publisher can be mid-send on a client already removed from
    the map.
  • **`Subscribe(ctx, …)` self-cleans on context cancel**, so an HTTP handler needs no
    explicit unsubscribe and a disconnected browser cannot leak a subscription.
  • Coalescing is last-write-wins per `(entity, id)` — which is exactly why
    `Signal.Event` is documented as advisory: the surviving event name is arbitrary.
- 2026-08-08 19:38 — 12 tests. The overflow test deliberately asserts the *contract*
  ("a resync is present") rather than an exact delivered sequence, since how many
  signals survive a collapse is an implementation detail and pinning it would make the
  test brittle without making it stronger.
- 2026-08-08 19:42 — ✅ All gates: `go test -race -count=2 ./internal/live/...` clean
  (publishers really are concurrent — JetStream consumer callbacks), `gofmt` clean,
  `go vet` and `go tool staticcheck` clean across `./...`, full `go test ./...` green.
- 2026-08-08 19:44 — Completed. Next: 031 (notify decorator), which connects this to
  the projections.
