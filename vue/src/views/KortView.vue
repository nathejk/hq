<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useDeferredApply } from '@/composables/useDeferredApply'
import { useKort, extentFromCorners, type Extent } from '@/composables/kort'
import {
  clusterPositions,
  isLive,
  simulatePositions,
  usePatruljePositions,
  type PatruljePosition,
  type PositionCluster,
} from '@/composables/patruljePositions'
import { formatClock, formatRelative } from '@/composables/usePositionPresence'
import {
  createBaseLayers,
  DEFAULT_BASE_LAYER,
  RACE_AREA_CENTER,
  RACE_AREA_ZOOM,
} from '@/composables/mapLayers'
import KortSettingsDialog from '@/components/kort/KortSettingsDialog.vue'

const route = useRoute()

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface Checkpoint {
  id: string
  checkgroupId: string
  name: string
  latitude: number | null
  longitude: number | null
  [key: string]: unknown
}

interface Checkgroup {
  id: string
  name: string
  checkpoints: Checkpoint[]
  [key: string]: unknown
}

interface DirtyEntry {
  checkpointId: string
  checkgroupId: string
  latitude: number
  longitude: number
}

// ---------------------------------------------------------------------------
// Refs
// ---------------------------------------------------------------------------
const mapContainer = ref<HTMLDivElement | null>(null)
const contextMenu = ref<HTMLDivElement | null>(null)
let map: L.Map | null = null

const checkgroups = ref<Checkgroup[]>([])
const markers = new Map<string, L.Marker>()

/**
 * The 500 m circle drawn around each checkpoint, by checkpoint id.
 *
 * Kept beside `markers` rather than replacing them, because the two do different jobs: the circle is
 * what an operator reads — it shows how much ground a post effectively covers, which is what decides
 * whether a sheet's area is enough — while the marker stays as the draggable centre and the anchor
 * for the popup and tooltip. `L.Circle` cannot be dragged, so dropping the marker would have cost
 * position editing.
 */
const circles = new Map<string, L.Circle>()

/**
 * Half of the 500 m diameter a checkpoint is drawn with.
 *
 * Metres, not pixels, which is why this is `L.circle` and not `L.circleMarker`: the point is a real
 * distance on the ground, so it has to shrink and grow with the zoom like the terrain does.
 */
const CHECKPOINT_RADIUS_M = 250

/**
 * The zoom at which the 500 m circle starts being worth drawing.
 *
 * Below this it is about as wide as the dot at its centre — 8 px at zoom 11, at this latitude — so it
 * stops describing ground and becomes a smudge that hides the position it is centred on. From zoom 12
 * up it is 16 px and growing, which reads as an area.
 *
 * Under the threshold the circle is hidden and the centre becomes a fixed-size ring instead: still
 * visibly a checkpoint, but no longer pretending to state a distance it cannot show.
 */
const CIRCLE_ZOOM_MIN = 12

// Edit mode
const editMode = ref(false)
const dirtyCheckpoints = ref<Map<string, DirtyEntry>>(new Map())
const saving = ref(false)

// ---------------------------------------------------------------------------
// The printed sheets (PRD 010)
// ---------------------------------------------------------------------------
//
// Loaded through the shared composable rather than another `useLiveResource` call here, so this
// view and the settings dialog cannot end up with different cache keys or dependency tokens.
const { data: kortSheets, refresh: refreshKort } = useKort()

const settingsOpen = ref(false)
/** The sheet whose checkpoints are highlighted; also what the dialog is editing. */
const selectedKortId = ref<string | undefined>(undefined)
/** The dialog holds unsaved work. Task 131 uses this to pause applying live payloads. */
const settingsDirty = ref(false)

/**
 * Moving markers and describing sheets are mutually exclusive.
 *
 * Both own marker interaction — one drags them, the other will click the map to pick corners
 * (task 130) — and both hold unsaved state. Letting them overlap would mean a drag saving
 * positions while a half-typed sheet name sits behind it, with no way to tell which "Gem" an
 * operator meant.
 */
const enterSettings = () => {
  if (editMode.value) return
  settingsOpen.value = true
}

// Closing the dialog must take its map decorations with it: PrimeVue keeps the component mounted
// when hidden, so nothing else would clear the rectangles, the fade or an armed picker — leaving the
// map crosshaired and half-drawn with no visible reason why.
watch(settingsOpen, (open) => {
  if (open) return
  selectedKortId.value = undefined
  extentsPreview.value = []
  overlay.value = []
  hoverExtents.value = []
  pickingExtent.value = false
})

/** The checkpoints drawn on the selected sheet, or none when nothing is selected. */
const highlightedCheckpoints = computed<Set<string>>(() => {
  const id = selectedKortId.value
  if (!id || !kortSheets.value) return new Set()
  const sheets = [...kortSheets.value.kortsaet.flatMap((set) => set.kort), ...kortSheets.value.orphanKort]
  const sheet = sheets.find((candidate) => candidate.id === id)
  return new Set(sheet?.checkpointIds ?? [])
})

/**
 * Fade every checkpoint not on the selected sheet.
 *
 * Applied by walking the existing markers rather than by rebuilding them: a rebuild would drop the
 * popups and, in edit mode, the drag handlers. Opacity rather than removal, because "this
 * checkpoint is not on this sheet" is exactly the mistake an operator is looking for, and a hidden
 * marker cannot be spotted.
 *
 * The circle fades with its marker, and by the same factor — they are one thing on screen, and a
 * faded dot inside a solid circle would read as two different states.
 */
const applyHighlight = () => {
  const highlighted = highlightedCheckpoints.value
  const anySelected = highlighted.size > 0 || selectedKortId.value !== undefined
  markers.forEach((marker, checkpointId) => {
    const on = !anySelected || highlighted.has(checkpointId)
    marker.setOpacity(on ? 1 : 0.35)
    circles.get(checkpointId)?.setStyle(on ? CIRCLE_STYLE : CIRCLE_STYLE_FADED)
  })
}

watch([highlightedCheckpoints, selectedKortId], () => applyHighlight())

// ---------------------------------------------------------------------------
// Extents: drawing them, and picking their corners (PRD 010, task 130)
// ---------------------------------------------------------------------------
//
// The dialog owns the values; this owns the map. So the dialog asks to be armed and this reports
// back the rectangle the operator drew.

/** Rectangles to draw — the dialog's draft, so an unsaved extent is visible immediately. */
const extentsPreview = ref<Extent[]>([])
/** A whole set's areas, shown so an operator can judge coverage against the terrain. */
const overlay = ref<Extent[]>([])
/** The areas of the sheet under the pointer in the dialog's list, drawn emphasised. */
const hoverExtents = ref<Extent[]>([])
/** True while the next two clicks are corners. */
const pickingExtent = ref(false)
/** The rectangle just drawn, handed to the dialog. `seq` so an identical redraw still registers. */
const pick = ref<{ extent: Extent; seq: number } | null>(null)
let pickSeq = 0
let firstCorner: L.LatLng | null = null

/**
 * The result of dragging an existing area, handed to the dialog by index.
 *
 * Separate from `pick` because they are different intents: a pick *creates* the rectangle in an armed
 * slot, this *replaces* the one at a known index. `seq` again, so dragging an area back to where it
 * started still reports — the dialog needs to hear the last word, not only new values.
 */
const extentEdit = ref<{ index: number; extent: Extent; seq: number } | null>(null)
let editSeq = 0

let extentLayer: L.LayerGroup | null = null
let rubberBand: L.Rectangle | null = null

/**
 * Build a north-west/south-east pair from two arbitrary corners.
 *
 * In the composable, with tests: north-is-larger-latitude and west-is-smaller-longitude is trivial
 * to write backwards, and a mirrored rectangle is not obviously wrong on screen — it is wrong on the
 * printed sheet, months later.
 */
const toExtent = (a: L.LatLng, b: L.LatLng): Extent => extentFromCorners(a, b)

const drawExtents = () => {
  if (!map) return
  // Never mid-drag: the drag owns the rectangle and its handles, and rebuilding the layer under the
  // pointer would drop the very layer being dragged. The dialog echoes the draft back on every move,
  // so without this guard a drag would destroy itself on its first millimetre.
  if (draggingExtent) return

  extentLayer?.remove()
  extentLayer = L.layerGroup().addTo(map)

  const rect = (extent: Extent, options: L.PolylineOptions, interactive = false) =>
    L.rectangle(
      [
        [extent.northWest.latitude, extent.northWest.longitude],
        [extent.southEast.latitude, extent.southEast.longitude]
      ],
      { ...options, interactive }
    ).addTo(extentLayer!)

  // A set's areas first, so the hovered and the selected sheet draw over them.
  //
  // Light blue, semi-transparent, with a thin darker border: a set's *other* sheets are context, and
  // context has to be visible as area rather than as four hairlines an operator has to trace. Where
  // two sheets overlap the fills stack and read darker, which is honest — overlap between adjacent
  // sheets is designed in, and seeing where it happens is useful.
  const setArea = { color: '#1e40af', weight: 1 }
  overlay.value.forEach((extent) => rect(extent, { ...setArea, fillColor: '#93c5fd', fillOpacity: 0.25 }))

  // The hovered sheet: the **same** border, and only slightly more fill. Enough to leave no doubt
  // which row the pointer is on, deliberately not enough to draw the eye away from the terrain — the
  // rectangles are there to be judged against the map, not to be looked at. Thickening the border
  // instead also shifts where the edge appears to sit, which matters when where the edges fall is
  // exactly the question.
  hoverExtents.value.forEach((extent) => rect(extent, { ...setArea, fillColor: '#60a5fa', fillOpacity: 0.3 }))

  extentsPreview.value.forEach((extent, index) => {
    // Translucent and thin: the rectangle describes the sheet, it is not the subject of the
    // screen, and a solid fill would hide the checkpoints it is drawn around.
    const shape = rect(
      extent,
      { color: '#2563eb', weight: 2, fillOpacity: 0.08, className: extentsEditable.value ? 'extent-body' : '' },
      extentsEditable.value
    )
    if (extentsEditable.value) attachExtentEditing(shape, index)
  })
}

// ---------------------------------------------------------------------------
// Dragging an area: moving it whole, and resizing it by its corners
// ---------------------------------------------------------------------------
//
// Two-click corner picking (task 130) stays, because it is the only way to draw an area that does not
// exist yet, and it is precise. But adjusting one — nudging a sheet ten degrees east, or pulling its
// edge out to take in a checkpoint — was two clicks that replaced the whole rectangle, which meant
// re-aiming both corners to change one. Dragging is the natural gesture for "this, but a bit over
// there", so the corners are handles and the body moves.
//
// Written by hand rather than by adding Leaflet.Editable or leaflet-draw: those bring an editing model
// for arbitrary geometry, and everything here is an axis-aligned rectangle whose only operations are
// "move" and "move one corner". The whole of it is below and is smaller than the adapter would be.
//
// The dialog remains the owner of the draft. This reports a finished gesture; it does not write the
// value. So "Annullér"/"Gem kort" keep meaning exactly what they meant before.

/** True while a drag is in flight, to keep `drawExtents` from rebuilding the layer being dragged. */
let draggingExtent = false

/**
 * Whether the areas can be dragged right now.
 *
 * Only with a sheet open in the dialog, and never while corner-picking is armed: an armed picker is
 * waiting for two clicks on the map, and handles that swallowed the first one would look like the
 * picker had failed. Checkpoint edit mode is excluded by construction — it and the dialog cannot both
 * be open — but named here so that stays true if that ever changes.
 */
const extentsEditable = computed(
  () => settingsOpen.value && selectedKortId.value !== undefined && !pickingExtent.value && !editMode.value
)

/** The four corners of an extent, in the order the handles are created. */
type Corner = 'nw' | 'ne' | 'se' | 'sw'

const cornerLatLng = (bounds: L.LatLngBounds, corner: Corner): L.LatLng => {
  switch (corner) {
    case 'nw':
      return bounds.getNorthWest()
    case 'ne':
      return bounds.getNorthEast()
    case 'se':
      return bounds.getSouthEast()
    case 'sw':
      return bounds.getSouthWest()
  }
}

/** The bounds that result from dragging one corner to `to`, the opposite corner staying put. */
const boundsWithCornerAt = (bounds: L.LatLngBounds, corner: Corner, to: L.LatLng): L.LatLngBounds => {
  const opposite: Record<Corner, Corner> = { nw: 'se', ne: 'sw', se: 'nw', sw: 'ne' }
  // L.latLngBounds normalises whichever way round the two corners end up, so dragging a corner past
  // its opposite flips the rectangle rather than collapsing it.
  return L.latLngBounds(to, cornerLatLng(bounds, opposite[corner]))
}

const extentFromBounds = (bounds: L.LatLngBounds): Extent => ({
  northWest: { latitude: bounds.getNorth(), longitude: bounds.getWest() },
  southEast: { latitude: bounds.getSouth(), longitude: bounds.getEast() }
})

/** Hand a finished gesture to the dialog, which owns the draft. */
const reportExtent = (index: number, bounds: L.LatLngBounds) => {
  editSeq += 1
  extentEdit.value = { index, extent: extentFromBounds(bounds), seq: editSeq }
}

/**
 * Give one drawn rectangle its corner handles and its move behaviour.
 *
 * Handles are markers rather than SVG vertices so that Leaflet's own dragging does the work, including
 * touch. They are drawn into the same layer group as the rectangle, so a redraw disposes of everything
 * together and no handle can outlive the shape it belongs to.
 */
const attachExtentEditing = (shape: L.Rectangle, index: number) => {
  if (!map) return
  const corners: Corner[] = ['nw', 'ne', 'se', 'sw']

  const handles = corners.map((corner) => {
    const handle = L.marker(cornerLatLng(shape.getBounds(), corner), {
      draggable: true,
      // Square, small, and cursor-hinted per corner, so it reads as a resize grip rather than as
      // another checkpoint.
      icon: L.divIcon({
        className: `extent-handle extent-handle--${corner}`,
        html: '<span></span>',
        iconSize: [14, 14],
        iconAnchor: [7, 7]
      }),
      // Above the checkpoint markers: while a sheet is open, its corners are what the operator is
      // reaching for.
      zIndexOffset: 1000
    }).addTo(extentLayer!)

    handle.on('dragstart', () => {
      draggingExtent = true
    })

    handle.on('drag', () => {
      const next = boundsWithCornerAt(shape.getBounds(), corner, handle.getLatLng())
      shape.setBounds(next)
      // Keep the other three grips on the rectangle they describe. The dragged one follows the pointer
      // on its own, and re-placing it here would fight Leaflet's drag.
      syncHandles(next, handle)
    })

    handle.on('dragend', () => {
      draggingExtent = false
      reportExtent(index, shape.getBounds())
    })

    return { corner, handle }
  })

  const syncHandles = (bounds: L.LatLngBounds, except?: L.Marker) => {
    handles.forEach(({ corner, handle }) => {
      if (handle !== except) handle.setLatLng(cornerLatLng(bounds, corner))
    })
  }

  // Moving the whole area. Not Leaflet's marker dragging — a rectangle has none — so the pointer is
  // followed on the map itself, with map panning switched off for the duration. Without that, dragging
  // the rectangle would drag the map underneath it and the area would appear not to move at all.
  shape.on('mousedown', (down: L.LeafletMouseEvent) => {
    if (!map || !extentsEditable.value) return
    // Left button only. A right-click belongs to the context menu, and starting a move on it would
    // leave the map panning-disabled while the menu was open.
    if ((down.originalEvent as MouseEvent).button !== 0) return
    L.DomEvent.stopPropagation(down)
    draggingExtent = true
    map.dragging.disable()
    map.getContainer().style.cursor = 'grabbing'

    let last = down.latlng
    let finished = false

    const onMove = (move: L.LeafletMouseEvent) => {
      const bounds = shape.getBounds()
      const dLat = move.latlng.lat - last.lat
      const dLng = move.latlng.lng - last.lng
      last = move.latlng
      const moved = L.latLngBounds(
        [bounds.getSouth() + dLat, bounds.getWest() + dLng],
        [bounds.getNorth() + dLat, bounds.getEast() + dLng]
      )
      shape.setBounds(moved)
      syncHandles(moved)
    }

    // Ends on the document as well as on the map: releasing the button outside the map — over the
    // dialog, or outside the window — would otherwise never reach Leaflet, and the map would stay
    // un-pannable with a grabbing cursor and no way back. Guarded so whichever fires first wins.
    const onUp = () => {
      if (finished) return
      finished = true
      map!.off('mousemove', onMove)
      map!.off('mouseup', onUp)
      document.removeEventListener('mouseup', onUp)
      map!.dragging.enable()
      map!.getContainer().style.cursor = ''
      draggingExtent = false
      reportExtent(index, shape.getBounds())
    }

    map.on('mousemove', onMove)
    map.on('mouseup', onUp)
    document.addEventListener('mouseup', onUp)
  })
}

watch(extentsEditable, () => drawExtents())

watch([extentsPreview, overlay, hoverExtents], () => drawExtents(), { deep: true })

const clearRubberBand = () => {
  rubberBand?.remove()
  rubberBand = null
  firstCorner = null
}

watch(pickingExtent, (armed) => {
  if (!armed) clearRubberBand()
  // The crosshair is the only signal that a click means something different now.
  if (map) map.getContainer().style.cursor = armed ? 'crosshair' : ''
})

const onPickClick = (e: L.LeafletMouseEvent) => {
  if (!pickingExtent.value) return
  if (!firstCorner) {
    firstCorner = e.latlng
    return
  }
  pickSeq += 1
  pick.value = { extent: toExtent(firstCorner, e.latlng), seq: pickSeq }
  clearRubberBand()
}

/** A rubber band between the first corner and the pointer, so the shape is visible before the click. */
const onPickMove = (e: L.LeafletMouseEvent) => {
  if (!pickingExtent.value || !firstCorner || !map) return
  const bounds = L.latLngBounds(firstCorner, e.latlng)
  if (rubberBand) rubberBand.setBounds(bounds)
  else rubberBand = L.rectangle(bounds, { color: '#f59e0b', weight: 2, dashArray: '4 4', fillOpacity: 0.05 }).addTo(map)
}

// Snapshot of original positions so we can revert on cancel
let originalPositions = new Map<string, { latitude: number | null; longitude: number | null }>()

// Route lines layer group
let routeLinesLayer: L.LayerGroup | null = null

// Right-click state
const menuVisible = ref(false)
const menuX = ref(0)
const menuY = ref(0)
let menuLatLng: L.LatLng | null = null

// ---------------------------------------------------------------------------
// Base layers
// ---------------------------------------------------------------------------
// Shared with the track dialog (task 150) via a factory rather than a module-level object: a Leaflet
// layer instance belongs to one map, so two maps sharing instances make the first go blank.
const baseLayers = createBaseLayers()

// ---------------------------------------------------------------------------
// Fix default Leaflet marker icons (webpack / vite asset issue)
// ---------------------------------------------------------------------------
//
// Kept although this view no longer places a single default-icon marker — checkpoints are a dot in a
// circle, and the distance labels are div-icons. `TrackMapPanel` does use the default icon, and this
// patches a Leaflet global, so removing it here would break that panel for anyone who happened to
// reach it without loading this view first. It belongs in the Leaflet setup rather than in a view;
// leaving it put keeps that a separate, deliberate move.
// @ts-ignore
delete (L.Icon.Default.prototype as any)._getIconUrl
L.Icon.Default.mergeOptions({
  iconRetinaUrl: new URL('leaflet/dist/images/marker-icon-2x.png', import.meta.url).href,
  iconUrl: new URL('leaflet/dist/images/marker-icon.png', import.meta.url).href,
  shadowUrl: new URL('leaflet/dist/images/marker-shadow.png', import.meta.url).href
})

// ---------------------------------------------------------------------------
// Haversine distance (km)
// ---------------------------------------------------------------------------
const haversineKm = (lat1: number, lon1: number, lat2: number, lon2: number): number => {
  const R = 6371 // Earth radius in km
  const toRad = (deg: number) => (deg * Math.PI) / 180
  const dLat = toRad(lat2 - lat1)
  const dLon = toRad(lon2 - lon1)
  const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) * Math.sin(dLon / 2)
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

// ---------------------------------------------------------------------------
// Load checkgroups + checkpoints from API
// ---------------------------------------------------------------------------
//
// This page is live and cached like the others, but it cannot simply re-render on
// new data: markers are placed imperatively on the Leaflet map, and edit mode holds
// unsaved positions. So the resource is the source of truth and `applyPayload`
// re-projects it onto the map, with two rules that matter:
//
//   - Nothing is applied while editing. An incoming revalidation would otherwise
//     discard the operator's unsaved marker positions, which is far worse than
//     briefly showing older data. The payload is applied when edit mode ends.
//   - The map is fitted to the checkpoints only on the first apply, so a later
//     update does not yank the viewport out from under someone.
//
// The payload is cloned before use because dragging mutates cp.latitude/longitude
// in place; mutating the cached value would corrupt it for the next reader.
const { data: kortData, error: kortError, refresh: refreshCheckgroups } = useLiveResource(
  'kort:checkgroups',
  async () => {
    const rsp = await http.get('/checkgroups', { withCredentials: true })
    return {
      checkgroups: (rsp.data.checkgroups ?? []) as Checkgroup[],
      checkpoints: (rsp.data.checkpoints ?? []) as Checkpoint[]
    }
  },
  { dependsOn: ['checkgroup', 'checkgroups', 'checkpoint'] }
)

watch(kortError, (err) => {
  if (err) console.error('Failed to load checkgroups', err)
})

/** The map has been fitted once; later applies leave the viewport alone. */
let fitted = false

const applyPayload = (payload: { checkgroups: Checkgroup[]; checkpoints: Checkpoint[] }) => {
  if (!map) return

  const cgs: Checkgroup[] = structuredClone(payload.checkgroups)
  const cps: Checkpoint[] = structuredClone(payload.checkpoints)
  cgs.forEach((cg) => {
    cg.checkpoints = cps.filter((cp) => cp.checkgroupId === cg.id)
  })

  // Drop every existing marker: a checkpoint may have been deleted or moved by
  // someone else, and placeMarker only replaces the ones it is given.
  markers.forEach((marker) => marker.remove())
  markers.clear()
  circles.forEach((circle) => circle.remove())
  circles.clear()

  checkgroups.value = cgs

  const bounds: L.LatLng[] = []
  cgs.forEach((cg) => {
    cg.checkpoints.forEach((cp) => {
      if (cp.latitude && cp.longitude) {
        placeMarker(cp, [cp.latitude, cp.longitude])
        bounds.push(L.latLng(cp.latitude, cp.longitude))
      }
    })
  })

  if (!fitted && bounds.length > 0) {
    map.fitBounds(L.latLngBounds(bounds), { padding: [50, 50], maxZoom: 15 })
    fitted = true
  }

  // The markers were just rebuilt, so any highlight has to be re-applied — otherwise a live
  // payload arriving with the settings dialog open would quietly un-fade the map.
  applyHighlight()

  // Positions may have moved or checkpoints appeared, so the legs are stale.
  drawCrowLines()
}

/**
 * Whether applying an incoming payload has to wait.
 *
 * Three reasons, and they are all the same reason: something on screen would be destroyed by a
 * rebuild. Edit mode holds unsaved marker positions; the settings dialog holds unsaved text, ticks
 * and rectangles; and before the map exists there is nothing to draw onto.
 *
 * The map's readiness is part of the *condition* rather than a separate "apply on mount" call. That
 * is what lets a warm cache render on arrival: the payload is already there at setup, gets held
 * because the map is not ready, and is applied the moment it is — no request, no empty frame, and
 * no second code path that can drift from this one.
 */
const mapReady = ref(false)
const applyPaused = computed(() => !mapReady.value || editMode.value || settingsDirty.value)

// The house mechanism (`composables/useDeferredApply.ts`), not a flag of this view's own. KortView
// had its own `applyDeferred` bookkeeping from before that composable existed; keeping it and adding
// the dialog's dirty state as a third condition would have meant two mechanisms doing one job — and
// the composable exists precisely because watching the *condition* cannot miss an exit, whereas a
// helper called from each exit path can.
const { updatesWaiting } = useDeferredApply(kortData, applyPaused, applyPayload)

// ---------------------------------------------------------------------------
// Route distance computation
// ---------------------------------------------------------------------------

/** Get checkgroups that have at least one checkpoint with coordinates, in order */
const positionedCheckgroups = computed(() => {
  return checkgroups.value.filter((cg) => cg.checkpoints.some((cp) => cp.latitude != null && cp.longitude != null))
})

/**
 * Build an adjacency structure: for each consecutive pair of positioned checkgroups,
 * compute the distance from every positioned checkpoint in group[i] to every positioned
 * checkpoint in group[i+1].
 */
interface LegEdge {
  fromCp: Checkpoint
  toCp: Checkpoint
  distKm: number
}

interface LegBetweenGroups {
  fromGroup: Checkgroup
  toGroup: Checkgroup
  edges: LegEdge[]
}

const computeLegs = (): LegBetweenGroups[] => {
  const cgs = positionedCheckgroups.value
  const legs: LegBetweenGroups[] = []
  for (let i = 0; i < cgs.length - 1; i++) {
    const fromCg = cgs[i]
    const toCg = cgs[i + 1]
    const edges: LegEdge[] = []
    const fromCps = fromCg.checkpoints.filter((cp) => cp.latitude != null && cp.longitude != null)
    const toCps = toCg.checkpoints.filter((cp) => cp.latitude != null && cp.longitude != null)
    for (const fcp of fromCps) {
      for (const tcp of toCps) {
        edges.push({
          fromCp: fcp,
          toCp: tcp,
          distKm: haversineKm(fcp.latitude!, fcp.longitude!, tcp.latitude!, tcp.longitude!)
        })
      }
    }
    if (edges.length > 0) {
      legs.push({ fromGroup: fromCg, toGroup: toCg, edges })
    }
  }
  return legs
}

/**
 * Compute shortest and longest total route.
 *
 * A route picks exactly one checkpoint per checkgroup.
 * The total distance is the sum of distances between consecutive chosen checkpoints.
 *
 * We use dynamic programming over the ordered checkgroups.
 * State: for each positioned checkpoint in group[i], the min/max cumulative distance
 * to reach that checkpoint from group[0].
 */
const routeDistances = ref<{ shortest: number | null; longest: number | null }>({
  shortest: null,
  longest: null
})

const recomputeRouteDistances = () => {
  const cgs = positionedCheckgroups.value
  if (cgs.length < 2) {
    routeDistances.value = { shortest: null, longest: null }
    return
  }

  // Map from checkpoint id -> { minDist, maxDist } reaching that checkpoint
  let current = new Map<string, { minDist: number; maxDist: number }>()

  // Initialize first group: distance 0 to reach any checkpoint in first group
  const firstCps = cgs[0].checkpoints.filter((cp) => cp.latitude != null && cp.longitude != null)
  if (firstCps.length === 0) {
    routeDistances.value = { shortest: null, longest: null }
    return
  }
  for (const cp of firstCps) {
    current.set(cp.id, { minDist: 0, maxDist: 0 })
  }

  // Process each subsequent group
  for (let i = 1; i < cgs.length; i++) {
    const nextCps = cgs[i].checkpoints.filter((cp) => cp.latitude != null && cp.longitude != null)
    if (nextCps.length === 0) continue

    const next = new Map<string, { minDist: number; maxDist: number }>()
    for (const tcp of nextCps) {
      let best = Infinity
      let worst = -Infinity
      current.forEach((state, _fromId) => {
        const fromCp = findCheckpointById(_fromId)
        if (!fromCp || fromCp.latitude == null || fromCp.longitude == null) return
        const d = haversineKm(fromCp.latitude!, fromCp.longitude!, tcp.latitude!, tcp.longitude!)
        const totalMin = state.minDist + d
        const totalMax = state.maxDist + d
        if (totalMin < best) best = totalMin
        if (totalMax > worst) worst = totalMax
      })
      if (best < Infinity) {
        next.set(tcp.id, { minDist: best, maxDist: worst })
      }
    }
    current = next
  }

  // Extract global min/max from the last group's states
  let globalMin = Infinity
  let globalMax = -Infinity
  current.forEach((state) => {
    if (state.minDist < globalMin) globalMin = state.minDist
    if (state.maxDist > globalMax) globalMax = state.maxDist
  })

  routeDistances.value = {
    shortest: globalMin < Infinity ? globalMin : null,
    longest: globalMax > -Infinity ? globalMax : null
  }
}

const findCheckpointById = (id: string): Checkpoint | undefined => {
  for (const cg of checkgroups.value) {
    for (const cp of cg.checkpoints) {
      if (cp.id === id) return cp
    }
  }
  return undefined
}

// ---------------------------------------------------------------------------
// Route lines drawing
// ---------------------------------------------------------------------------
const drawRouteLines = () => {
  if (!map) return

  // Clear existing lines
  if (routeLinesLayer) {
    routeLinesLayer.clearLayers()
  } else {
    routeLinesLayer = L.layerGroup().addTo(map)
  }

  if (!editMode.value) return

  const legs = computeLegs()

  // Use a palette of hues for different leg pairs
  const legColors = ['#6366f1', '#ec4899', '#f59e0b', '#10b981', '#ef4444', '#8b5cf6', '#06b6d4', '#f97316']

  legs.forEach((leg, legIdx) => {
    const color = legColors[legIdx % legColors.length]

    leg.edges.forEach((edge) => {
      const from = L.latLng(edge.fromCp.latitude!, edge.fromCp.longitude!)
      const to = L.latLng(edge.toCp.latitude!, edge.toCp.longitude!)

      // Draw the polyline
      const line = L.polyline([from, to], {
        color,
        weight: 4,
        opacity: 0.7,
        dashArray: '6 4'
      })
      routeLinesLayer!.addLayer(line)

      // Place a distance label at the midpoint
      const midLat = (from.lat + to.lat) / 2
      const midLng = (from.lng + to.lng) / 2
      const label = L.marker([midLat, midLng], {
        icon: L.divIcon({
          className: 'distance-label',
          html: `<span style="
            background: ${color};
            color: #fff;
            padding: 1px 5px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            white-space: nowrap;
            pointer-events: none;
            box-shadow: 0 1px 3px rgba(0,0,0,0.3);
          ">${edge.distKm.toFixed(1)} km</span>`,
          iconSize: [0, 0],
          iconAnchor: [0, 0]
        }),
        interactive: false
      })
      routeLinesLayer!.addLayer(label)
    })
  })

  // Recompute shortest / longest
  recomputeRouteDistances()
}

const clearRouteLines = () => {
  if (routeLinesLayer) {
    routeLinesLayer.clearLayers()
  }
  routeDistances.value = { shortest: null, longest: null }
}

// ---------------------------------------------------------------------------
// Fugleflugslinjer — an overlay in the layers control
// ---------------------------------------------------------------------------
//
// The same edges edit mode draws — every checkpoint to every checkpoint in the next checkgroup, which
// is what a patrol may actually walk — but available without entering edit mode, and drawn as thin red
// lines rather than thick coloured ones with distance labels. Two different questions: edit mode asks
// "how far is this leg now that I have moved the post?", this asks "where do the legs run?".
//
// Reusing `computeLegs()` is the point. A second traversal of the checkgroups would be a second place
// for "the next checkgroup" to be defined, and the two would eventually disagree about what the
// operator is looking at.
//
// It is a layers-control overlay rather than a button of ours because Leaflet already provides that
// checkbox, and it is where an operator looks for things to switch on over the map.
const CROW_LINES_LABEL = 'Fugleflugslinjer'

let crowLinesLayer: L.LayerGroup | null = null

/**
 * Fill the overlay with one thin line per possible leg.
 *
 * Populated whether or not the overlay is currently on the map: a `LayerGroup` that is not added just
 * holds its children until it is, so Leaflet's checkbox does the showing and hiding and there is no
 * `overlayadd` wiring to keep in step. At roughly a hundred edges the cost of drawing into the void is
 * not worth a state machine.
 */
const drawCrowLines = () => {
  if (!crowLinesLayer) return
  crowLinesLayer.clearLayers()

  computeLegs().forEach((leg) => {
    leg.edges.forEach((edge) => {
      crowLinesLayer!.addLayer(
        L.polyline(
          [
            [edge.fromCp.latitude!, edge.fromCp.longitude!],
            [edge.toCp.latitude!, edge.toCp.longitude!],
          ],
          // Thin, and not interactive: with every pair drawn there are a lot of these, and they are
          // background information — they must not swallow a right-click meant for the map or a click
          // meant for a checkpoint.
          //
          // The weight here is the zoomed-out one; CSS thickens it slightly from CIRCLE_ZOOM_MIN up,
          // keyed off the same `kort--far` class as the checkpoint circles. Done in CSS rather than by
          // redrawing on every zoom because stroke width is presentation, and a hundred polylines
          // rebuilt per zoom step is work for nothing.
          { color: '#dc2626', weight: 1, opacity: 0.6, interactive: false, className: 'crow-line' },
        ),
      )
    })
  })
}

// ---------------------------------------------------------------------------
// Edit mode helpers
// ---------------------------------------------------------------------------
const enterEditMode = () => {
  // Refused while the settings dialog is open, for the reason given on enterSettings: both own
  // marker interaction and both hold unsaved state.
  if (settingsOpen.value) return
  editMode.value = true
  dirtyCheckpoints.value = new Map()

  // Take a snapshot of all current checkpoint positions so we can revert
  originalPositions = new Map()
  checkgroups.value.forEach((cg) => {
    cg.checkpoints.forEach((cp) => {
      originalPositions.set(cp.id, {
        latitude: cp.latitude,
        longitude: cp.longitude
      })
    })
  })

  // Make all existing markers draggable
  markers.forEach((marker) => {
    marker.dragging?.enable()
  })

  // Draw route lines
  drawRouteLines()
}

const cancelEditMode = () => {
  // Revert all dirty changes
  checkgroups.value.forEach((cg) => {
    cg.checkpoints.forEach((cp) => {
      const orig = originalPositions.get(cp.id)
      if (orig) {
        cp.latitude = orig.latitude
        cp.longitude = orig.longitude

        // Move marker back or remove if it was newly placed. The circle goes with it either way —
        // leaving a 500 m circle behind on a cancelled placement would look like a saved checkpoint.
        if (markers.has(cp.id)) {
          if (orig.latitude && orig.longitude) {
            markers.get(cp.id)!.setLatLng([orig.latitude, orig.longitude])
            circles.get(cp.id)?.setLatLng([orig.latitude, orig.longitude])
          } else {
            // It was placed freshly during edit — remove the marker
            markers.get(cp.id)!.remove()
            markers.delete(cp.id)
            circles.get(cp.id)?.remove()
            circles.delete(cp.id)
          }
        }
      }
    })
  })

  editMode.value = false
  dirtyCheckpoints.value = new Map()
  originalPositions = new Map()

  // Make all markers non-draggable
  markers.forEach((marker) => {
    marker.dragging?.disable()
  })

  // Clear route lines
  clearRouteLines()

  // Positions were just reverted, so the legs are wrong until redrawn. Not covered by the deferred
  // apply below: if no payload arrived while editing, nothing else would redraw them.
  drawCrowLines()

  // Nothing else to call here: leaving edit mode flips `applyPaused`, and useDeferredApply applies
  // whatever arrived meanwhile. That is the point of watching the condition — an exit path cannot
  // forget to ask.
}

const saveChanges = async () => {
  if (dirtyCheckpoints.value.size === 0) {
    editMode.value = false
    markers.forEach((marker) => {
      marker.dragging?.disable()
    })
    clearRouteLines()
    return
  }

  saving.value = true

  try {
    // Group dirty entries by checkgroupId
    const grouped = new Map<string, { id: string; latitude: number; longitude: number }[]>()
    dirtyCheckpoints.value.forEach((entry) => {
      if (!grouped.has(entry.checkgroupId)) {
        grouped.set(entry.checkgroupId, [])
      }
      grouped.get(entry.checkgroupId)!.push({
        id: entry.checkpointId,
        latitude: entry.latitude,
        longitude: entry.longitude
      })
    })

    // Send one PUT per checkgroup
    const promises: Promise<any>[] = []
    grouped.forEach((checkpoints, checkgroupId) => {
      const payload = {
        checkpoints: checkpoints.map((cp) => ({
          id: cp.id,
          latitude: cp.latitude,
          longitude: cp.longitude
        }))
      }
      promises.push(http.put(`/checkgroup/${checkgroupId}`, payload, { withCredentials: true }))
    })

    await Promise.all(promises)
    console.log(`Saved ${dirtyCheckpoints.value.size} checkpoint position(s)`)

    editMode.value = false
    dirtyCheckpoints.value = new Map()
    originalPositions = new Map()

    // Make all markers non-draggable
    markers.forEach((marker) => {
      marker.dragging?.disable()
    })

    // Clear route lines
    clearRouteLines()

    // Our own PUTs will come back as signals too, but revalidate directly rather
    // than waiting for the round trip through the stream.
    void refreshCheckgroups()
  } catch (err) {
    console.error('Failed to save checkpoint positions', err)
    alert('Der opstod en fejl under gem af positioner. Prøv igen.')
  } finally {
    saving.value = false
  }
}

const markDirty = (cp: Checkpoint) => {
  if (cp.latitude != null && cp.longitude != null) {
    dirtyCheckpoints.value.set(cp.id, {
      checkpointId: cp.id,
      checkgroupId: cp.checkgroupId,
      latitude: cp.latitude,
      longitude: cp.longitude
    })
  }
  // Redraw route lines whenever a checkpoint moves
  if (editMode.value) {
    drawRouteLines()
  }
  // The crow lines follow a moved checkpoint too, whether or not edit mode is drawing its own: they
  // are the same edges, so leaving them behind would show two different answers at once.
  drawCrowLines()
}

// ---------------------------------------------------------------------------
// Marker helpers
// ---------------------------------------------------------------------------
/**
 * How a checkpoint's 500 m circle is drawn.
 *
 * Red, so it cannot be confused with the blue sheet areas it is judged against: the whole reason for
 * drawing it is to see whether a sheet's rectangle covers the ground its posts reach, and two shades
 * of one colour would make that comparison harder rather than easier. Thin border and a faint fill,
 * because forty of these overlap.
 */
const CIRCLE_STYLE: L.PathOptions = {
  color: '#dc2626',
  weight: 3,
  fillColor: '#dc2626',
  fillOpacity: 0.08,
  opacity: 1,
}

/** The same circle, faded with its marker when it is not on the selected sheet. */
const CIRCLE_STYLE_FADED: L.PathOptions = { ...CIRCLE_STYLE, opacity: 0.35, fillOpacity: 0.03 }

const placeMarker = (cp: Checkpoint, latlng: [number, number] | L.LatLng) => {
  if (!map) return

  // Remove any existing marker and circle for this checkpoint
  markers.get(cp.id)?.remove()
  circles.get(cp.id)?.remove()

  const checkgroup = checkgroups.value.find((cg) => cg.id === cp.checkgroupId)
  const groupName = checkgroup?.name ?? ''

  // `interactive: false` so the circle is scenery: at 500 m across it covers a lot of map, and an
  // interactive one would swallow the right-click that places a checkpoint and the drags that pan.
  const circle = L.circle(latlng, {
    ...CIRCLE_STYLE,
    radius: CHECKPOINT_RADIUS_M,
    interactive: false,
    // Named so CSS can hide every circle at once when zoomed too far out (see farOut).
    className: 'cp-circle',
  }).addTo(map)

  // The centre: a small dot rather than Leaflet's balloon, since the position *is* the centre of the
  // circle and a balloon's tip sitting on it reads as an offset. The icon is bigger than the dot on
  // purpose — it is the grab target when dragging positions in edit mode.
  const marker = L.marker(latlng, {
    draggable: editMode.value,
    icon: L.divIcon({
      className: 'cp-centre',
      html: '<span></span>',
      iconSize: [18, 18],
      iconAnchor: [9, 9],
    }),
  })
    .addTo(map)
    .bindPopup(`<strong>${cp.name}</strong><br><span style="color:#666">${groupName}</span>`)
    .bindTooltip(cp.name, { permanent: false, direction: 'top', offset: [0, -12] })

  // The circle follows during the drag, not only at the end: it is what the operator is aiming with,
  // and a circle that jumped into place afterwards would make them aim at the dot instead.
  marker.on('drag', () => circle.setLatLng(marker.getLatLng()))

  marker.on('dragend', () => {
    if (!editMode.value) return
    const pos = marker.getLatLng()
    circle.setLatLng(pos)
    cp.latitude = pos.lat
    cp.longitude = pos.lng
    markDirty(cp)
    console.log(`Moved ${cp.name} to ${pos.lat.toFixed(6)}, ${pos.lng.toFixed(6)}`)
  })

  markers.set(cp.id, marker)
  circles.set(cp.id, circle)

  // Update the checkpoint data
  cp.latitude = typeof latlng === 'object' && 'lat' in latlng ? latlng.lat : (latlng as [number, number])[0]
  cp.longitude = typeof latlng === 'object' && 'lng' in latlng ? latlng.lng : (latlng as [number, number])[1]
}

// ---------------------------------------------------------------------------
// Context menu
// ---------------------------------------------------------------------------
const hideContextMenu = () => {
  menuVisible.value = false
}

const onMapContextMenu = (e: L.LeafletMouseEvent) => {
  if (!editMode.value) return
  e.originalEvent.preventDefault()
  menuLatLng = e.latlng
  menuX.value = e.originalEvent.clientX
  menuY.value = e.originalEvent.clientY
  menuVisible.value = true
}

const pickCheckpoint = (cp: Checkpoint) => {
  if (!menuLatLng || !editMode.value) return
  placeMarker(cp, menuLatLng)
  markDirty(cp)
  hideContextMenu()
}

// Close context menu on any click / map interaction
const onDocumentClick = () => hideContextMenu()

// ---------------------------------------------------------------------------
// Zoom
// ---------------------------------------------------------------------------
//
// Tracked for one reason: a checkpoint's 500 m circle is a real distance, so below CIRCLE_ZOOM_MIN it
// shrinks to the size of the dot at its centre and stops meaning anything. The readout that was here
// while that threshold was being chosen is gone — it reported the circle's diameter in pixels, which is
// how 12 was picked.
const zoom = ref(RACE_AREA_ZOOM)

/**
 * Too far out for the 500 m circle to mean anything — see CIRCLE_ZOOM_MIN.
 *
 * Applied as one class on the map *wrapper* rather than by adding and removing a layer per checkpoint:
 * the circles carry a `cp-circle` class, so CSS can switch the whole map's presentation in one place.
 * Nothing is destroyed, so no popup, drag handler or fade state has to be rebuilt when the operator
 * zooms back in.
 */
const farOut = computed(() => zoom.value < CIRCLE_ZOOM_MIN)

const readZoom = () => {
  if (!map) return
  zoom.value = map.getZoom()
}

// ---------------------------------------------------------------------------
// Sidst kendte position — an overlay in the layers control
// ---------------------------------------------------------------------------
//
// Every patrulje's last known position: the newer of a member's reported position and the team's last
// QR scan. See `composables/patruljePositions.ts` for why those two are different kinds of evidence
// and why nothing here interpolates between them.
//
// Filled dot = live (under five minutes). Hollow = the last thing we heard. Overlapping dots are
// grouped into one bigger dot carrying the count, because five patrols in one spot must not look like
// one patrol — and a group is clickable for the list of who is in it.
const POSITIONS_LABEL = 'Sidst kendte position'

let positionsLayer: L.LayerGroup | null = null

const { data: positionsData } = usePatruljePositions()

/**
 * The positions to draw — real, or simulated on request.
 *
 * `?simulate=200` exists because a development database has no telemetry and no scans, so this layer
 * is empty exactly when its density and its clustering are what need looking at. Dev builds only: it
 * is scaffolding, and fake patrols on an operator's screen during a race would be worse than no
 * feature at all.
 */
const positions = computed<PatruljePosition[]>(() => {
  const simulate = Number(route.query.simulate)
  if (import.meta.env.DEV && Number.isFinite(simulate) && simulate > 0) {
    return simulatePositions(Math.min(simulate, 2000))
  }
  return positionsData.value?.positions ?? []
})

/**
 * Draw the dots, grouping the ones that would overlap.
 *
 * Clustering is by *screen* distance, so this has to run again on every zoom — `clusterPositions`
 * takes the projection rather than owning one, and `latLngToLayerPoint` is that projection at the
 * current zoom.
 */
const drawPositions = () => {
  if (!map || !positionsLayer) return
  positionsLayer.clearLayers()

  const clusters = clusterPositions(positions.value, (position) =>
    map!.latLngToLayerPoint([position.latitude, position.longitude])
  )

  clusters.forEach((cluster) => {
    const count = cluster.members.length
    const grouped = count > 1
    // One glyph type for both cases, sized by how many are in it: a group is the same thing as a dot,
    // only more of it, and a second icon design would suggest otherwise. The count is rendered rather
    // than implied, because "bigger" alone cannot tell three apart from thirty.
    const size = grouped ? Math.min(34, 18 + String(count).length * 6) : 12

    const marker = L.marker([cluster.latitude, cluster.longitude], {
      icon: L.divIcon({
        className: `pos-dot${cluster.live ? ' pos-dot--live' : ''}${grouped ? ' pos-dot--group' : ''}`,
        html: grouped ? `<span>${count}</span>` : '<span></span>',
        iconSize: [size, size],
        iconAnchor: [size / 2, size / 2]
      }),
      // Above the checkpoint dots: this layer is switched on to answer a question about patrols.
      zIndexOffset: 500
    }).addTo(positionsLayer!)

    marker.bindPopup(positionPopup(cluster), { maxHeight: 260 })
  })
}

/**
 * What a dot says when clicked.
 *
 * A group lists its patrols with the age of each, because "12 patruljer" is only useful if you can
 * then find out which. Long lists are capped: the popup is for orientation, and thirty names in a
 * scrolling box is a table pretending to be a tooltip.
 */
const positionPopup = (cluster: PositionCluster): string => {
  const line = (p: PatruljePosition) => {
    const live = isLive(p) ? ' · <span style="color:#16a34a">live</span>' : ''
    const via = p.source === 'scan' ? 'scan' : 'telemetri'
    return `<div><strong>${escapeHtml(p.teamNumber)}</strong> ${escapeHtml(p.name)}<br>
      <span style="color:#666">${formatRelative(p.ts)} · ${formatClock(p.ts)} · ${via}${live}</span></div>`
  }

  if (cluster.members.length === 1) return line(cluster.members[0])

  const shown = cluster.members.slice(0, 12)
  const rest = cluster.members.length - shown.length
  return (
    `<div><strong>${cluster.members.length} patruljer</strong> her</div>` +
    `<div style="margin-top:4px;display:flex;flex-direction:column;gap:4px">${shown.map(line).join('')}</div>` +
    (rest > 0 ? `<div style="margin-top:4px;color:#666">… og ${rest} mere</div>` : '')
  )
}

/** Names and numbers come from operator input, and this builds HTML by hand. */
const escapeHtml = (value: string) =>
  value.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)

// Zoom changes which dots overlap, so the grouping is recomputed — not just re-placed.
watch(positions, () => drawPositions(), { deep: true })

/**
 * Re-style the dots as they age.
 *
 * Liveness is a fact about *now*, not about the payload: with no new data arriving, a dot filled at
 * 21:58 is still filled at 22:30 unless something redraws it — and a five-minute promise that quietly
 * becomes a half-hour lie is worse than no promise. A minute is fine granularity against a five-minute
 * threshold, and redrawing a few hundred div-icons costs nothing.
 */
let livenessTimer: number | undefined

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------
onMounted(async () => {
  if (!mapContainer.value) return

  map = L.map(mapContainer.value, {
    center: RACE_AREA_CENTER,
    zoom: RACE_AREA_ZOOM,
    zoomControl: false,
    layers: [baseLayers[DEFAULT_BASE_LAYER]]
  })

  L.control.zoom({ position: 'topright' }).addTo(map)
  // The overlay starts unchecked — not added to the map — so the map opens as it always has and the
  // lines are there for whoever wants them.
  crowLinesLayer = L.layerGroup()
  positionsLayer = L.layerGroup()
  L.control
    .layers(
      baseLayers,
      { [CROW_LINES_LABEL]: crowLinesLayer, [POSITIONS_LABEL]: positionsLayer },
      { position: 'topright', collapsed: true }
    )
    .addTo(map)
  L.control.scale({ metric: true, imperial: false }).addTo(map)

  map.on('contextmenu', onMapContextMenu)
  map.on('click', hideContextMenu)
  map.on('click', onPickClick)
  map.on('mousemove', onPickMove)
  map.on('movestart', hideContextMenu)
  // Only the settled events: `move`/`zoom` fire continuously during a drag, and each one would set
  // refs and re-render the component for a readout nobody can read mid-gesture.
  map.on('zoomend moveend', readZoom)
  // Which dots overlap is a question about the screen, so the grouping is recomputed after a zoom.
  map.on('zoomend', drawPositions)
  readZoom()
  drawPositions()
  livenessTimer = window.setInterval(drawPositions, 60_000)
  document.addEventListener('click', onDocumentClick)

  // The map exists now, so let payloads through. If the data was already cached this renders it in
  // the same tick, with no request — see applyPaused for why readiness is a condition rather than a
  // call.
  mapReady.value = true
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
  window.clearInterval(livenessTimer)
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<template>
  <!-- The zoomed-out modifier lives here, on an element Vue owns exclusively — never on the map
       container. Vue patches a class binding with `el.className = value`, which *replaces* the
       attribute, and Leaflet keeps its own state in classes on that container (`leaflet-container`,
       `leaflet-grab`, `leaflet-fade-anim`, `leaflet-zoom-anim` during an animation). Binding a class
       there wiped all of them the moment the zoom crossed the threshold, and the map lost the CSS
       that positions and reveals its tiles: a blank map from zoom 12 up. -->
  <div class="kort-wrapper" :class="{ 'kort--far': farOut }">
    <div ref="mapContainer" class="kort-map" />

    <!-- Live updates are paused while there is unsaved work. The page has taught its operator to
         trust that it is current, so it owes them a word the one time it deliberately is not. -->
    <div v-if="updatesWaiting" class="kort-paused">
      <i class="pi pi-pause-circle" />
      <span>Opdateringer sat på pause — der er ugemte ændringer</span>
    </div>

    <!-- Edit-mode toolbar -->
    <div class="edit-toolbar">
      <template v-if="!editMode">
        <button class="edit-btn edit-btn--enter" :disabled="settingsOpen" @click="enterEditMode">
          <i class="pi pi-pencil" />
          <span>Redigér</span>
        </button>
        <!-- The sheets we print (PRD 010). Beside the edit button because both are about the same
             map; disabled while editing positions, since the two cannot both own the markers. -->
        <button class="edit-btn edit-btn--enter" :disabled="editMode" @click="enterSettings">
          <i class="pi pi-cog" />
          <span>Kort</span>
        </button>
      </template>
      <template v-else>
        <div class="edit-toolbar__active">
          <div class="edit-toolbar__indicator">
            <span class="edit-toolbar__dot" />
            <span>Redigeringstilstand</span>
            <span v-if="dirtyCheckpoints.size > 0" class="edit-toolbar__badge"> {{ dirtyCheckpoints.size }} ændring{{ dirtyCheckpoints.size === 1 ? '' : 'er' }} </span>
          </div>
          <div class="edit-toolbar__hint">Højreklik på kortet for at placere checkpoints. Træk markører for at flytte dem.</div>

          <!-- Route distances -->
          <div v-if="routeDistances.shortest != null && routeDistances.longest != null" class="edit-toolbar__distances">
            <div class="edit-toolbar__distance-row">
              <span class="edit-toolbar__distance-label">
                <i class="pi pi-arrows-h" style="font-size: 11px" />
                Korteste rute
              </span>
              <span class="edit-toolbar__distance-value edit-toolbar__distance-value--short"> {{ routeDistances.shortest.toFixed(1) }} km </span>
            </div>
            <div class="edit-toolbar__distance-row">
              <span class="edit-toolbar__distance-label">
                <i class="pi pi-arrows-h" style="font-size: 11px" />
                Længste rute
              </span>
              <span class="edit-toolbar__distance-value edit-toolbar__distance-value--long"> {{ routeDistances.longest.toFixed(1) }} km </span>
            </div>
          </div>

          <div class="edit-toolbar__actions">
            <button class="edit-btn edit-btn--cancel" :disabled="saving" @click="cancelEditMode">
              <i class="pi pi-times" />
              <span>Annullér</span>
            </button>
            <button class="edit-btn edit-btn--save" :disabled="saving || dirtyCheckpoints.size === 0" @click="saveChanges">
              <i class="pi pi-check" />
              <span>{{ saving ? 'Gemmer…' : 'Gem ændringer' }}</span>
            </button>
          </div>
        </div>
      </template>
    </div>

    <!-- Context menu -->
    <div v-if="menuVisible" ref="contextMenu" class="context-menu" :style="{ left: menuX + 'px', top: menuY + 'px' }" @click.stop>
      <div class="context-menu-header">Placér checkpoint</div>
      <template v-for="cg in checkgroups" :key="cg.id">
        <div v-if="cg.checkpoints.length" class="context-menu-group">{{ cg.name }}</div>
        <button v-for="cp in cg.checkpoints" :key="cp.id" class="context-menu-item" @click="pickCheckpoint(cp)">
          <i class="pi pi-map-marker" />
          <span>{{ cp.name }}</span>
          <span v-if="cp.latitude && cp.longitude" class="context-menu-badge">📍</span>
        </button>
      </template>
      <div v-if="checkgroups.length === 0" class="context-menu-empty">Ingen checkpoints fundet</div>
    </div>

    <!-- The printed sheets (PRD 010). Not modal: the map behind it is the feedback surface, and
         selecting a sheet fades every checkpoint that is not on it. -->
    <KortSettingsDialog
      v-model:visible="settingsOpen"
      v-model:selectedId="selectedKortId"
      :payload="kortSheets"
      :checkgroups="checkgroups"
      :pick="pick"
      :extent-edit="extentEdit"
      @saved="refreshKort"
      @update:dirty="settingsDirty = $event"
      @update:picking="pickingExtent = $event"
      @update:extentsPreview="extentsPreview = $event"
      @update:overlay="overlay = $event"
      @update:hoverExtents="hoverExtents = $event"
    />
  </div>
</template>

<style scoped>
.kort-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
}

.kort-map {
  width: 100%;
  height: 100%;
}

/* ---- Edit toolbar ---- */
.kort-paused {
  position: absolute;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  background: #fef3c7;
  border: 1px solid #fcd34d;
  color: #92400e;
  font-size: 12px;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.1);
}

.edit-toolbar {
  position: absolute;
  top: 10px;
  right: 56px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  pointer-events: none;
}

.edit-toolbar > * {
  pointer-events: auto;
}

.edit-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition:
    background-color 0.15s,
    box-shadow 0.15s,
    opacity 0.15s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  white-space: nowrap;
}

.edit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.edit-btn .pi {
  font-size: 12px;
}

.edit-btn--enter {
  background: #fff;
  color: #333;
}

.edit-btn--enter:hover {
  background: #f0f4f8;
}

.edit-btn--cancel {
  background: #fff;
  color: #666;
}

.edit-btn--cancel:hover:not(:disabled) {
  background: #f5f5f5;
}

.edit-btn--save {
  background: #2563eb;
  color: #fff;
}

.edit-btn--save:hover:not(:disabled) {
  background: #1d4ed8;
}

.edit-toolbar__active {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.25);
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 260px;
}

.edit-toolbar__indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #333;
}

.edit-toolbar__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
  animation: pulse-dot 1.5s ease-in-out infinite;
  flex-shrink: 0;
}

@keyframes pulse-dot {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

.edit-toolbar__badge {
  font-size: 11px;
  font-weight: 600;
  background: #dbeafe;
  color: #1d4ed8;
  padding: 2px 8px;
  border-radius: 10px;
  margin-left: auto;
}

.edit-toolbar__hint {
  font-size: 11px;
  color: #888;
  line-height: 1.4;
}

/* ---- Route distances ---- */
.edit-toolbar__distances {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 0 4px;
  border-top: 1px solid #eee;
}

.edit-toolbar__distance-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.edit-toolbar__distance-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #555;
}

.edit-toolbar__distance-value {
  font-size: 13px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.edit-toolbar__distance-value--short {
  color: #16a34a;
}

.edit-toolbar__distance-value--long {
  color: #dc2626;
}

.edit-toolbar__actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding-top: 4px;
  border-top: 1px solid #eee;
}

/* ---- Context menu ---- */
.context-menu {
  position: fixed;
  z-index: 10000;
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
  min-width: 220px;
  max-height: 400px;
  overflow-y: auto;
  padding: 4px 0;
  font-size: 14px;
}

.context-menu-header {
  padding: 8px 14px 4px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  color: #999;
  letter-spacing: 0.05em;
}

.context-menu-group {
  padding: 6px 14px 2px;
  font-size: 12px;
  font-weight: 600;
  color: #445e65;
  border-top: 1px solid #eee;
  margin-top: 2px;
}

.context-menu-group:first-of-type {
  border-top: none;
  margin-top: 0;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 14px 6px 24px;
  border: none;
  background: none;
  cursor: pointer;
  text-align: left;
  font-size: 14px;
  color: #333;
  transition: background-color 0.1s;
}

.context-menu-item:hover {
  background-color: #f0f4f8;
}

.context-menu-item .pi {
  color: #888;
  font-size: 12px;
}

.context-menu-badge {
  margin-left: auto;
  font-size: 12px;
}

.context-menu-empty {
  padding: 12px 14px;
  color: #999;
  font-style: italic;
}
</style>

<style>
/* Style the layer control */
.kort-map .leaflet-control-layers {
  border: none;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

/* Ensure distance label div-icons don't interfere */
.distance-label {
  background: none !important;
  border: none !important;
  box-shadow: none !important;
}

/* The centre of a checkpoint's 500 m circle.

   Global rather than scoped: Leaflet builds these elements itself, so they never receive the
   component's scope attribute and a scoped rule would not match them.

   The icon box is 18px so there is something to grab when dragging a position in edit mode, while
   the dot itself is 8px — the visible mark has to be small, because it stands for a point, and the
   circle around it is what carries the meaning. */
.cp-centre {
  background: none;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cp-centre span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #dc2626;
  /* A white ring, so the dot stays visible where several circles overlap into solid red. */
  box-shadow: 0 0 0 2px #fff;
}

/* A patrol's last known position: filled when live (under five minutes), hollow when it is only the
   last thing we heard. Green in both cases, and never red — a stale position means "we do not know",
   not "something is wrong", and a phone in a pocket looks exactly like a phone in a lake.

   Global, like the other Leaflet-built icons. */
.pos-dot {
  background: none;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pos-dot span {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  border-radius: 50%;
  border: 2px solid #16a34a;
  /* Hollow by default: the common case during a race is a position from a while ago, and the filled
     dot has to be the one that stands out. */
  background: rgba(255, 255, 255, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #14532d;
  line-height: 1;
}

.pos-dot--live span {
  background: #16a34a;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.9);
}

/* A group carries its count, so three is distinguishable from thirty — which size alone cannot do. */
.pos-dot--group span {
  border-width: 3px;
}

/* An area's corner grips and its move cursor, while a sheet is open in the dialog. Global for the same
   reason as .cp-centre: Leaflet builds these elements, so they never carry the scope attribute. */
.extent-handle {
  background: none;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.extent-handle span {
  width: 10px;
  height: 10px;
  background: #fff;
  border: 2px solid #2563eb;
  border-radius: 2px;
}

/* Square grips with per-corner cursors: the cursor is what tells an operator this drags the edge
   rather than the whole thing. */
.extent-handle--nw,
.extent-handle--se {
  cursor: nwse-resize;
}

.extent-handle--ne,
.extent-handle--sw {
  cursor: nesw-resize;
}

.extent-body {
  cursor: move;
}

/* The Fugleflugslinjer overlay: slightly thicker from CIRCLE_ZOOM_MIN up, where there is room for a
   line to read as a line, and a hairline below that, where a hundred of them would turn the map red.
   CSS overrides Leaflet's stroke-width presentation attribute, so the zoom needs no redraw. */
.crow-line {
  stroke-width: 2;
}

.kort--far .crow-line {
  stroke-width: 1;
}

/* Zoomed out past CIRCLE_ZOOM_MIN: the 500 m circle would be no wider than the dot inside it, so it
   is hidden and the centre becomes a fixed-size ring — a checkpoint you can see and click, making no
   claim about the ground it covers.

   Keyed off the wrapper, not the map container: see the note in the template for why a class binding
   must never be put on an element Leaflet also writes classes to. */
.kort--far .cp-circle {
  display: none;
}

.kort--far .cp-centre span {
  width: 12px;
  height: 12px;
  background: rgba(220, 38, 38, 0.25);
  border: 2px solid #dc2626;
  box-shadow: 0 0 0 1px #fff;
}
</style>
