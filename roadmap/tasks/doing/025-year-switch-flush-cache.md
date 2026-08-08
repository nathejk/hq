# 025 — Year switching flushes the cache and refetches

**Status:** doing
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:**

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

- [ ] Changing the selected year clears **every** cache entry
- [ ] The transport is stopped and restarted with the new year
- [ ] All mounted resources refetch for the new year
- [ ] No cache entry from the previous year is observable afterwards
- [ ] The year used by the transport and the year sent by `http` provably come from
      the same source
- [ ] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:49 — Task created. Depends on 023, 024.
- 2026-08-08 17:26 — Picked up. Plan: a `useLiveYear` composable that watches the one
  source of truth (`globalstate`), and on change stops the transport, clears the cache,
  restarts for the new year and refetches. Tests assert no pre-change entry survives.
