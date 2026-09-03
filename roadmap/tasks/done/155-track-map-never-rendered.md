# 155 — track map never rendered: container behind a v-else

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

Reported by knj: the track dialog showed a correct legend — member colours, coverage, "sidst
18.43 · for 40 minutter siden" — beside a **blank white rectangle**. No tiles, no track, and
no Leaflet zoom, scale or attribution controls either.

The absent *controls* are the tell: Leaflet had never initialised at all. A map with a broken
tile source still draws its own chrome.

**Root cause.** In `TrackMapPanel.vue` the map container lived inside `<template v-else>`,
guarded by the loading and empty states:

```
<div v-if="pending && !data">Henter spor…</div>
<div v-else-if="isEmpty">…</div>
<template v-else>
  <div ref="mapContainer" …></div>   ← only exists once data has arrived
```

On first mount `pending` is true, so the div did not exist, `onMounted` found
`mapContainer.value === null` and returned early. Map creation happened **only** in
`onMounted`, so when the data landed and the div appeared, nothing created the map — and
`draw()` returned immediately forever after, since `map` stayed null.

Introduced in task 150. It survived review because every check I ran was on the API, and the
one visual check I could not run is the one that mattered.

## Acceptance Criteria

- [x] Map container rendered unconditionally, so the ref exists for the component's whole life
- [x] Loading, empty and "scans only" notices shown above the map rather than replacing it
- [x] `invalidateSize()` called on draw, not only at mount
- [x] Coverage text no longer reads as a contradiction for isolated points
- [x] `vite build` and the frontend suite pass

## Progress Log

- 2026-09-03 — Task created from knj's report, with the cause identified from the missing Leaflet controls.
- 2026-09-03 — Fixed by rendering the container **unconditionally** and moving the notices above it.
  The map's lifecycle then has exactly one state to cope with, instead of depending on which render
  the ref happened to appear in. It also reads better empty: base tiles centred on the race area say
  "nothing here" far more clearly than a blank rectangle.
- 2026-09-03 — Moved `invalidateSize()` into `draw()` as well as mount. The dialog animates in, so a
  size measured at mount can be mid-transition, and Leaflet renders grey tiles until told otherwise.
  Cheap and idempotent, so calling it on every draw costs nothing.
- 2026-09-03 — Also fixed a smaller honesty problem visible in the same screenshot: the legend read
  "0 min data af 7 min" for a member who plainly had positions. Coverage is deliberately conservative
  — an isolated fix contributes **zero** recorded time, because it evidences an instant rather than an
  interval — so the figure was right but read as a contradiction. It now leads with the point count:
  "2 positioner · 0 min data af 7 min". Both numbers together say the true thing: we have two fixes and
  know nothing about the time between them.
- 2026-09-03 — Lesson for the rest of this PRD: two of the three bugs knj has reported were in
  rendering paths I verified only through the API. `vite build` passing and an endpoint returning
  correct JSON says nothing about whether a map appears.
