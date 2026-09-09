# 173 — mark low-confidence positions on the map

**Status:** open
**Priority:** low
**Created:** 2026-09-09
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §11 (residual UI question) and PRD 013 §11.

The producer already drops what is not a position at all — NaN, Null Island, impossible clocks,
accuracy worse than 100 km — and deliberately keeps poor-but-real fixes, because a bad position
is still the only evidence of where somebody was. HQ stores `accuracy` as sent and filters
nothing.

So a multi-kilometre cell-tower fix currently renders exactly like a 10 m GPS fix: as a point.
That slightly overstates what HQ knows, and it does so in the two places an operator acts on:

- the **position layer** on `/kort` (PRD 013) — a dot whose true position could be anywhere in a
  2 km circle;
- the **track dialog** (task 150) — a vertex that makes a route look more certain than it is.

Options, cheapest first: a hollow or thinner vertex above some accuracy threshold; an accuracy
circle drawn on click; a note in the popup ("±2 km"). The popup line is nearly free and may be
enough — the question is whether operators need to *see* uncertainty at a glance or only to be
able to check it.

## Acceptance Criteria

- [ ] Decision on how uncertainty is shown (glyph, circle on click, popup text)
- [ ] Implemented in the position layer and/or the track dialog, per that decision
- [ ] Threshold — what counts as low confidence — chosen and recorded with its reasoning
- [ ] Recorded in PRD 011 §11 and PRD 013 §11

## Progress Log

- 2026-09-09 — Created while closing PRD 011 and writing PRD 013, so the residual survives them.
