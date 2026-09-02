# 132 — Per-set extent overlay with the gaps shaded

**Status:** open
**Priority:** low
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §7 (seam check), §8 ("Coverage: an overlay, not a percentage"). Depends on 130.
**Sequenced last and cuttable** — see the risk below.

A per-set toggle draws every extent in the set at once and shades the space between them, so
a seam — a strip of ground no sheet shows — is immediately visible.

This deliberately replaces an earlier design that computed a coverage *percentage*
server-side. There is no denominator to compute one against: the race area has no recorded
boundary, and all maps lie inside it by construction. Worse, a percentage measures the wrong
thing — sheets overlap by design, so the failure that matters is a seam between two sheets,
which barely moves a percentage while losing a patrol.

**Known risk:** shading the *complement* of a union of rectangles is real geometry and
Leaflet will not do it. Options are a polygon-clipping dependency or a canvas/grid
approximation. If it proves expensive, **fall back to drawing the extents as outlines only**
— a human already spots a seam from that, and it delivers the goal at a fraction of the
cost. Do not add a heavy dependency for the shading without saying so.

## Acceptance Criteria

- [ ] Per-set toggle drawing all extents in the set together
- [ ] Gaps between them visibly distinguished (shaded, or outlines-only fallback)
- [ ] No coverage percentage and no coverage endpoint introduced
- [ ] A deliberate seam between two test maps is visible
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
