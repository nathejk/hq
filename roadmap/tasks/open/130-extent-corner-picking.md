# 130 — Extent corner picking and rectangle rendering

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 010 §5 (double-sided A3), §6, §7. Depends on 127.

A map's extent is set by picking two corners on the underlying Leaflet map, and drawn back
as a translucent `L.rectangle`.

Extents are a **list of 0–2**, because a double-sided sheet is one map — one handover, one
QR, one reveal — showing two different rectangles. So the editor offers "Tilføj område" for
the reverse side, and each extent can be removed.

The two extents are simply two areas: **nothing labels front or back**, and checkpoints are
not split per side. Do not add either — both sides are handed over together, so the
distinction has no consumer.

Zero extents is normal: a skitse has no extent worth recording.

Corner picking claims map interaction, so it must be mutually exclusive with marker
dragging (task 127).

## Acceptance Criteria

- [ ] Pick two corners on the map to set an extent; picked either way round, stored as
      north-west + south-east
- [ ] Up to two extents per map, each removable, zero allowed
- [ ] Selected map's extents drawn as translucent rectangles
- [ ] No front/back labelling and no per-side checkpoints introduced
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
