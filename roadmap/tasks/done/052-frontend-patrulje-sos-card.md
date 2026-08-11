# 052 — Frontend: "Kontakt med nødtelefon" card on the patrol page

**Status:** done
**Priority:** low
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §7. Add a "Kontakt med nødtelefon" card to
`vue/src/views/PatruljeView.vue` listing the patrol's SOS cases — created date, headline,
open/closed badge — with a click through to the case.

Render-only: the data arrives in the existing `GET /api/patrulje/:id` payload (task 048),
so the only wiring is adding `'sos'` to that view's existing `dependsOn`
(`['patrulje:{id}', 'spejder', 'order', 'payment']`). No second request, no new resource.

Hide the card, or show a quiet empty state, when the patrol has no cases — most patrols
never call.

Depends on 048.

## Acceptance Criteria

- [x] Card lists the patrol's cases, newest first, and navigates on click
- [x] `'sos'` added to the view's `dependsOn`, so a new case for this patrol appears
      without a reload
- [x] A patrol with no cases does not show a broken or noisy empty card
- [x] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Render-only, as planned: the cases arrive in the existing
  `GET /api/patrulje/:id` payload (task 048), so the change is a table plus `'sos'` on
  the view's existing `dependsOn`. Type-level token, because a case opened for this
  patrol has an id this client has never seen.
- 2026-08-11 — The card is **hidden entirely** when the patrol has no cases, rather than
  showing an empty state. Most patruljer never call the nødtelefon, and an empty card on
  every patrol page trains operators to ignore the exact place a real incident would
  appear.
- 2026-08-11 — ✅ Criteria met: 60 frontend tests pass, no new type errors, Vite compiled
  the view cleanly.
