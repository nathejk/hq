# 184 — Opt-in previous-years search

**Status:** done
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 183. Implements PRD 014's year rule end to end.

**Searching this year is the primary function. Previous years are a deliberate act by the
operator, never something the system slides into.** The server must not widen the year scope
on its own — not on an empty result, not on an exact phone match, not ever. An operator who has
not asked for history must be able to trust that what they are reading is this year.

Backend: honour `includePreviousYears` in the query layer and handler (the parameter and its
`false` default were wired in task 181).

Frontend: **a checkbox that is visibly off**, reading "søg også tidligere år" — not a year
dropdown pre-filled with the current year. A dropdown invites the operator to change the year
without noticing, and every list in HQ is already year-scoped by the global year selector.

Cross-year results state their year **on every row** and are grouped *after* the current
year's, never interleaved. A hit from 2024 presented alongside this year's is exactly the
stale-data-that-looks-current failure the live-updates design exists to avoid.

The control belongs to the query, so it belongs in the URL alongside `q` — a shared link must
reproduce what the sender saw.

## Acceptance Criteria

- [x] ~~`includePreviousYears`~~ **`includeOtherYears`** honoured through query layer and handler
      — delivered by task 181 under the corrected name; see log 2026-09-15 00:10
- [x] Test: the server never widens the scope on its own, including on an empty result — exists as
      `TestSearchPersonOnlyWidensTheYearOnAnExactOptIn` and `TestOnlyTheOptInWidensTheYear`
- [x] Checkbox off by default, labelled ~~"søg også tidligere år"~~ **"Søg også i andre år"** —
      wording changed deliberately; see log 2026-09-15 00:14
- [x] Cross-year rows labelled with their year and grouped after the current year's
- [x] The toggle is reflected in the URL
- [x] `go test ./...`, `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:06 — Picked up. Plan: frontend only — the two backend criteria were delivered by
  task 181, under the deliberately different name `includeOtherYears`. Add a checkbox, off by
  default, reflected in the URL alongside `q` and folded into the live-cache key (otherwise the two
  result sets collide on one entry). Extend `toRows` with the active year so cross-year rows form
  their own groups *after* every current-year group, and give them a visual treatment plus a Year
  column. First job: establish how the SPA can know the active year at all, since
  `globalstate.yearSlug` is deliberately empty for the current calendar year.
- 2026-09-15 00:10 — ✅ Both backend criteria were already met by task 181, so nothing in `go/` was
  touched (also outside this session's write scope). Verified rather than assumed: the parameter is
  read in `cmd/api/search.go` and threaded to `searchperson.Query.IncludeOtherYears`, and
  `search_test.go`'s `TestSearchPersonOnlyWidensTheYearOnAnExactOptIn` pins that only the exact
  string `true` widens — `1`, `yes`, `TRUE`, `true ` and the old name `includePreviousYears` all
  leave it false. `go test ./...` in the API container: 16 packages ok, no failures.
- 2026-09-15 00:12 — The criterion's parameter name is **struck**: it is `includeOtherYears`, not
  `includePreviousYears`, and the difference is the point rather than a typo. The dev stream carries
  fixture rows under year 9999, and nothing prevents a future edition existing before the current one
  closes, so "previous" would be false for some of the rows the flag returns.
- 2026-09-15 00:14 — Same reasoning applied to the label, which the criterion specified as "søg også
  tidligere år". Shipped as **"Søg også i andre år"**, with a sentence beside it stating what the
  current setting does ("Der søges kun i indeværende år" / "… i alle år — rækker fra andre år står
  nederst og er mærket med deres år"). A control that states its effect is what PRD §7 asks for, and
  "andre" is the only word that is true of every row it can return. Verified against real data that
  this matters: `includeOtherYears=true` on `q=Wedenborg` returns a 2025 row today.
- 2026-09-15 00:18 — Blocker: the SPA has no direct read of the active year.
  `globalstate.yearSlug` is a computed that returns **''** whenever the selected slug equals the
  current calendar year — which is precisely when the axios interceptor omits `X-YearSlug`
  altogether. Resolved by reconstructing the server's own fallback in `activeYearSlug()`:
  `yearSlug || String(new Date().getFullYear())`, which mirrors `app.YearSlug` in
  `cmd/api/routes.go` exactly. Documented at the function, since it is a real coupling to a Go
  default. Its only job is telling a cross-year row from a current-year one, so the cost of it being
  wrong is a mislabelled group heading, never a wrong search.
- 2026-09-15 00:22 — The flag is part of the live-cache key (`search:person:all:…` vs
  `search:person:year:…`), not only of the request. Sharing one entry would have shown the operator
  the narrow set immediately after they asked for the wide one. Pinned by a test that counts requests
  across a toggle.
- 2026-09-15 00:24 — The parameter is *omitted* rather than sent as `false` when off, so there is no
  shape in which a stray value could widen the scope — which also keeps the default URL clean, so
  "off" is never something an operator has to read a query string to confirm.
- 2026-09-15 00:28 — ✅ Cross-year grouping: `toRows` now takes the active year and places every
  cross-year row in a hard tier *after* every current-year row (`OTHER_YEAR_TIER`), grouped by year
  with the year leading the heading ("2019 · Spejdere", most recent year first). A test asserts that
  no current-year row can follow a cross-year one, which is the property that matters rather than the
  particular order. The year is also its own column — shown only when the flag is on, since a column
  of identical values is noise — so the year is on every row and not only in the heading.
- 2026-09-15 00:31 — Cross-year rows are tinted and muted (`.search-other-year`), not merely
  labelled: the label is what an operator *confirms* with, the tint is what stops them reading the
  row as current in the first place. Tested on the class, not on the colour.
- 2026-09-15 00:33 — Toggling is deliberately **not** debounced — it is one considered act on a
  settled query, so 250 ms of lag would be all cost. It commits the current text along with it, so a
  toggle mid-typing cannot search the previous query under the new scope.
- 2026-09-15 00:35 — The "ingen match" state now names the scope it searched, and when narrow it
  points at the checkbox. That is the moment an operator is deciding whether somebody was ever
  involved at all, so it is the moment the year scope needs stating.
- 2026-09-15 00:38 — ✅ `go test ./...` (16 ok), `npm run test:unit` 331 passed (21 files),
  `npm run build` clean.
- 2026-09-15 00:40 — Completed. Two criteria struck on naming/wording grounds, both because the
  word "previous" is not true of the rows the flag returns; the behaviour behind them is delivered.
