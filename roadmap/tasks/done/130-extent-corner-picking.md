# 130 — Extent corner picking and rectangle rendering

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Pick two corners on the map to set an extent; picked either way round, stored as
      north-west + south-east
- [x] Up to two extents per map, each removable, zero allowed
- [x] Selected map's extents drawn as translucent rectangles
- [x] No front/back labelling and no per-side checkpoints introduced
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: the dialog asks the view to pick a corner pair (it owns the Leaflet
  map); the view returns two clicks and the dialog saves. Picking has to claim map clicks, which the
  context menu and marker dragging also use — so it goes through the same mutual exclusion.
- 2026-09-03 — Split the ownership: the dialog holds the values, the view owns the map. The dialog
  emits `update:picking` to arm, and the view reports the drawn rectangle back through a `pick` prop
  carrying a **sequence number**. The seq is the part worth explaining — comparing coordinates would
  swallow a second, identical rectangle, so re-drawing the same area after a mistake would appear to
  do nothing.
- 2026-09-03 — Added a **rubber band** on mousemove after the first corner. Without it, picking is two
  blind clicks and the operator only learns what they drew afterwards; with it the rectangle is
  visible before committing. Cheap, and it is the difference between a usable tool and a guess.
- 2026-09-03 — The map draws the **draft**, not the saved value, so a rectangle appears the instant it
  is drawn. Otherwise „Gem områder” would appear to be the thing that drew it.
- 2026-09-03 — Arming clears itself after one rectangle. Staying armed would turn the operator's next
  click — meant for a sheet in the list, or a marker — into a silently redrawn extent.
- 2026-09-03 — Extracted `extentFromCorners`, `sameExtent` and `isDegenerate` into the composable and
  tested them. North-is-larger-latitude and west-is-smaller-longitude is trivial to write backwards,
  and a mirrored rectangle is **not obviously wrong on screen** — it is wrong on the printed sheet,
  months later. Both diagonals are covered, including the assertion that drawing from either corner
  yields the identical rectangle (otherwise re-drawing the same area would look like an edit and
  publish an event).
- 2026-09-03 — A degenerate rectangle is caught **before** the request as well as by the API, so the
  message arrives while the operator still remembers the click rather than after a round trip.
- 2026-09-03 — Closing the dialog clears the rectangles, the fade and any armed picker. PrimeVue keeps
  a hidden `Dialog` mounted, so nothing else would: the map would sit crosshaired and half-decorated
  with no visible cause. Found by reading the close path rather than by testing it, which is worth
  admitting.
- 2026-09-03 — Deliberately did **not** add front/back labelling or per-side checkpoints. Two extents
  are two areas; both sides are handed over at once, so neither distinction has a consumer (PRD 010
  §5, confirmed with knj).
- 2026-09-03 — Verified against the real API: two areas saved and read back, three refused with „et
  kort kan højst have 2 områder — for- og bagside”, and `extents: []` clears a sheet to a skitse.
- 2026-09-03 — Completed. 28 tests pass, `vite build` clean, no type errors in the feature. The
  picking interaction itself is still unverified in a browser.
