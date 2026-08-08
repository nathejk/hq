# 023 — Live-update transport interface + polling implementation

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

First slice of PRD 004 Phase 1 (`roadmap/prd/004-live-updates-spa.md`). Introduce the
seam that every later piece depends on: a transport that tells the client *what
changed*, with a polling implementation behind it so no backend work is needed yet.

PRD 004 decided SSE is the eventual transport, but Phase 1 is deliberately
frontend-only. The interface must express **both** semantics:

- **SSE (later):** reports which `(entity, id)` changed.
- **Polling (now):** cannot know, so it says "something might have changed" and the
  cache revalidates everything mounted.

Getting that asymmetry into the interface now is the whole point of the ticket — it
is what lets SSE drop in later without touching a single page.

Also in scope, because it blocks the fan-out decision: `vue/src/plugins/bus/index.ts`
does `mitt<TBusEvent>()` with **`TBusEvent` never declared** (a pre-existing type
error). PRD 004 decided to reuse this bus rather than add a second emitter, so the
event map needs declaring properly.

Deliberately **not** in scope: the cache itself (024), year switching (025),
connection-state UI (026), optimistic writes (027), and anything server-side.

### Design notes

- Signal shape mirrors the wire format PRD 004 specifies:
  `{ entity, id?, year, event? }`. `event` is advisory only — coalescing makes it
  arbitrary, so nothing may branch on it.
- A `resync` signal means "you missed something, revalidate everything you hold".
  The polling implementation emits exactly this on every tick, which is why the
  cache needs no polling-specific code path.
- Dispatch on the signal *type* so a future `version.changed` (PRD 005) is additive.
- No page may import the transport directly; it publishes onto the bus.

## Acceptance Criteria

- [x] `TBusEvent` declared, typed for the live-update signals, with the existing
      `toast` event preserved — `plugins/bus/index.ts` no longer errors
- [x] A transport interface exists with `start()` / `stop()` and publishes signals
      onto the mitt bus, not to callers directly
- [x] A polling implementation of that interface, with a configurable interval,
      emitting `resync` per tick
- [x] Signal types are exported and documented: `entity.changed` and `resync`
- [x] Starting twice is safe (idempotent), and `stop()` fully tears down timers
- [x] No new `vue-tsc` errors in the files touched (baseline is 109 pre-existing)
- [x] `npm run lint` clean for the new files

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:45 — Task created. Split from PRD 004 Phase 1; first of 023–027.
- 2026-08-08 15:55 — Picked up. Plan: declare the bus event map first (it blocks using
  the bus at all), then a `transport` module exporting the interface + signal types,
  then the polling implementation. Verification is "no new vue-tsc errors in touched
  files" against a 109-error baseline, plus lint.
- 2026-08-08 16:05 — ✅ `TBusEvent` declared in `plugins/bus/index.ts` as a typed event
  map (`toast` preserved, `live` added). This removed a real pre-existing type error;
  baseline drops 109 → 108.
- 2026-08-08 16:12 — Added `plugins/live/signals.ts`: `entity.changed` + `resync`,
  narrowing helpers, and `signalMatches()` for type-or-instance dependency matching
  (the latter is what task 024 needs so lists learn about ids they have never seen).
  Documented `event` as advisory-only, since coalescing makes the surviving name
  arbitrary.
- 2026-08-08 16:20 — Added `plugins/live/transport.ts`: `LiveTransport` interface
  (`name`, reactive `state`, `start(year)`, `stop()`) plus `createPollingTransport`.
  Decision: the polling implementation emits a single `resync` per tick rather than
  synthesising per-entity signals — so consumers need no polling-specific branch, and
  "revalidate everything" has exactly one code path shared with reconnect and
  server-side buffer overflow. It reports `polling`, never `live`.
  Decision: `year` is a `start()` parameter, not a module constant, because
  `EventSource` cannot send `X-YearSlug` and the eventual SSE URL needs it at
  subscribe time.
- 2026-08-08 16:26 — Added `plugins/live/index.ts`: app-wide singleton with
  `startLive`/`stopLive`/`liveState` and `setLiveTransport` so Phase 2 can swap in SSE
  (and tests can inject a fake) without any page changing. Pages are documented as
  never importing this — they compose the cache primitive instead.
- 2026-08-08 16:34 — ✅ Verified in the container: `vue-tsc` reports 108 errors total,
  **zero** in `plugins/live` or `plugins/bus`; eslint clean after dropping an unused
  type import. Note the `ui` image is still the pre-Node-20 one (`npm 8.19.1`), so a
  `docker compose build ui` is needed to exercise the new Dockerfile — unrelated to
  this task's correctness.
- 2026-08-08 16:36 — Completed. All criteria met. Transport seam in place with polling
  behind it; no backend work required. Next: 024 (cache primitive).
