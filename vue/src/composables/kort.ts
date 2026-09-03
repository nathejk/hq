// Shared bits of the kort UI (PRD 010): the types the API sends, the Danish vocabulary, and the
// one definition of how the maps are loaded.
//
// # Why loading lives here and not in each component
//
// `DispatchView` calls `useLiveResource` inline, and that is fine for a resource one view owns.
// The maps are read by two components — the map view and its settings dialog — and the cache is
// keyed by string, so inlining twice means two chances to write a different key or a different
// `dependsOn`. Both failures are silent in the worst way: a mismatched key gives each component
// its own copy of the data that the other's writes never touch, and a missing dependency token
// gives a page that looks live and simply never updates (`.rules` → Live updates).
//
// So `useKort()` is the only way to read them. It is not a store and holds no state of its own —
// the value lives in useLiveResource's module cache, which is already shared. Duplicating it in
// Pinia is how the legacy dims channel ended up with two read models (PRD 004 §2.1).

import { http } from '@/plugins/axios'
import { useLiveResource } from './useLiveResource'

// --- what the API sends ---

/** a4 | a3 | skitse | andet. Danish labels below; the values are the domain's. */
export type Format = 'a4' | 'a3' | 'skitse' | 'andet'

/**
 * A team type a set may be marked for.
 *
 * `patrulje` and `klan` are the two that mean anything for maps. Never compare a set's name to
 * find the patrol sheets — names are free text an organizer may rename mid-season.
 */
export type TeamType = 'patrulje' | 'klan' | 'crew' | 'gøgler'

export interface Position {
  latitude: number
  longitude: number
}

/**
 * One rectangle of ground a sheet shows.
 *
 * Always a true north-west/south-east pair: the API normalises whichever two corners were
 * clicked, so nothing here has to do a min/max dance before handing it to Leaflet.
 */
export interface Extent {
  northWest: Position
  southEast: Position
}

/**
 * One printed sheet — one thing handed to a team, one QR code, one reveal.
 *
 * A double-sided A3 is *one* Kort with two `extents`, not two. The two extents are simply two
 * areas: nothing says which is the front, and the checkpoints are not split per side, because
 * both sides are handed over at once.
 */
export interface Kort {
  id: string
  kortsaetId: string
  year: string
  version: number
  name: string
  format: Format | ''
  note: string
  sortOrder: number

  /**
   * The checkpoints drawn on this sheet.
   *
   * Ids that no longer resolve are filtered out by the API, so this list is always live — which
   * is also how deleting a whole checkgroup reaches the maps.
   */
  checkpointIds: string[]

  /** Zero rectangles for a skitse, one for a normal sheet, two for a double-sided one. */
  extents: Extent[]
}

/**
 * A set of sheets — most often "Patruljer" and one for everybody else.
 *
 * `teamType` is nullable, and null is the *ordinary* case: it means the set is not for one
 * specific team type, which is true of the crew set that klaner also draw from. It is also not
 * unique, so more than one set may be marked `patrulje`. Read it as a filter, never as a key.
 */
export interface Kortsaet {
  id: string
  year: string
  version: number
  name: string
  sortOrder: number
  teamType: TeamType | null
  kort: Kort[]
}

export interface KortPayload {
  kortsaet: Kortsaet[]
  /** Sheets whose set is unknown. Normally empty; present so a mis-assigned sheet is visible. */
  orphanKort: Kort[]
}

// --- loading ---

/**
 * The cache key. One string, in one place, because two components share this entry.
 */
export const KORT_KEY = 'kort:all'

/**
 * What invalidates the maps.
 *
 * Entity *types*, not instances, because this is a list: a newly created sheet has an id no
 * client has seen, so an instance dependency could never bring it in.
 *
 * `checkpoint` and `checkgroup` are here because the payload's usefulness depends on them, not
 * because the maps changed: the picker shows checkpoints grouped by checkgroup, so a renamed
 * checkpoint or a regrouped one must redraw. `kortsaet` is separate from `kort` since a set can
 * be renamed or re-marked without any sheet changing.
 *
 * Every token is one the API advertises — verified against the boot log rather than assumed,
 * because a token nothing can emit is invisible except for a page that never updates.
 */
export const KORT_DEPENDENCIES = ['kort', 'kortsaet', 'checkpoint', 'checkgroup'] as const

/**
 * Read the year's map sets and their sheets.
 *
 * Renders from cache and revalidates in the background, so navigating back to `/kort` shows the
 * maps with no request and no flash. `pending` is true only when there is nothing cached at all —
 * wire it to a table's `:loading` and do not add a spinner of your own.
 */
export function useKort() {
  return useLiveResource<KortPayload>(
    KORT_KEY,
    async () => {
      const response = await http.get('/kort', { withCredentials: true })
      return {
        kortsaet: (response.data?.kortsaet ?? []) as Kortsaet[],
        orphanKort: (response.data?.orphanKort ?? []) as Kort[],
      }
    },
    { dependsOn: [...KORT_DEPENDENCIES] },
  )
}

// --- the Danish vocabulary ---

const formatLabels: Record<string, string> = {
  a4: 'A4',
  a3: 'A3',
  skitse: 'Skitse',
  andet: 'Andet',
}

/**
 * Label a format, falling back to the raw value.
 *
 * A fallback rather than a blank, so a value this build does not know about still reads as
 * *something* on screen. An unlabelled row looks like missing data.
 */
export const formatLabel = (format?: string) => (format ? (formatLabels[format] ?? format) : '')

export const formatOptions = (Object.keys(formatLabels) as Format[]).map((value) => ({
  value,
  label: formatLabels[value],
}))

const teamTypeLabels: Record<string, string> = {
  patrulje: 'Patruljer',
  klan: 'Klaner',
  crew: 'Crew',
  gøgler: 'Gøglere',
}

export const teamTypeLabel = (teamType?: string | null) =>
  teamType ? (teamTypeLabels[teamType] ?? teamType) : ''

/**
 * The choices for a set's team type, empty first.
 *
 * The empty option is not a placeholder — it is the crew set, and the commonest answer. Labelled
 * so it reads as a deliberate choice rather than an unfilled field.
 */
export const teamTypeOptions = [
  { value: null, label: 'Ingen bestemt holdtype' },
  ...(Object.keys(teamTypeLabels) as TeamType[]).map((value) => ({
    value,
    label: teamTypeLabels[value],
  })),
]

// --- derived questions the UI asks ---

/** Every sheet in the payload, sets and orphans together. */
export const allMaps = (payload?: KortPayload): Kort[] => {
  if (!payload) return []
  return [...payload.kortsaet.flatMap((set) => set.kort), ...payload.orphanKort]
}

/**
 * Checkpoints that are on no sheet of a given set.
 *
 * Per set, not overall: a checkpoint being absent from the crew maps is a different mistake from
 * its being absent from the patrol maps, and only the set's own list can tell an operator which
 * one they are looking at.
 */
export const checkpointsWithoutMap = (set: Kortsaet | undefined, checkpointIds: string[]): string[] => {
  if (!set) return [...checkpointIds]
  const covered = new Set(set.kort.flatMap((sheet) => sheet.checkpointIds))
  return checkpointIds.filter((id) => !covered.has(id))
}

/**
 * Whether any single sheet in the set shows all of the given checkpoints.
 *
 * The test behind the split-checkgroup warning, and deliberately **existential**: a checkgroup is
 * revealed as a whole, so what matters is that *some* sheet covers it. Two overlapping sheets
 * that both cover it are fine — adjacent sheets overlap by design, so a partitioning test would
 * fire constantly and be ignored.
 *
 * It is about sheet membership, never geometry: a checkgroup's checkpoints may legitimately sit in
 * two different areas of the same double-sided sheet.
 */
export const someMapContainsAll = (set: Kortsaet, checkpointIds: string[]): boolean => {
  if (checkpointIds.length === 0) return true
  return set.kort.some((sheet) => {
    const on = new Set(sheet.checkpointIds)
    return checkpointIds.every((id) => on.has(id))
  })
}

// --- extents ---

/**
 * Build a north-west/south-east pair from two arbitrary corners.
 *
 * The API normalises on save as well, and this still does it: the *preview* has to be a well-formed
 * rectangle before anything is saved, or Leaflet would happily draw an inverted one and the shape
 * would change under the operator when they pressed Gem.
 *
 * North is the larger latitude and west the smaller longitude — true in Denmark and everywhere else
 * east of Greenwich and north of the equator, which is the only place this event happens.
 */
export const extentFromCorners = (
  a: { lat: number; lng: number },
  b: { lat: number; lng: number },
): Extent => ({
  northWest: { latitude: Math.max(a.lat, b.lat), longitude: Math.min(a.lng, b.lng) },
  southEast: { latitude: Math.min(a.lat, b.lat), longitude: Math.max(a.lng, b.lng) },
})

/** Whether two rectangles are the same, corner for corner. */
export const sameExtent = (a: Extent, b: Extent): boolean =>
  a.northWest.latitude === b.northWest.latitude &&
  a.northWest.longitude === b.northWest.longitude &&
  a.southEast.latitude === b.southEast.latitude &&
  a.southEast.longitude === b.southEast.longitude

/**
 * Whether a rectangle has no area.
 *
 * Refused by the API, and worth knowing here so the operator is told before a round trip: two
 * clicks on the same latitude or longitude draw nothing, and a saved invisible extent reads as a
 * save that failed.
 */
export const isDegenerate = (extent: Extent): boolean =>
  extent.northWest.latitude === extent.southEast.latitude ||
  extent.northWest.longitude === extent.southEast.longitude

// --- the checkpoint picker's rules ---

/** Minimal shape the picker needs of a checkgroup. */
export interface CheckgroupLike {
  id: string
  name: string
  checkpoints: { id: string }[]
}

/**
 * Whether a checkgroup is fully, partly or not at all on the sheet.
 *
 * `some` exists to be visible: a half-ticked checkgroup is usually a mistake in the making, since a
 * checkgroup is revealed as a whole, so the header shows a third state rather than rounding to on
 * or off.
 */
export const groupSelectionState = (group: CheckgroupLike, picked: Set<string>): 'all' | 'some' | 'none' => {
  if (group.checkpoints.length === 0) return 'none'
  const on = group.checkpoints.filter((cp) => picked.has(cp.id)).length
  if (on === 0) return 'none'
  return on === group.checkpoints.length ? 'all' : 'some'
}

/**
 * Apply the select-all for a checkgroup.
 *
 * Ticks all when any are missing rather than inverting each one: with three of four already on, an
 * operator reaching for the group header means "all of them", never "swap them". Only a fully
 * ticked group clears.
 */
export const toggleGroupSelection = (group: CheckgroupLike, picked: Set<string>): Set<string> => {
  const next = new Set(picked)
  if (groupSelectionState(group, picked) === 'all') {
    group.checkpoints.forEach((cp) => next.delete(cp.id))
  } else {
    group.checkpoints.forEach((cp) => next.add(cp.id))
  }
  return next
}

/**
 * The selection as a list, in checkgroup order.
 *
 * Order carries no meaning — a sheet's checkpoints are a set — but a *stable* order does: the API
* compares the submitted list against the stored one to decide whether anything changed, so ticking
 * A then B and ticking B then A must produce the same list, or every re-save would look like an
 * edit and emit a live signal.
 *
 * Picked ids that are in no checkgroup are kept at the end rather than dropped. They can only come
 * from a checkpoint that vanished from the payload mid-edit, and silently discarding them would
 * make a save do something the operator did not ask for.
 */
export const orderPicks = (groups: CheckgroupLike[], picked: Set<string>): string[] => {
  const ordered: string[] = []
  const seen = new Set<string>()
  for (const group of groups) {
    for (const cp of group.checkpoints) {
      if (picked.has(cp.id) && !seen.has(cp.id)) {
        ordered.push(cp.id)
        seen.add(cp.id)
      }
    }
  }
  picked.forEach((id) => {
    if (!seen.has(id)) {
      ordered.push(id)
      seen.add(id)
    }
  })
  return ordered
}
