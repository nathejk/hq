import { describe, it, expect } from 'vitest'
import {
  clusterPositions,
  isLive,
  simulatePositions,
  LIVE_WITHIN_MS,
  POSITION_DEPENDENCIES,
  type PatruljePosition,
} from './patruljePositions'

const NOW = Date.parse('2026-09-05T22:00:00Z')

const at = (teamId: string, latitude: number, longitude: number, ageMs = 0): PatruljePosition => ({
  teamId,
  teamNumber: teamId,
  name: teamId,
  latitude,
  longitude,
  ts: NOW - ageMs,
  source: 'track',
})

describe('POSITION_DEPENDENCIES', () => {
  // The tokens are the event subjects' entities, and a wrong one fails silently — the layer looks live
  // and never updates. `qr` in particular is not guessable from the UI's vocabulary: there is no
  // `scan` token, because scans arrive on NATHEJK.*.qr.*.scanned.
  it('depends on the event subjects, not the projections', () => {
    expect([...POSITION_DEPENDENCIES]).toEqual(['track', 'qr', 'patrulje'])
  })
})

describe('isLive', () => {
  it('is live just inside five minutes and not just outside', () => {
    expect(isLive(at('a', 55, 12, LIVE_WITHIN_MS - 1000), NOW)).toBe(true)
    expect(isLive(at('a', 55, 12, LIVE_WITHIN_MS + 1000), NOW)).toBe(false)
  })

  // A phone with a fast clock reports a position "in the future". It is certainly not stale.
  it('treats a future timestamp as live', () => {
    expect(isLive({ ts: NOW + 30_000 }, NOW)).toBe(true)
  })
})

describe('clusterPositions', () => {
  // A projection standing in for Leaflet's, with the race area's corner as its origin so pixel values
  // in these tests are small and exact: 0.0001° = 10 px. Anchoring it matters — with an unanchored
  // scale, test points land on arbitrary grid lines and a test can fail on the documented
  // cell-boundary artefact rather than on the behaviour it means to check.
  const project = (p: PatruljePosition) => ({ x: (p.longitude - 12) * 100000, y: (p.latitude - 55) * 100000 })

  it('leaves well-separated positions as single dots', () => {
    const clusters = clusterPositions([at('a', 55.0, 12.0), at('b', 55.5, 12.5)], project, 34, NOW)

    expect(clusters).toHaveLength(2)
    expect(clusters.every((c) => c.members.length === 1)).toBe(true)
  })

  it('groups positions that would overlap on screen', () => {
    const clusters = clusterPositions(
      [at('a', 55.0, 12.0), at('b', 55.0001, 12.0001), at('c', 55.0002, 12.0002)],
      project,
      34,
      NOW,
    )

    expect(clusters).toHaveLength(1)
    expect(clusters[0].members.map((m) => m.teamId).sort()).toEqual(['a', 'b', 'c'])
  })

  // Overlap is a property of the screen, not of the ground: the same two patrols must group when
  // zoomed out and separate when zoomed in. This is why the caller projects.
  it('groups or separates the same pair depending on the projection scale', () => {
    const pair = [at('a', 55.0, 12.0), at('b', 55.001, 12.001)]
    const zoomedOut = (p: PatruljePosition) => ({ x: (p.longitude - 12) * 10000, y: (p.latitude - 55) * 10000 })
    const zoomedIn = (p: PatruljePosition) => ({ x: (p.longitude - 12) * 1000000, y: (p.latitude - 55) * 1000000 })

    expect(clusterPositions(pair, zoomedOut, 34, NOW)).toHaveLength(1)
    expect(clusterPositions(pair, zoomedIn, 34, NOW)).toHaveLength(2)
  })

  it('draws a group at the mean of its members', () => {
    // Both inside one 34 px cell: 0.0002° is 20 px at this projection.
    const clusters = clusterPositions([at('a', 55.0, 12.0), at('b', 55.0002, 12.0002)], project, 34, NOW)

    expect(clusters).toHaveLength(1)
    expect(clusters[0].latitude).toBeCloseTo(55.0001, 6)
    expect(clusters[0].longitude).toBeCloseTo(12.0001, 6)
  })

  // "Any", not "all": the dot answers "is anything here current?". With "all", one stale patrol in a
  // group of ten would hollow out the dot and hide nine live ones.
  it('is live when any member is live', () => {
    const clusters = clusterPositions(
      [at('stale', 55.0, 12.0, 6 * 60 * 60 * 1000), at('fresh', 55.0001, 12.0001, 60_000)],
      project,
      34,
      NOW,
    )

    expect(clusters).toHaveLength(1)
    expect(clusters[0].live).toBe(true)
  })

  it('is not live when every member is stale', () => {
    const clusters = clusterPositions(
      [at('a', 55.0, 12.0, LIVE_WITHIN_MS + 1), at('b', 55.0001, 12.0001, 60 * 60 * 1000)],
      project,
      34,
      NOW,
    )

    expect(clusters[0].live).toBe(false)
  })

  it('lists a group newest first, so a long popup leads with what matters', () => {
    const clusters = clusterPositions(
      [
        at('old', 55.0, 12.0, 90_000),
        at('newest', 55.0001, 12.0001, 1_000),
        at('mid', 55.0002, 12.0002, 30_000),
      ],
      project,
      34,
      NOW,
    )

    expect(clusters[0].members.map((m) => m.teamId)).toEqual(['newest', 'mid', 'old'])
  })

  it('handles an empty input', () => {
    expect(clusterPositions([], project, 34, NOW)).toEqual([])
  })
})

describe('simulatePositions', () => {
  // Deterministic on purpose: a layout that reshuffles on every redraw makes it impossible to tell a
  // rendering bug from new random numbers.
  it('produces the same layout every time', () => {
    expect(simulatePositions(20, NOW)).toEqual(simulatePositions(20, NOW))
  })

  it('produces the requested number of distinct patruljer', () => {
    const positions = simulatePositions(200, NOW)

    expect(positions).toHaveLength(200)
    expect(new Set(positions.map((p) => p.teamId)).size).toBe(200)
  })

  // The point of the fixture is to exercise both dot styles and the clustering, so a run that happened
  // to be all-live or all-spread would be useless.
  it('mixes live and stale positions', () => {
    const positions = simulatePositions(200, NOW)
    const live = positions.filter((p) => isLive(p, NOW)).length

    expect(live).toBeGreaterThan(0)
    expect(live).toBeLessThan(200)
  })

  it('produces overlapping clumps, not just an even spread', () => {
    const positions = simulatePositions(200, NOW)
    const project = (p: PatruljePosition) => ({ x: p.longitude * 40000, y: p.latitude * 40000 })
    const clusters = clusterPositions(positions, project, 34, NOW)

    expect(clusters.length).toBeLessThan(positions.length)
    expect(clusters.some((c) => c.members.length > 2)).toBe(true)
  })
})
