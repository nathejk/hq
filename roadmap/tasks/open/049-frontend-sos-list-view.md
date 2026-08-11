# 049 — Frontend: SosListView, route and nav entry

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `/sos` renders both groups with the specified columns and Danish labels
- [ ] A case created in a second tab appears without a reload
- [ ] Revisiting the page renders instantly from cache, no spinner
- [ ] Nav entry present and highlighted on the route
- [ ] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
