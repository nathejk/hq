# 150 — TrackMapDialog.vue

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Dialog component with `/kort`'s base layers and `fitBounds` on the data
- [x] One polyline per segment; no solid line across a gap
- [x] Time-window control wired to `from`/`to`/`maxPoints`
- [x] Legend with colour, name, membership interval, coverage, last seen
- [x] Scan markers distinguishable from track vertices
- [x] Both empty states handled
- [x] `dependsOn` includes `patrulje:{id}` and `track`; `pending` wired to loading
- [x] Opened from `PositionIndicator`, routing spejder → patrol, others → person
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: extract `/kort`'s base layers for reuse, a singleton viewer so the glyph can open the map from anywhere, then the dialog.
- 2026-09-03 — Extracted the base layers into `composables/mapLayers.ts` as a **factory**, not a shared object. A Leaflet layer instance belongs to one map: adding the same instance to a second map detaches it from the first, and the symptom would have been `/kort` going blank whenever this dialog opened. `KortView` now calls the factory too, so the definitions (and the Dataforsyningen token and its attribution) exist once.
- 2026-09-03 — Built `useTrackViewer` as a **singleton** with one `<TrackMapDialog>` in `App.vue`, like PrimeVue's toast service. The glyph appears in seven places; mounting a dialog at each would mean seven copies of the state and a Leaflet map per list. **The spejder rule lives in the viewer**, expressed once: a call site says only what it knows ("this is a member of team X"), and the viewer decides that a scout opens the *patrol's* map while a gøgler or crew member opens their own.
- 2026-09-03 — ⚠️ **Caught a real defect in my own first version and restructured.** I had composed `useLiveResource` inside a `watch` so the dialog could follow a changing target. That is wrong twice over: `useLiveResource` registers an `onScopeDispose` for reference counting, so calling it outside setup escapes the component's effect scope, and each target change would leak another watcher. Split into `TrackMapDialog` (frame + window control) and `TrackMapPanel` (map + legend, one resource at setup with a key fixed for its lifetime), with the panel **remounted via `:key`** when the target or window changes — the idiomatic way to let a keyed cache and a changing subject coexist. Map creation moved to `onMounted` with cleanup in `onBeforeUnmount`, which is also simply correct rather than watching a visibility flag.
- 2026-09-03 — Gap rendering: a **dashed, thin, half-opacity connector** with an "Ingen data i N min" tooltip, plus a small terminator dot where each segment's data stops. The intent is that it cannot be misread as a route — an operator sending a car must not believe a straight line across three hours of silence.
- 2026-09-03 — Single-point segments are drawn as a dot. Without that a lone fix would be invisible, and a lone fix is sometimes the only thing we have.
- 2026-09-03 — Scans are drawn **last**, as standard markers rather than circles, so they sit above the lines and read as different in kind. Rows with no coordinates are skipped rather than drawn at (0,0) — which is in the Atlantic and would wreck `fitBounds`. Real dev data has such rows, so this is a case that occurs rather than a hypothetical.
- 2026-09-03 — Fixed palette assigned by position, so a member keeps their colour when the window changes; a legend that reshuffles as an operator switches preset is actively misleading.
- 2026-09-03 — Window presets (1 t / 6 t / 12 t / hele løbet) rather than date pickers, defaulting to **6 hours**. Operators ask "where were they recently", not in epoch milliseconds. The panel key carries a nonce so re-picking the same preset re-reads — "seneste time" must mean the last hour *now*, not the last hour when the dialog opened.
- 2026-09-03 — The indicator became a real `<button type="button">` with `@click.stop`: it opens a dialog, so it must be keyboard-reachable and announced as an action, `type` matters because several call sites sit inside forms, and `.stop` matters because in `HoensegaardView` and `SosTeamCard` the glyph sits beside a name that is itself a row-opening control.
- 2026-09-03 — ✅ `vite build` clean, `vue-tsc` reports no errors in any touched file, and the full frontend suite passes (229 tests, 15 files). Completed.
