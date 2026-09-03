# 132 — Per-set extent overlay with the gaps shaded

**Status:** done
**Priority:** low
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Per-set toggle drawing all extents in the set together
- [x] Gaps between them visibly distinguished — **shaded, exactly**, not the outlines-only fallback
- [x] No coverage percentage and no coverage endpoint introduced
- [x] A deliberate seam between two test maps is visible
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Before reaching for the outlines-only fallback: the rectangles are all
  **axis-aligned**, which makes the complement exactly computable by grid decomposition — cut on
  every distinct latitude and longitude, then keep the cells no rectangle covers. No clipping
  library, and it is a pure function, so it can be tested rather than eyeballed.
- 2026-09-03 — **The fallback was not needed.** The risk noted in this task — that shading the
  complement of a union means polygon clipping — does not apply to axis-aligned rectangles: cut on
  every distinct latitude and longitude, and each grid cell is *wholly* covered or *wholly*
  uncovered, so the uncovered cells are the complement exactly. `gapCells` is ~30 lines, has no
  dependency, and is exact rather than sampled — which matters, because a **thin** seam is the
  dangerous one and a fixed sampling grid is precisely what would miss it. Tested at 0.0001°.
- 2026-09-03 — The cell midpoint decides coverage, which is sound only because every rectangle edge
  became a grid line: no edge can fall strictly inside a cell, so the midpoint's answer is the whole
  cell's answer. Worth writing down, since it looks like sampling and is not.
- 2026-09-03 — Gap cells are **merged along each row**, so two sheets side by side yield one red band
  rather than a mosaic of squares an operator has to interpret as a band.
- 2026-09-03 — Decided the **L-shaped case reports its uncovered corner**, and tested it. Strictly that
  corner is not a seam *between* sheets, but it is genuinely uncovered ground inside the area the set
  spans, and showing it lets the operator decide. The alternative would be guessing which uncovered
  ground is intentional — and the one time the guess is wrong is the time it matters.
- 2026-09-03 — Touching and overlapping sheets report nothing. Non-negotiable: overlap is designed in,
  and a check that cried wolf on the normal case would be ignored within a day.
- 2026-09-03 — On the map, the set's own areas are drawn as **dashed outlines with no fill**, and only
  the gaps are filled. With a dozen translucent fills stacked, overlaps read as darker patches and an
  operator starts seeing "coverage" in what is only paint. The gaps are the answer, so the gaps are
  the only thing shaded.
- 2026-09-03 — The summary also counts sheets with **no** area ("tæller ikke med (fx skitser)"). Without
  it, a set of five sheets where three have no rectangle would report a clean bill of health from two
  — technically true, dangerously misleading.
- 2026-09-03 — No percentage and no endpoint, per PRD §8: there is no denominator (the race area has no
  recorded boundary, and every sheet lies inside it by construction), and a percentage would read
  ~100% while a seam went unnoticed.
- 2026-09-03 — Verified end to end: gave the two live sheets a deliberate 0.1° gap through the API, then
  ran `gapCells` over the **exact payload the API returned** — including its ordering, which is
  reversed relative to how the sheets were drawn — and got the seam rectangle 9.4→9.5 back.
- 2026-09-03 — Left those extents in the dev data, so the toggle has something to show; the two patrol
  sheets deliberately leave a seam.
- 2026-09-03 — Completed. 38 kort tests pass, `vite build` clean, no type errors in the feature. The
  rendering itself is unverified in a browser.
