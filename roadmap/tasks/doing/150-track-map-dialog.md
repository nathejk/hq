# 150 — TrackMapDialog.vue

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §7. Depends on tasks 147 and 149.

A Leaflet dialog over the current page — **not** a new route, so an operator working through a
list does not lose their place. Reuses `/kort`'s base layers (Dataforsyningen `dtk_25_DAF`
topo, ArcGIS ortofoto, OSM) from `views/KortView.vue`.

Opened by clicking the position indicator: for a **spejder** it shows the patrol (all members,
current and former, plus that team's scans); for personnel and crew it shows that person's own
track. One affordance, and the spejder rule is discoverable rather than hidden in a menu.

Rendering rules that carry the PRD's intent:

- **One polyline per segment**, never one per member — most tracks are a few recorded stretches
  separated by hours. Segment ends get a small terminator so it reads as "data stops here", not
  "the person stopped here". A bridged gap, if drawn at all, is dashed and dimmed, never default.
- **Time-window control is part of the feature**, not a refinement: 30 hours on one screen is a
  tangle, and the real question is "where were they between 22:00 and 02:00?". The window
  doubles as the fidelity control (`maxPoints`), so opening on a sensible default window rather
  than everything.
- **Legend** per member: colour, name (or "tidligere medlem"), membership interval, coverage
  ("3 t 40 min data af 12 t") and last-seen time.
- Scan markers visually distinct from track vertices.
- Empty states: no telemetry → scans only, and say so ("Ingen positioner rapporteret — viser
  kun scanninger"); neither → centred empty state.

Read-only, so the plain live adoption applies (`useLiveResource` + `pending` → `:loading`); no
dirty-guard needed, unlike `KortView`. Depends on `patrulje:{teamId}` **and** the type token
`track` — a point from a newly joined member carries an id the view has never seen, so an
instance-only dependency would miss exactly the member being looked for.

## Acceptance Criteria

- [ ] Dialog component with `/kort`'s base layers and `fitBounds` on the data
- [ ] One polyline per segment; no solid line across a gap
- [ ] Time-window control wired to `from`/`to`/`maxPoints`
- [ ] Legend with colour, name, membership interval, coverage, last seen
- [ ] Scan markers distinguishable from track vertices
- [ ] Both empty states handled
- [ ] `dependsOn` includes `patrulje:{id}` and `track`; `pending` wired to loading
- [ ] Opened from `PositionIndicator`, routing spejder → patrol, others → person
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
