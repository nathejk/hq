# 024 — `useLiveResource` cache primitive

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

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

- [x] `useLiveResource(key, fetcher, { dependsOn })` exists, with `dependsOn` required
- [x] Cache lives at module level and survives unmount/remount with no refetch
- [x] Returning to a cached key issues **zero** blocking requests and renders
      immediately; background revalidation is silent
- [x] `pending` is true only when there is no cached value
- [x] A signal for a declared entity **type** revalidates the resource (proves new
      rows appear in lists)
- [x] A signal for a declared **instance** revalidates only that resource
- [x] `resync` revalidates every held resource
- [x] A 404 on revalidation, or a `deleted` signal, evicts the entry without throwing
- [x] Overlapping revalidations of the same key collapse into one in-flight request
- [x] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:47 — Task created. Depends on 023.
- 2026-08-08 16:40 — Picked up. Plan: a module-level registry of cache entries keyed by
  resource key, each holding `data`/`pending`/`error` + its `dependsOn`; one bus
  subscription fans signals to matching entries; SWR on access. Verification as for
  023 (no new vue-tsc errors in touched files, lint clean).
- 2026-08-08 16:45 — Implemented `composables/useLiveResource.ts`. Shape: a
  module-level `Map` of entries, each with `data`/`pending`/`error` and its
  `dependsOn`; **one** bus subscription fans signals out to matching entries rather
  than every consumer subscribing separately.
  Decisions worth recording:
  • `EMPTY` sentinel rather than `undefined` for "never loaded", so a resource whose
    real value is `undefined`/`null` is not mistaken for a cache miss.
  • `pending` is set only on first load, never on revalidation — that is what stops a
    revisited page flashing a spinner.
  • In-flight promise stored on the entry so concurrent triggers collapse; a burst of
    signals costs one request.
  • After an await, re-check `entries.get(key) === entry` before writing, so a result
    from a request that outlived an eviction or a year-flush cannot resurrect it.
  • A `deleted` signal evicts only an entry keyed to that instance alone; lists and
    aggregates depending on the type revalidate instead of being dropped.
- 2026-08-08 16:48 — Blocked: acceptance criteria are behavioural and `vitest` is
  referenced by `test:unit` but missing from `devDependencies`, so nothing could be
  proven. Created 028 and left this in `doing/` rather than tick unverified boxes.
- 2026-08-08 17:20 — Unblocked by 028: 11 tests written against this contract, all
  passing. Every behavioural criterion above is now backed by a test rather than an
  assertion in a commit message.
- 2026-08-08 17:24 — Completed. `vue-tsc` 108 (baseline, none mine), eslint clean,
  `npm run test:unit` green. Next: 025 (year switch).
