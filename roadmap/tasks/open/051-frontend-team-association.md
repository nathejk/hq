# 051 — Frontend: associate patrols with a case

**Status:** open
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §7. The **Tilknyttede patruljer** card in `SosView.vue`.

- A picker **searchable by team number, name and group** — filtering the year's patrol
  list already in the SPA's live cache (`GET /api/patrulje`, as `PatruljeListView` uses).
  No new search endpoint. Operators search by number, because that is what a caller reads
  out
- One row per associated patrol: number, name, group, korps, contact phone, a link to the
  patrol's page, and a remove (disassociate) action
- **No member rows.** PRD 001 deliberately ships without them; PRD 006 introduces them
  together with member status and actions. Leave vertical room for that
- Only patrols — klaner cannot be associated

Depends on 047 and 050.

## Acceptance Criteria

- [ ] Searching by team number finds a patrol and associating it updates the card and the
      timeline
- [ ] Associating an already-associated patrol is harmless (idempotent)
- [ ] Disassociating removes the row and records a timeline entry
- [ ] Contact phone is click-to-call on mobile (`tel:` link)
- [ ] No member rows rendered
- [ ] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
