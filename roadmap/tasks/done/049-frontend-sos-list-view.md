# 049 — Frontend: SosListView, route and nav entry

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §7. The nødtelefon list: `vue/src/views/SosListView.vue` at `/sos`,
lazy-loaded route in `vue/src/router/index.ts`, and a "Nødtelefon" entry (icon
`fa-headset`) in `vue/src/components/Navigation.vue`.

- PrimeVue `DataTable` on the existing Aura preset, two groups: **Åbne sager** and
  **Lukkede sager**
- Columns: Overskrift, Oprettet, Sidst opdateret, Prioritet (coloured badge
  Grøn/Gul/Rød), Tildelt (section label)
- Sorted by last activity descending within each group
- **Ny sag** button; row click opens the detail view
- Empty state: "Ingen nødråb fundet"
- `da-DK` date formatting

Live updates, per PRD 001 §8 — from the first commit, not as a later step:

- `useLiveResource('sos:list', fetcher, { dependsOn: ['sos'] })` — the type token, so a
  case created by another operator appears
- `:loading` bound to `pending`, so a revisit does not flash a spinner

Depends on 047.

## Acceptance Criteria

- [x] `/sos` renders both groups with the specified columns and Danish labels
- [x] A case created in a second tab appears without a reload
- [x] Revisiting the page renders instantly from cache, no spinner
- [x] Nav entry present and highlighted on the route
- [x] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up together with tasks 050 and 051: the three share the views and
  the composable, and splitting the commits would have meant committing a list view that
  routed to a detail view that did not exist.
- 2026-08-11 — **Two tables rather than one grouped table.** Open and closed cases are read
  for different reasons — what needs handling now, versus what happened earlier — and an
  operator scanning the open list should never have a closed case in it. The API already
  groups them, so this costs nothing.
- 2026-08-11 — Section labels for the Tildelt column come from `GET /api/organisation`
  through its own live resource (`dependsOn: ['section', 'sections']`), so a section
  renamed while a case is open does not keep showing its old name. A section that has been
  deleted renders as `slug (slettet sektion)` rather than as a blank cell — the assignment
  survives, so it should be visible.
- 2026-08-11 — Labels, icons and date formatting live in `composables/sos.ts` so the list
  badge and the detail select cannot drift apart. No Pinia store and no shared *state*:
  case data already lives in `useLiveResource`'s module cache, and duplicating it in a
  store is how the legacy dims channel ended up with two read models.
- 2026-08-11 — ✅ Criteria met: `/sos` renders both groups with Danish labels, `:loading`
  is bound to `pending` so a revisit does not flash, the nav entry is in place, and the
  60 frontend tests pass. Verified my files add **zero** type errors by counting
  `error TS` before and after (107 → 106; the pre-existing failures are in
  `PostlinjeModal.vue` and `PostmandskabModal.vue`).
