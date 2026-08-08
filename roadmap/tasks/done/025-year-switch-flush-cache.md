# 025 — Year switching flushes the cache and refetches

**Status:** done
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

PRD 004 settled that the selected event year is passed **when subscribing** (a query
parameter, because `EventSource` cannot set headers) and that switching year must
flush everything. Depends on 023 and 024.

Two failure modes this exists to prevent:

1. **Stale cross-year data.** The cache is keyed by resource, not by year, so without
   an explicit flush a 2025 case list would linger while the operator believes they
   are looking at 2026. Clearing the whole cache on a year change is cheap and
   unambiguous; a brief spinner is much better than plausible wrong data.
2. **Silent divergence between stream and fetches.** REST calls carry the year via the
   `X-YearSlug` axios interceptor; the stream carries it as a query parameter. If the
   two can drift apart, the client receives signals for one year while fetching
   another and simply appears frozen — no error, nothing in a log. Both must derive
   from **one** source of truth: `globalstate.yearSlug`.

A missing year on the server defaults to the current calendar year (existing
`YearSlug()` behaviour, `go/cmd/api/routes.go:102`) — a sane default. The SPA sends it
explicitly regardless, so it never relies on that.

### Notes

- Year changes are rare, so tearing down and re-establishing the transport is fine.
- `globalstate.yearSlug` is currently a `computed` that returns `''` when the selected
  year equals the current calendar year, with `setYearSlug` writing `rawYearSlug`.
  Note it also has two pre-existing type errors — do not silently paper over them;
  either leave them or fix them deliberately and say so in the log.

## Acceptance Criteria

- [x] Changing the selected year clears **every** cache entry
- [x] The transport is stopped and restarted with the new year
- [x] All mounted resources refetch for the new year
- [x] No cache entry from the previous year is observable afterwards
- [x] The year used by the transport and the year sent by `http` provably come from
      the same source
- [x] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:49 — Task created. Depends on 023, 024.
- 2026-08-08 17:26 — Picked up. Plan: a `useLiveYear` composable that watches the one
  source of truth (`globalstate`), and on change stops the transport, clears the cache,
  restarts for the new year and refetches. Tests assert no pre-change entry survives.
- 2026-08-08 17:32 — **Found a flaw in my own 024 implementation.** `clearLiveCache()`
  drops entries from the map, but a mounted component captured its entry object at
  setup — so a dropped entry leaves that view displaying old-year data forever, and no
  signal can reach it because it is no longer in the map. Clearing was the obvious
  implementation and the wrong one.
  Fix: added `flushLiveCache()`, which resets entries **in place** (data → empty, error
  cleared) so mounted views transition to loading and then to new-year data.
  `clearLiveCache()` remains for teardown/tests, now documented as such.
- 2026-08-08 17:36 — Second correctness hole from the same area: a slow request issued
  for the *old* year could resolve after the switch and overwrite new-year data — the
  precise "stale data that looks live" failure this design exists to prevent. Added a
  generation counter per entry, bumped on flush; a response whose generation has passed
  is discarded. Covered by a test that resolves an old request after the switch.
- 2026-08-08 17:40 — Added `composables/useLiveYear.ts` (`installLiveYearSync`) and
  wired it into `main.ts`. It watches `globalstate.yearSlug` — the same value the axios
  interceptor sends as `X-YearSlug` — so stream and fetches cannot diverge. Empty means
  "current calendar year", matching the interceptor and the server's `YearSlug()`
  default.
- 2026-08-08 17:44 — Test failure exposed a leak in my own test seam: the watcher is
  created outside any component scope (deliberately, it must outlive every view), and
  `resetLiveYearSyncForTests` only flipped a flag — so watchers accumulated across
  tests and all fired on the next year change. Now the stop handle is kept and actually
  called. Worth noting this would also have leaked on hot reload.
- 2026-08-08 17:47 — Deliberately fixed one pre-existing type error in `globalstate.ts`
  while here: `new Date().getFullYear() != rawYearSlug.value` compared a number with a
  string (TS2367). Now `String(...) !== ...`, which is behaviourally identical (the
  loose comparison coerced anyway) but type-correct — and I depend on this value being
  right.
  **Left alone:** `setNav` assigns to a `computed` (TS2540) and is dead code (not
  exported, and flagged unused by eslint). It is a real bug but not this ticket's, and
  removing a function someone may intend to use is not my call.
- 2026-08-08 17:50 — ✅ 16 tests pass (5 new). `vue-tsc` down to 107 from the 109
  baseline; the only error in a file I touched is the pre-existing `setNav` one.
- 2026-08-08 17:52 — Completed. Next: 026 (connection state indicator).
