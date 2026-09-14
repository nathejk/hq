# 182 — SearchView.vue and the /search route

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 181. `vue/src/views/SearchView.vue`, lazy-loaded at `/search?q=…` in
`vue/src/router/`.

The query lives in the URL so a search is linkable and survives reload.

**Live, not fetch-on-mount.** Load through
`useLiveResource('search:' + query, fetcher, { dependsOn })`. `dependsOn` names entity
**types** — a new match is a row whose id was never seen — and the tokens are the *event
subject's* entity, not the projection's name. **Task 178 verified the set; use it exactly:**

```
crew, crewmember, friend, gøgler, klan, patrulje, senior, spejder
```

There is no `personnel` token (that is the table's name) and no `bandit` (a bandit is a senior
with an arm number). Both mistakes are documented in `go/internal/live/entities.go` because
both have been made before, and both fail *silently*: the page looks live and never updates.
`searchperson/entities_test.go` pins the set, so if a source is ever added, that test and this
list change together. The SPA also warns in the dev console — check it.

**Debounce before the key changes, not after.** Each distinct key is a module-level cache
entry that survives route changes, so feeding raw keystrokes into the key would leave an entry
per prefix of everything ever typed. Debounce, then set the key.

**Minimum query length** (3 characters / 4 digits) enforced client-side, so the first
keystroke does not ask the server for every person in the event.

Rows are grouped by `kind` with Danish headings (Spejdere, Seniorer, Gøglere, Crew,
Kontaktpersoner). Each row shows name, matched phone, team/section and status, and the whole
row is the link to the page that holds the person. The matched phone is annotated when it is
not the person's own — "forælders nummer", "kontaktperson" — because an operator about to
speak to someone must not be misled about who will answer.

**Three empty states, not two:** nothing typed, too short to search, and searched-with-no-match.
An empty box and an empty result must not look alike.

Wire `pending` to the list's `:loading` and add no separate spinner — `pending` is true only
when nothing is cached, so a repeated query must not flash. The view is read-only, so it needs
none of the dirty-state deferral that `KlanListView` and `KortView` carry.

Status badges are task 183; the previous-years control is task 184.

## Acceptance Criteria

- [x] `/search?q=` route, lazy-loaded, query in the URL and reload-safe
- [x] Data via `useLiveResource` with the eight `dependsOn` tokens above — no `onMounted` + `http.get`
- [x] No live-dependency warnings in the dev console
- [x] Debounce applied before the cache key changes
- [x] Minimum length enforced client-side
- [x] Results grouped by kind with Danish headings; whole row links out
- [x] Matched phone annotated when it is a parent's or a contact's
- [x] Three distinct empty states
- [x] `pending` wired to `:loading`, no extra spinner
- [x] `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-14 22:20 — Picked up. Plan: (1) a pure `composables/personSearch.ts` holding the
  response types, the client-side minimum-length rule, the kind→Danish-group mapping, the
  phone-role annotations, the link targets and the flattening/ordering of rows, with a spec —
  so the decisions that matter are testable without mounting anything; (2) `views/SearchView.vue`
  consuming it, with the debounced value written to the URL so the URL *is* the cache key
  (debounce lands before the key changes, and the search stays linkable/reload-safe in one
  mechanism rather than two); (3) lazy route `/search`. Re-keying `useLiveResource` as the query
  changes will use a detached `effectScope` per key, disposed on change, because the composable
  takes a static key by design.
- 2026-09-14 22:35 — Wrote `composables/personSearch.ts`: response types, `queryDigits`/`isSearchable`
  (the server's 4-digit / 3-letter rule mirrored client-side), `emptyState`, the kind→heading map,
  `phoneRoleLabel`, `displayName`/`teamLabel`, `resultRoute`, and `toRows`.
- 2026-09-14 22:38 — Decision: `friend` gets its own heading ("Friends") rather than being folded
  into "Gøglere". PRD §7 lists five headings and this makes six, but gøgler and friend are separate
  personnel kinds in hq and folding them would discard the one word on the row that says which —
  and the operator's next action differs. Noted for the PRD.
- 2026-09-14 22:40 — Decision on link targets: only `patrulje` has a detail route, so
  spejder/patruljekontakt → `/patrulje/:teamId`, senior/klankontakt → `/klan`, gøgler/friend →
  `/badut`, crew → `/organisation`. A row with no team returns `null` and is rendered as text, not a
  dead link. Verified against real data that `patruljekontakt.teamId` is the patrol's id, so the
  contact row lands on the same page as its scouts.
- 2026-09-14 22:44 — Wrote `SearchView.vue`. The debounced value is written to the URL with
  `router.replace`, and the URL is what the cache key is derived from — one mechanism serving
  linkability, reload-safety and the debounce-before-key requirement, rather than a separate
  debounced ref shadowing the URL. `replace` not `push`, so Back leaves the page instead of walking
  backwards through the operator's typing; Enter commits immediately rather than waiting out the
  250 ms.
- 2026-09-14 22:47 — One `DataTable` with `rowGroupMode="subheader"` rather than a table per group:
  it gives `pending` a single place to land (the criterion) and keeps the whole result set in one
  keyboard tab order. Grouping and heading order are driven by the `order` field `toRows` computes,
  which also leaves room for task 184 to order cross-year groups after current-year ones.
- 2026-09-14 22:52 — Blocker found and fixed before it shipped: `useLiveResource` is keyed once by
  design, so the resource is recreated in a detached `effectScope` per key. The first cut held it in
  a `ref`, which reactive-proxies the returned object and *unwraps* the refs inside it — so
  `pending.value` and `data.value` were `undefined`, i.e. a loading state that silently never
  appears. Changed to `shallowRef` and pinned it with a component test, since no logic test could
  see it.
- 2026-09-14 22:56 — ✅ Criteria: route lazy-loaded with the query in the URL; data through
  `useLiveResource` (no `onMounted` + `http.get`); the eight tokens taken from a single exported
  `SEARCH_DEPENDS_ON` constant and asserted against the Go set in `personSearch.spec.ts`;
  minimum length enforced before the key is set, so a short query issues no request at all.
- 2026-09-14 22:58 — Checked the advertised token set directly rather than trusting the list:
  `curl -N /api/stream` inside the API container returns an `entities` frame containing all eight
  (`crew, crewmember, friend, gøgler, klan, patrulje, senior, spejder`), so the SPA's dev-console
  validator has nothing to warn about. ✅ no live-dependency warnings.
- 2026-09-14 23:04 — Added `views/SearchView.spec.ts` (jsdom): counts requests across four
  minimum-passing keystrokes to prove one request for the settled query rather than one per prefix;
  asserts the three empty states render distinct wording; asserts the `:loading` prop rather than a
  PrimeVue overlay class, plus a revisit-from-cache case proving `pending` stays false so the page
  does not flash.
- 2026-09-14 23:10 — Caught myself over-ticking "the whole row is the link": the first cut anchored
  only the name. Implemented `@row-click` on top of the anchor — the anchor stays because a row
  handler is invisible to the keyboard and to open-in-new-tab — and a row with no route now does
  nothing rather than navigating somewhere approximate. Two tests cover both.
- 2026-09-14 23:14 — ✅ Remaining criteria: Danish headings with the whole row linking out; phone
  annotated only when it is a parent's or a contact's (`own` and `''` are deliberately unannotated,
  with `''` rendering "intet nummer" so a nameless-by-phone row does not look broken); `pending`
  wired to `:loading` with no spinner of our own.
- 2026-09-14 23:16 — ✅ `npm run test:unit` 306 passed (21 files), `npm run build` clean, and
  `vue-tsc` reports no errors in the two new files (the repo has pre-existing errors elsewhere,
  which is why `build` does not typecheck).
- 2026-09-14 23:18 — Completed. `/search?q=…` is live, cached, debounced-before-key and reachable
  only by URL for now — the navigation entry point is task 185, deliberately last. Status is plain
  text pending task 183.
