# 182 — SearchView.vue and the /search route

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Depends on task 181. `vue/src/views/SearchView.vue`, lazy-loaded at `/search?q=…` in
`vue/src/router/`.

The query lives in the URL so a search is linkable and survives reload.

**Live, not fetch-on-mount.** Load through
`useLiveResource('search:' + query, fetcher, { dependsOn })`. `dependsOn` names entity
**types** — a new match is a row whose id was never seen — and the tokens are the *event
subject's* entity, not the projection's name. Take the list verbatim from task 178's progress
log; do not invent it. There is no `personnel` token. A wrong token fails silently: the page
looks live and never updates. The SPA warns in the dev console, so check it.

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
- [ ] Data via `useLiveResource` with `dependsOn` from task 178 — no `onMounted` + `http.get`
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
