<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useDeferredApply } from '@/composables/useDeferredApply'
import { useKort, extentFromCorners, type Extent } from '@/composables/kort'
import KortSettingsDialog from '@/components/kort/KortSettingsDialog.vue'

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
  overlay.value = null
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
 * Fade every marker not on the selected sheet.
 *
 * Applied by walking the existing markers rather than by rebuilding them: a rebuild would drop the
 * popups and, in edit mode, the drag handlers. Opacity rather than removal, because "this
 * checkpoint is not on this sheet" is exactly the mistake an operator is looking for, and a hidden
 * marker cannot be spotted.
 */
const applyHighlight = () => {
  const highlighted = highlightedCheckpoints.value
  const anySelected = highlighted.size > 0 || selectedKortId.value !== undefined
  markers.forEach((marker, checkpointId) => {
    if (!anySelected) {
      marker.setOpacity(1)
      return
    }
    marker.setOpacity(highlighted.has(checkpointId) ? 1 : 0.35)
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
/** A whole set's areas and the ground between them, for the seam check. */
const overlay = ref<{ extents: Extent[]; gaps: Extent[] } | null>(null)
/** True while the next two clicks are corners. */
const pickingExtent = ref(false)
/** The rectangle just drawn, handed to the dialog. `seq` so an identical redraw still registers. */
const pick = ref<{ extent: Extent; seq: number } | null>(null)
let pickSeq = 0
let firstCorner: L.LatLng | null = null

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
  extentLayer?.remove()
  extentLayer = L.layerGroup().addTo(map)

  const rect = (extent: Extent, options: L.PolylineOptions) =>
    L.rectangle(
      [
        [extent.northWest.latitude, extent.northWest.longitude],
        [extent.southEast.latitude, extent.southEast.longitude]
      ],
      { ...options, interactive: false }
    ).addTo(extentLayer!)

  // The seam check first, so the selected sheet draws over it.
  //
  // The set's own areas are outlined without a fill: with a dozen translucent fills stacked, the
  // overlaps read as darker patches and an operator starts seeing "coverage" where there is only
  // paint. The gaps are the only thing filled, because the gaps are the answer.
  overlay.value?.extents.forEach((extent) => rect(extent, { color: '#2563eb', weight: 1, fill: false, dashArray: '3 4' }))
  overlay.value?.gaps.forEach((extent) => rect(extent, { color: '#dc2626', weight: 1, fillColor: '#dc2626', fillOpacity: 0.35 }))

  extentsPreview.value.forEach((extent) => {
    // Translucent and thin: the rectangle describes the sheet, it is not the subject of the
    // screen, and a solid fill would hide the checkpoints it is drawn around.
    rect(extent, { color: '#2563eb', weight: 2, fillOpacity: 0.08 })
  })
}

watch([extentsPreview, overlay], () => drawExtents(), { deep: true })

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
const baseLayers: Record<string, L.TileLayer | L.TileLayer.WMS> = {
  'Topografisk 1:25.000': L.tileLayer.wms('https://api.dataforsyningen.dk/dtk_25_DAF', {
    layers: 'DTK25',
    format: 'image/png',
    transparent: true,
    attribution: '&copy; <a target="_blank" href="https://download.kortforsyningen.dk/content/vilk%C3%A5r-og-betingelser">Styrelsen for Dataforsyning og Effektivisering</a>',
    // @ts-ignore – extra param passed as query string by Leaflet
    token: '0d5816d7e175e934301f0277686c43f8',
    maxZoom: 19
  } as L.WMSOptions),
  Luftfoto: L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
    attribution: '&copy; Esri &mdash; Sources: Esri, DigitalGlobe, Earthstar Geographics',
    maxZoom: 19
  }),
  OpenStreetMap: L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    maxZoom: 19
  }),
  Topografisk: L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://opentopomap.org">OpenTopoMap</a> (<a href="https://creativecommons.org/licenses/by-sa/3.0/">CC-BY-SA</a>)',
    maxZoom: 17
  })
}

// ---------------------------------------------------------------------------
// Fix default Leaflet marker icons (webpack / vite asset issue)
// ---------------------------------------------------------------------------
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

        // Move marker back or remove if it was newly placed
        if (markers.has(cp.id)) {
          if (orig.latitude && orig.longitude) {
            markers.get(cp.id)!.setLatLng([orig.latitude, orig.longitude])
          } else {
            // It was placed freshly during edit — remove the marker
            markers.get(cp.id)!.remove()
            markers.delete(cp.id)
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

  // Nothing to call here: leaving edit mode flips `applyPaused`, and useDeferredApply applies
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
}

// ---------------------------------------------------------------------------
// Marker helpers
// ---------------------------------------------------------------------------
const placeMarker = (cp: Checkpoint, latlng: [number, number] | L.LatLng) => {
  if (!map) return

  // Remove existing marker for this checkpoint
  if (markers.has(cp.id)) {
    markers.get(cp.id)!.remove()
  }

  const checkgroup = checkgroups.value.find((cg) => cg.id === cp.checkgroupId)
  const groupName = checkgroup?.name ?? ''

  const marker = L.marker(latlng, { draggable: editMode.value })
    .addTo(map)
    .bindPopup(`<strong>${cp.name}</strong><br><span style="color:#666">${groupName}</span>`)
    .bindTooltip(cp.name, { permanent: false, direction: 'top', offset: [0, -36] })

  marker.on('dragend', () => {
    if (!editMode.value) return
    const pos = marker.getLatLng()
    cp.latitude = pos.lat
    cp.longitude = pos.lng
    markDirty(cp)
    console.log(`Moved ${cp.name} to ${pos.lat.toFixed(6)}, ${pos.lng.toFixed(6)}`)
  })

  markers.set(cp.id, marker)

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
// Lifecycle
// ---------------------------------------------------------------------------
onMounted(async () => {
  if (!mapContainer.value) return

  map = L.map(mapContainer.value, {
    center: [55.7, 12.1],
    zoom: 11,
    zoomControl: false,
    layers: [baseLayers['Topografisk 1:25.000']]
  })

  L.control.zoom({ position: 'topright' }).addTo(map)
  L.control.layers(baseLayers, {}, { position: 'topright', collapsed: true }).addTo(map)
  L.control.scale({ metric: true, imperial: false }).addTo(map)

  map.on('contextmenu', onMapContextMenu)
  map.on('click', hideContextMenu)
  map.on('click', onPickClick)
  map.on('mousemove', onPickMove)
  map.on('movestart', hideContextMenu)
  document.addEventListener('click', onDocumentClick)

  // The map exists now, so let payloads through. If the data was already cached this renders it in
  // the same tick, with no request — see applyPaused for why readiness is a condition rather than a
  // call.
  mapReady.value = true
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<template>
  <div class="kort-wrapper">
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
      @saved="refreshKort"
      @update:dirty="settingsDirty = $event"
      @update:picking="pickingExtent = $event"
      @update:extentsPreview="extentsPreview = $event"
      @update:overlay="overlay = $event"
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
</style>
