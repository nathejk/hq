# 134 — Document `GET /api/kort` for the hej-app team

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §8, §9. Depends on 125.

The hej-app lives in another repo and is the reason this feature exists. Hand its team a
short written contract for `GET /api/kort`, covering the things they cannot infer from the
JSON and would otherwise get wrong:

- **Two reveal units, not one.** A map's checkpoints are revealed when its QR is linked or
  scanned; a checkgroup is revealed as a whole when any of its checkpoints is scanned. A
  skitse has no QR — its checkpoints are revealed via the previous checkpoint.
- **Find patrol maps via the set's `teamType`, never by name.** Set names are Danish free
  text an organizer may rename mid-season.
- **`teamType` is a filter, not a key.** Several sets may share one. It is nullable, and an
  unmarked set is the general/crew set — which klaner draw from. So filtering by `klan`
  usually returns nothing, and that is **not** an error: fall back to the unmarked set.
- Year-scoped by `X-YearSlug`; arrays are `[]`, never `null`.
- Maps are per year and never carried forward — the event is in a different area each year.

## Acceptance Criteria

- [ ] Written contract committed in the repo and shared with the hej-app team
- [ ] Covers both reveal units, the `teamType` semantics and the klan fallback
- [ ] Sample response payload included
- [ ] OpenAPI annotations verified to match the documented shape

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
