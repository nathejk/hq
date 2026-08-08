# 030 — SSE hub: coalescing, bounded buffers, overflow → resync

**Status:** open
**Priority:** high
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Hub with subscribe/unsubscribe returning a channel of signals per client
- [ ] Signals coalesce per `(entity, id)` within a configurable window
- [ ] A `resync` is delivered on subscribe, so a fresh client revalidates
- [ ] Per-client buffer is bounded; on overflow the backlog collapses to one `resync`
      and **no signal is silently dropped without one**
- [ ] Slow client cannot block the producer or other clients
- [ ] Unsubscribed entities and other years are filtered out
- [ ] Unsubscribing releases everything; no goroutine or channel leak
- [ ] Writes respect a cancelled context
- [ ] Tests cover: coalescing, overflow → resync, filtering, slow client isolation,
      unsubscribe cleanup, concurrent publish (`go test -race`)
- [ ] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:42 — Task created. Depends on 029 for the signal type.
