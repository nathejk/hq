// Where every patrulje was last seen (PRD 011).
//
// # What this is, and what it very much is not
//
// A patrol's last *known* position, which is not the same as where the patrol is. It is the newest of
// two kinds of evidence, and they differ in nature:
//
//   - **scan** — certain but old. A QR scan happened at a post whose coordinates HQ knows, so the
//     patrol was definitely there; it may have walked 8 km since.
//   - **track** — recent but indirect. A member's phone reported a position, which is the patrol's
//     position only insofar as the patrol stays together and that member is carrying the phone.
//
// The API merges them and reports whichever is newer, with `source` saying which won. Nothing here
// interpolates, predicts or smooths: an operator reading this screen during a race must be able to
// tell "we know" from "we knew, two hours ago", and a dot that moved on its own would destroy that.
//
// Like `useKort` and `usePositionPresence`, this is the only way to read the resource: the
// useLiveResource cache is keyed by string, so inlining the fetch in a second place is two chances to
// write a different key or a different `dependsOn` — both silent failures.

import { http } from '@/plugins/axios'
import { useLiveResource } from './useLiveResource'

/** One patrol's last known position, as `GET /api/telemetry/positions` reports it. */
export interface PatruljePosition {
  teamId: string
  teamNumber: string
  name: string
  latitude: number
  longitude: number

  /**
   * Epoch **milliseconds**.
   *
   * The API converts scan timestamps (which are stored in *seconds*) on the way out, so both sources
   * share one axis and nothing downstream has to know which unit it started in.
   */
  ts: number

  /** Which evidence this position came from — see the note at the top of the file. */
  source: 'track' | 'scan'
}

/**
 * How fresh a position has to be to count as "live".
 *
 * Five minutes, which is deliberately much tighter than the 30 minutes `usePositionPresence` uses to
 * mute its glyph, because the two answer different questions. There, the question is "does this person
 * report positions at all?", and gaps of an hour are normal — phones lock and batteries die. Here the
 * question is "can I act on this dot right now?", and a position from twenty minutes ago is a patrol
 * that could be two kilometres away in any direction.
 *
 * So the filled dot means **"this is current"**, and the hollow one means **"this is the last thing we
 * heard"** — never "something is wrong". A phone in a pocket looks exactly like a phone in a lake.
 */
export const LIVE_WITHIN_MS = 5 * 60 * 1000

/** Whether a position is fresh enough to be drawn as a filled dot. */
export const isLive = (position: Pick<PatruljePosition, 'ts'>, now = Date.now()): boolean =>
  now - position.ts < LIVE_WITHIN_MS

/** The cache key. One string, in one place, because more than one component may read this. */
export const POSITIONS_KEY = 'patrulje:positions'

/**
 * What invalidates the positions.
 *
 * Entity **types**, not instances: a patrol whose first position has just arrived was never in the
 * payload, so an instance dependency could not bring it in.
 *
 * Every token is an event *subject's* entity, which is not the same as the projection's name:
 * telemetry arrives on `NATHEJK.*.track.*` (token `track`) and scans on `NATHEJK.*.qr.*.scanned`
 * (token `qr` — there is no `scan` token). `patrulje` is here because the payload joins in the team's
 * name and number, so a renamed patrol should redraw its popup.
 */
export const POSITION_DEPENDENCIES = ['track', 'qr', 'patrulje'] as const

export interface PositionsPayload {
  positions: PatruljePosition[]
}

/**
 * Read every patrulje's last known position.
 *
 * Renders from cache and revalidates in the background, so switching to the map shows the dots with no
 * request and no flash. `pending` is true only when nothing is cached at all.
 */
export function usePatruljePositions() {
  return useLiveResource<PositionsPayload>(
    POSITIONS_KEY,
    async () => {
      const response = await http.get('/telemetry/positions', { withCredentials: true })
      return { positions: (response.data?.positions ?? []) as PatruljePosition[] }
    },
    { dependsOn: [...POSITION_DEPENDENCIES] },
  )
}

// --- clustering ---
//
// ~200 patruljer on a map a few kilometres across means dots on top of dots, especially at the start
// and the finish where every patrol is in one field. Overlapping dots do not just look untidy: they
// *lie*, because five patrols in one spot look like one patrol.
//
// Hand-rolled rather than leaflet.markercluster. The requirement is one gesture ("how many are in
// there?") over a few hundred points in a small area, while markercluster brings its own icon set,
// stylesheet, spiderfy animation and click-to-zoom behaviour that would then have to be configured
// back out. The whole of it is the twenty lines below, it is a pure function, and it is therefore
// testable without a map.

/** A projected screen position, in pixels, at the current zoom. */
export interface PixelPoint {
  x: number
  y: number
}

/** One dot to draw: either a single patrol or a group of them. */
export interface PositionCluster {
  /** Where to draw it: the mean of its members' coordinates. */
  latitude: number
  longitude: number
  /** The patrols in this dot. Length 1 is a plain dot; more is a group. */
  members: PatruljePosition[]
  /** True when *any* member is live \u2014 see `clusterPositions` for why any and not all. */
  live: boolean
}

/**
 * Group positions that would overlap on screen into single dots.
 *
 * A fixed pixel grid, not a distance in metres: overlap is a property of the *screen*, so the same
 * two patrols must group when zoomed out and separate when zoomed in. That is why the caller passes a
 * projection — this function knows nothing about Leaflet, and the map does the projecting.
 *
 * A grid rather than proper agglomerative clustering. It is O(n), it is stable (the same input gives
 * the same output, so dots do not shuffle between redraws), and its one artefact — two dots either
 * side of a cell boundary staying separate — is harmless here: the question being asked is "is there
 * more than one patrol in this spot?", and a group of four drawn as three-and-one still answers it.
 * Proper clustering would move dots around as the set changes, which is worse on a screen an operator
 * is watching.
 *
 * `live` is true when *any* member is live, because the dot answers "is anything here current?". With
 * "all", one stale patrol in a cluster of ten would hollow out the whole group and hide nine live ones.
 */
export const clusterPositions = (
  positions: PatruljePosition[],
  project: (position: PatruljePosition) => PixelPoint,
  cellPx = 34,
  now = Date.now(),
): PositionCluster[] => {
  const cells = new Map<string, PatruljePosition[]>()

  for (const position of positions) {
    const point = project(position)
    const key = `${Math.floor(point.x / cellPx)}:${Math.floor(point.y / cellPx)}`
    const cell = cells.get(key)
    if (cell) cell.push(position)
    else cells.set(key, [position])
  }

  return [...cells.values()].map((members) => ({
    // The mean, so a group sits among its members rather than on whichever one happened to be first.
    latitude: members.reduce((sum, m) => sum + m.latitude, 0) / members.length,
    longitude: members.reduce((sum, m) => sum + m.longitude, 0) / members.length,
    // Newest first, so a popup listing a big group leads with the patrols worth reading about.
    members: [...members].sort((a, b) => b.ts - a.ts),
    live: members.some((m) => isLive(m, now)),
  }))
}

// --- simulation ---

/**
 * Fake positions for ~200 patruljer, for judging the map before a race exists.
 *
 * Dev-only and opt-in (`/kort?simulate=200`): there is no telemetry and there are no scans in a
 * development database, so the layer is empty exactly when its density and clustering are what need
 * looking at.
 *
 * Deterministic, from a seeded generator rather than `Math.random`: a layout that reshuffles on every
 * redraw makes it impossible to tell a rendering bug from new random numbers.
 *
 * The shape of the fake data is chosen to exercise the real problems rather than to look pretty:
 * clumps as well as spread, because patrols bunch at the start and the finish and that is where the
 * dots collide, and a mix of ages either side of the five-minute line so both dot styles appear.
 */
export const simulatePositions = (count = 200, now = Date.now()): PatruljePosition[] => {
  // Mulberry32: small, fast, and good enough for placing dots.
  let seed = 0x9e3779b9
  const random = () => {
    seed = (seed + 0x6d2b79f5) | 0
    let t = seed
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }

  // A few gathering points plus open country, so both the clustered and the sparse case are visible.
  const clumps = [
    { lat: 55.84, lng: 12.05 },
    { lat: 55.78, lng: 12.22 },
    { lat: 55.72, lng: 12.11 },
  ]

  return Array.from({ length: count }, (_, i) => {
    const inClump = random() < 0.55
    const base = clumps[Math.floor(random() * clumps.length)]
    const spread = inClump ? 0.004 : 0.09
    const ageMs = random() < 0.4 ? random() * LIVE_WITHIN_MS : random() * 6 * 60 * 60 * 1000

    return {
      teamId: `sim-${i + 1}`,
      teamNumber: String(1000 + i),
      name: `Simuleret patrulje ${i + 1}`,
      latitude: base.lat + (random() - 0.5) * spread,
      longitude: base.lng + (random() - 0.5) * spread * 1.8,
      ts: now - ageMs,
      source: random() < 0.5 ? 'track' : 'scan',
    }
  })
}
