# 182 — SearchView.vue and the /search route

**Status:** doing
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

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

- [ ] `/search?q=` route, lazy-loaded, query in the URL and reload-safe
- [ ] Data via `useLiveResource` with the eight `dependsOn` tokens above — no `onMounted` + `http.get`
- [ ] No live-dependency warnings in the dev console
- [ ] Debounce applied before the cache key changes
- [ ] Minimum length enforced client-side
- [ ] Results grouped by kind with Danish headings; whole row links out
- [ ] Matched phone annotated when it is a parent's or a contact's
- [ ] Three distinct empty states
- [ ] `pending` wired to `:loading`, no extra spinner
- [ ] `npm run build` and unit tests pass

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
