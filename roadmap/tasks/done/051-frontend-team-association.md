# 051 — Frontend: associate patrols with a case

**Status:** done
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

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

- [x] Searching by team number finds a patrol and associating it updates the card and the
      timeline
- [x] Associating an already-associated patrol is harmless (idempotent)
- [x] Disassociating removes the row and records a timeline entry
- [x] Contact phone is click-to-call on mobile (`tel:` link)
- [x] No member rows rendered
- [x] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up with tasks 049 and 050. Built as `components/SosTeamCard.vue` so
  PRD 006 can extend one component rather than a slice of a large view.
- 2026-08-11 — The picker reuses the **same cache key** as `PatruljeListView`
  (`patrulje:list`), not a second fetch of the same endpoint. Opening a case therefore
  costs no extra request, and the patrol list cannot be stale in one place and fresh in
  the other.
- 2026-08-11 — Search requires two characters and shows at most ten matches: this is a
  mid-call lookup by the number the caller reads out, not a browse. Already-associated
  patrols are filtered out of the candidates, so the same patrol cannot be added twice
  from the UI (the API is idempotent regardless).
- 2026-08-11 — Contact phone is a `tel:` link — an operator on a phone taps to call rather
  than copying digits.
- 2026-08-11 — **No member rows**, per PRD 001's decision: they arrive in PRD 006 with the
  status and actions that give them a purpose. The card is laid out with room for them
  rather than around their absence.
- 2026-08-11 — ✅ Criteria met. 60 frontend tests pass; no new type errors.
