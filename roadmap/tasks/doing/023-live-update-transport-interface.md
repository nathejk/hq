# 023 — Live-update transport interface + polling implementation

**Status:** doing
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:**

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

- [ ] `TBusEvent` declared, typed for the live-update signals, with the existing
      `toast` event preserved — `plugins/bus/index.ts` no longer errors
- [ ] A transport interface exists with `start()` / `stop()` and publishes signals
      onto the mitt bus, not to callers directly
- [ ] A polling implementation of that interface, with a configurable interval,
      emitting `resync` per tick
- [ ] Signal types are exported and documented: `entity.changed` and `resync`
- [ ] Starting twice is safe (idempotent), and `stop()` fully tears down timers
- [ ] No new `vue-tsc` errors in the files touched (baseline is 109 pre-existing)
- [ ] `npm run lint` clean for the new files

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:45 — Task created. Split from PRD 004 Phase 1; first of 023–027.
- 2026-08-08 15:55 — Picked up. Plan: declare the bus event map first (it blocks using
  the bus at all), then a `transport` module exporting the interface + signal types,
  then the polling implementation. Verification is "no new vue-tsc errors in touched
  files" against a 109-error baseline, plus lint.
