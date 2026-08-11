# 052 — Frontend: "Kontakt med nødtelefon" card on the patrol page

**Status:** open
**Priority:** low
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Card lists the patrol's cases, newest first, and navigates on click
- [ ] `'sos'` added to the view's `dependsOn`, so a new case for this patrol appears
      without a reload
- [ ] A patrol with no cases does not show a broken or noisy empty card
- [ ] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
