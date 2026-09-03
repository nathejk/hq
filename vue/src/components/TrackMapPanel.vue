<script setup lang="ts">
/**
 * The map and legend for one track target (PRD 011).
 *
 * # Why this is a separate component from the dialog
 *
 * `useLiveResource` is keyed by a string and must be called during setup: it registers an
 * `onScopeDispose` for reference counting, so calling it from inside a watcher would both escape the
 * component's effect scope and leak a watcher per target change. The key here is therefore fixed for
 * this component's lifetime, and the dialog remounts it with `:key` when the target or window changes.
 * That is the idiomatic way to make a keyed cache and a changing subject coexist.
 *
 * # One polyline per segment, never one per person
 *
 * The easiest thing in this feature to get wrong. Nobody records unbroken for a 30-hour race — phones
 * lock, apps are killed, batteries die — so a track is a handful of stretches separated by hours of
 * nothing. Joining them would draw a straight line across three hours of silence and present it as a
 * walked route, and an operator deciding where to send a car would believe it. The API returns segments
 * so this component *cannot* make that mistake; the connector between two segments is drawn
 * deliberately unlike a route — thin, dashed, dimmed — so "we do not know" is legible rather than
 * implied.
 *
 * # Scans are not track points
 *
 * A scan happened at a known post, at a known time, witnessed by a person. A track point is a phone's
 * best guess. They are drawn differently for that reason, and scans are never simplified.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import type { TrackTarget } from '@/composables/useTrackViewer'
import {
  createBaseLayers,
  DEFAULT_BASE_LAYER,
  RACE_AREA_CENTER,
  RACE_AREA_ZOOM,
} from '@/composables/mapLayers'
import { formatClock, formatRelative } from '@/composables/usePositionPresence'

export interface ApiPoint {
  ts: number
  lat: number
  lng: number
  accuracy: number
}

export interface ApiSegment {
  from: number
  to: number
  points: ApiPoint[]
}

export interface ApiTrack {
  personId: string
  personType: string
  name?: string
  coverage: {
    window: { from: number; to: number }
    recordedMs: number
    ratio: number
    points: number
  }
  segments: ApiSegment[]
  reduced: boolean
  maxPoints: number
  membershipFrom?: number
  membershipTo?: number | null
}

export interface ApiScan {
  qrId: string
  teamId: string
  teamNumber: number
  scannerId: string
  ts: number
  lat: string
  lng: string
}

const props = defineProps<{
  target: TrackTarget
  /** 0 means the whole race. */
  windowMs: number
}>()

// Captured once rather than computed: this component is remounted when either changes, so treating
// them as reactive would imply an update path that does not exist.
const target = props.target
const from = props.windowMs > 0 ? Date.now() - props.windowMs : 0
const query = from > 0 ? `?from=${from}` : ''

type Payload = { teamId: string; members: ApiTrack[]; scans: ApiScan[] }

/**
 * `dependsOn` carries the **type** token `track` as well as the instance.
 *
 * A point from a member this view has never seen carries an unknown personId, so an instance-only
 * dependency would miss exactly the newly-joined member somebody is looking for. `patrulje:{id}`
 * catches the roster changing, and `qr` — not `scan` — is the token scans are published under.
 */
const { data, pending } = useLiveResource<Payload>(
  target.kind === 'patrulje'
    ? `track:patrulje:${target.teamId}:${from}`
    : `track:person:${target.personId}:${from}`,
  async () => {
    if (target.kind === 'patrulje') {
      const response = await http.get(`/telemetry/patrulje/${target.teamId}/track${query}`)
      return response.data.patrulje as Payload
    }
    const response = await http.get(`/telemetry/person/${target.personId}/track${query}`)
    return { teamId: '', members: [response.data.track as ApiTrack], scans: [] }
  },
  {
    dependsOn:
      target.kind === 'patrulje'
        ? ['track', `patrulje:${target.teamId}`, 'qr']
        : ['track', `track:${target.personId}`],
  },
)

const members = computed(() => data.value?.members ?? [])
const scans = computed(() => data.value?.scans ?? [])
const hasAnyPoints = computed(() => members.value.some((m) => m.coverage.points > 0))
const isEmpty = computed(() => !hasAnyPoints.value && scans.value.length === 0)
const anyReduced = computed(() => members.value.some((m) => m.reduced))

/**
 * A fixed palette, assigned by position.
 *
 * Fixed rather than generated so a member keeps their colour when the window changes — a legend that
 * reshuffles as an operator switches preset is actively misleading. Eight is more than a patrol has
 * members; beyond that it wraps, which beats becoming unreadable.
 */
const palette = [
  '#2563eb',
  '#dc2626',
  '#16a34a',
  '#d97706',
  '#7c3aed',
  '#0891b2',
  '#db2777',
  '#65a30d',
]
const colourOf = (i: number) => palette[i % palette.length]

const mapContainer = ref<HTMLElement | null>(null)
let map: L.Map | null = null
let layer: L.LayerGroup | null = null

function draw() {
  if (!map || !layer) return

  // Cheap and idempotent, and needed more than once: the dialog animates in, so a size measured at
  // mount can be mid-transition, and Leaflet renders grey tiles until something tells it otherwise.
  map.invalidateSize()

  layer.clearLayers()

  const bounds: L.LatLngExpression[] = []

  members.value.forEach((member, i) => {
    const colour = colourOf(i)

    member.segments.forEach((segment, s) => {
      const latlngs = segment.points.map((p) => [p.lat, p.lng] as L.LatLngExpression)
      if (latlngs.length === 0) return
      bounds.push(...latlngs)

      // A single-point segment has no line to draw, so it gets a dot. Otherwise a lone fix would be
      // invisible — and a lone fix is sometimes the only thing we have.
      if (latlngs.length === 1) {
        L.circleMarker(latlngs[0], { radius: 4, color: colour, weight: 2, fillOpacity: 0.6 })
          .bindTooltip(`${label(member)} · ${formatClock(segment.from)}`)
          .addTo(layer!)
        return
      }

      L.polyline(latlngs, { color: colour, weight: 3, opacity: 0.85 })
        .bindTooltip(`${label(member)} · ${formatClock(segment.from)}–${formatClock(segment.to)}`)
        .addTo(layer!)

      // Where the data stops. Small, so it reads as a terminator rather than a place.
      L.circleMarker(latlngs[latlngs.length - 1], {
        radius: 3,
        color: colour,
        weight: 2,
        fillOpacity: 1,
      }).addTo(layer!)

      const next = member.segments[s + 1]
      if (next && next.points.length > 0) {
        const gapMinutes = Math.round((next.from - segment.to) / 60000)
        L.polyline([latlngs[latlngs.length - 1], [next.points[0].lat, next.points[0].lng]], {
          color: colour,
          weight: 1,
          opacity: 0.35,
          dashArray: '4 6',
        })
          .bindTooltip(`Ingen data i ${duration(next.from - segment.to)} (${gapMinutes} min)`)
          .addTo(layer!)
      }
    })
  })

  // Scans last so they sit above the lines: they are the certain positions, and an operator looks for
  // them first.
  scans.value.forEach((s) => {
    const lat = Number(s.lat)
    const lng = Number(s.lng)
    // Some scan rows have no coordinates at all — a post that registered a team without a GPS fix.
    // Skipped rather than drawn at (0,0), which is in the Atlantic and would wreck fitBounds.
    if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) return
    bounds.push([lat, lng])
    L.marker([lat, lng])
      .bindTooltip(`Scanning · ${formatClock(s.ts)}`)
      .addTo(layer!)
  })

  if (bounds.length > 0) {
    map.fitBounds(L.latLngBounds(bounds), { padding: [24, 24], maxZoom: 16 })
  }
}

function label(member: ApiTrack): string {
  return member.name || 'tidligere medlem'
}

/**
 * "2 positioner · 0 min data af 7 min" — what this member's phone actually gave us.
 *
 * The point count leads because coverage alone is misleading for sparse data: recorded time is
 * deliberately conservative and an isolated fix contributes **zero** — it evidences an instant, not an
 * interval — so "0 min data" next to a visible dot on the map reads as a contradiction unless the
 * number of positions is stated too. Both together say the true thing: we have two fixes and know
 * nothing about the time between them.
 */
function coverageText(member: ApiTrack): string {
  const points = member.coverage.points
  if (points === 0) return 'ingen positioner'

  const span = member.coverage.window.to - member.coverage.window.from
  const label = `${points} ${points === 1 ? 'position' : 'positioner'}`
  if (span <= 0) return label
  return `${label} · ${duration(member.coverage.recordedMs)} data af ${duration(span)}`
}

function duration(ms: number): string {
  const minutes = Math.round(ms / 60000)
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest === 0 ? `${hours} t` : `${hours} t ${rest} min`
}

function lastSeen(member: ApiTrack): string {
  const last = member.segments[member.segments.length - 1]
  if (!last) return ''
  return `${formatClock(last.to)} · ${formatRelative(last.to)}`
}

onMounted(() => {
  if (!mapContainer.value) return
  const baseLayers = createBaseLayers()
  map = L.map(mapContainer.value, {
    center: RACE_AREA_CENTER,
    zoom: RACE_AREA_ZOOM,
    zoomControl: false,
    layers: [baseLayers[DEFAULT_BASE_LAYER]],
  })
  L.control.zoom({ position: 'topright' }).addTo(map)
  L.control.layers(baseLayers, {}, { position: 'topright', collapsed: true }).addTo(map)
  L.control.scale({ metric: true, imperial: false }).addTo(map)
  layer = L.layerGroup().addTo(map)

  // Leaflet measures its container on creation, and inside a dialog that has just appeared the size
  // can still be zero — which leaves a grey box until the first pan.
  map.invalidateSize()
  draw()
})

onBeforeUnmount(() => {
  map?.remove()
  map = null
  layer = null
})

// Live updates land here: a new point arrives, the resource revalidates, and the map redraws.
watch([members, scans], () => draw())
</script>

<template>
  <div class="flex flex-col gap-3">
    <!--
      The notices sit *above* an always-rendered map, rather than replacing it.

      That is a bug fix, not a layout preference (task 155): the map container used to live inside a
      `v-else`, so on first mount — while `pending` was still true — the div did not exist, `onMounted`
      found a null ref and returned, and nothing ever created the map afterwards. The result was a white
      box with a correct legend beside it. Keeping the container unconditional means the ref exists for
      the whole component's life, which is the only state the map's lifecycle has to cope with.

      It also reads better empty: base tiles centred on the race area say "no data here" far more
      clearly than a blank rectangle does.
    -->
    <div v-if="pending && !data" class="text-sm text-gray-500">Henter spor…</div>

    <div v-else-if="isEmpty" class="text-sm text-gray-500">
      Ingen positioner og ingen scanninger i det valgte tidsrum.
    </div>

    <div
      v-else-if="!hasAnyPoints"
      class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded p-2"
    >
      Ingen positioner rapporteret — viser kun scanninger.
    </div>

    <!-- Said out loud rather than left for someone to infer from a thin-looking line. -->
    <div v-if="anyReduced" class="text-xs text-gray-500">
      Sporet er forenklet — vælg et kortere tidsrum for flere detaljer.
    </div>

    <div ref="mapContainer" class="track-map w-full rounded border border-gray-200"></div>

    <!--
      The legend is not decoration: it is where "this line stops because the scout left the patrol"
      is distinguished from "this line stops because the phone did", and where coverage says whether
      the picture is worth reasoning from at all.
    -->
    <div v-if="members.length" class="grid gap-1 text-xs sm:grid-cols-2">
      <div v-for="(m, i) in members" :key="m.personId" class="flex items-center gap-2 flex-wrap">
        <span
          class="inline-block w-3 h-1 rounded"
          :style="{ backgroundColor: colourOf(i) }"
          aria-hidden="true"
        />
        <span class="font-medium" :class="{ 'italic text-gray-500': !m.name }">
          {{ label(m) }}
        </span>
        <span v-if="m.membershipTo" class="text-gray-500">
          (forlod patruljen {{ formatClock(m.membershipTo) }})
        </span>
        <span class="text-gray-500">· {{ coverageText(m) }}</span>
        <span v-if="lastSeen(m)" class="text-gray-400">· sidst {{ lastSeen(m) }}</span>
      </div>
      <div v-if="scans.length" class="flex items-center gap-2">
        <i class="pi pi-map-marker text-gray-600" aria-hidden="true" />
        <span>{{ scans.length }} scanninger (nøjagtige tidspunkter og steder)</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.track-map {
  height: min(62vh, 620px);
}
</style>
