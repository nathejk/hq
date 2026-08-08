# 024 — `useLiveResource` cache primitive

**Status:** doing
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:**

## Description

The piece that actually delivers the perceived speed in PRD 004: a single cache
primitive every page composes, so no view implements its own fetching, caching or
loading state. Depends on 023 (transport + bus signals).

`useLiveResource(key, fetcher, { dependsOn })` owns one cache entry, subscribes to
signals, and exposes explicit state. Module-level storage means the cache survives
route changes — which is what makes navigating back instant, with **zero** requests.

### Settled decisions this must implement

- **`dependsOn` is mandatory, even when empty.** Dependencies may be entity *types*
  (`['sos']`) or specific *instances* (`['sos:123', 'sos_activity']`). A signal
  invalidates every resource declaring a dependency on that type or instance.
  Rationale: a signal names one instance, but lists (a *new* row has an id the client
  has never seen) and derived aggregates (an in-our-care count, a team strength) do
  not map to `entity:id` at all. Without declarations, new rows never appear and
  computed figures never move — silently, with no error anywhere. Making the argument
  mandatory forces the decision into review.
- **Stale-while-revalidate:** render cache immediately, revalidate in the background.
  A spinner appears **only** when there is no cached value at all.
- **Explicit state:** `data` / `pending` / `error` modelled once, here. This is the
  direct fix for legacy trap §2.1.6, where `lastModify[view]` timestamps doubled as
  "have we loaded?" and every getter re-implemented the check, making loading, empty
  and missing indistinguishable.
- **Keyed map + derived list**, never arrays-used-as-maps recomputed per message
  (legacy trap §2.1.7).
- **Eviction:** a `deleted` signal, or a revalidation that 404s, evicts the entry
  rather than surfacing an error.
- **`resync`** revalidates everything currently held — the same path a reconnect
  uses, which is why transport overflow can degrade into it safely.

### Notes

- No page may touch the transport or the bus directly; they compose this.
- Concurrent revalidations of one key must not stampede — collapse them.
- Verification: no new `vue-tsc` errors in touched files (109 pre-existing baseline),
  lint clean. There is no test runner wired up (`test:unit` references `vitest`, which
  is **not** in `devDependencies`) — see 028.

## Acceptance Criteria

- [ ] `useLiveResource(key, fetcher, { dependsOn })` exists, with `dependsOn` required
- [ ] Cache lives at module level and survives unmount/remount with no refetch
- [ ] Returning to a cached key issues **zero** blocking requests and renders
      immediately; background revalidation is silent
- [ ] `pending` is true only when there is no cached value
- [ ] A signal for a declared entity **type** revalidates the resource (proves new
      rows appear in lists)
- [ ] A signal for a declared **instance** revalidates only that resource
- [ ] `resync` revalidates every held resource
- [ ] A 404 on revalidation, or a `deleted` signal, evicts the entry without throwing
- [ ] Overlapping revalidations of the same key collapse into one in-flight request
- [ ] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:47 — Task created. Depends on 023.
- 2026-08-08 16:40 — Picked up. Plan: a module-level registry of cache entries keyed by
  resource key, each holding `data`/`pending`/`error` + its `dependsOn`; one bus
  subscription fans signals to matching entries; SWR on access. Verification as for
  023 (no new vue-tsc errors in touched files, lint clean).
