// Photographs of the patrols (foto's PRD 001, wired into hq).
//
// # Where the pictures come from
//
// The camera crew's photographs are ingested by the foto service, which stores the
// bytes and publishes `NATHEJK.<year>.patrulje.<teamId>.photographed`. hq projects
// those events, so this API is hq's, but every `url` in the payload points at
// foto's content-addressed store. Those URLs are hashes of the bytes, so they are
// immutable and cached by the browser forever — which is why nothing here worries
// about revving an image.
//
// # Two resources, on purpose
//
// A list of ~200 patruljer shows one thumbnail per row, and a component that
// fetched its own team's photographs would turn one page into 200 requests. So
// `usePatruljeCovers` is a single request for the whole year, and the per-team
// resource is only ever loaded by the dialog that shows every picture — i.e. after
// somebody clicks.
//
// Both are live for free: the photograph and cover-choice events are on the
// *patrulje* subject, so the signal is `patrulje:<teamId>` and a new photograph
// invalidates exactly what an open patrol page already depends on.

import { computed } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from './useLiveResource'

/** One rendition: a cached smaller version of a photograph. */
export interface PhotoRendition {
  name: string
  url: string
  width: number
  height: number
  bytes: number
}

/** One photograph, as `GET /api/patrulje/:id/photos` reports it. */
export interface PatruljePhoto {
  /** Content hash of the display image. Its identity, and what the picker sends back. */
  ref: string
  url: string
  /** The smallest rendition. Absent for a photograph recorded before renditions existed. */
  thumbUrl?: string
  width: number
  height: number
  /** The camera app's category — "start", "finish", … Free text. */
  type?: string
  /** The crew marked this one as needing a look. */
  attention?: boolean
  capturedAt?: string
  /** The one photograph that stands for the team: the choice, else the newest. */
  cover: boolean
  /** True only when an organizer actually picked it — see `cover`. */
  chosen: boolean
  renditions?: PhotoRendition[]
}

/** One photograph per team, for thumbnails in lists. */
export interface PatruljeCover {
  teamId: string
  ref: string
  url: string
  thumbUrl?: string
  width: number
  height: number
  /** How many photographs the team has, so a row can hint that there are more. */
  count: number
  chosen: boolean
}

/** The cache key for one team's photographs. One place, because two components read it. */
export const photosKey = (teamId: string) => `patrulje:photos:${teamId}`

/**
 * Every photograph of one patrulje, newest first.
 *
 * `dependsOn` is the team *instance*: a photograph event names the team, so unlike
 * members or orders this one really can be keyed, and another patrol being
 * photographed does not refetch this one.
 */
export function usePatruljePhotos(teamId: string) {
  return useLiveResource<PatruljePhoto[]>(
    photosKey(teamId),
    async () => {
      const response = await http.get(`/patrulje/${teamId}/photos`)
      return (response.data?.photos ?? []) as PatruljePhoto[]
    },
    { dependsOn: [`patrulje:${teamId}`] },
  )
}

export const COVERS_KEY = 'patrulje:covers'

/**
 * One photograph per team for the whole year.
 *
 * The dependency is the entity *type*, not the ids currently held: a team whose
 * first photograph has just arrived was not in the payload, so an instance
 * dependency could never bring it in.
 */
export function usePatruljeCovers() {
  const resource = useLiveResource<PatruljeCover[]>(
    COVERS_KEY,
    async () => {
      const response = await http.get('/patruljefotos')
      return (response.data?.covers ?? []) as PatruljeCover[]
    },
    { dependsOn: ['patrulje'] },
  )

  // Indexed once per change rather than scanned per row: a table body re-renders
  // often and a linear search over 200 covers in every cell adds up.
  const byTeam = computed(() => {
    const map = new Map<string, PatruljeCover>()
    for (const cover of resource.data.value ?? []) map.set(cover.teamId, cover)
    return map
  })

  return { ...resource, byTeam }
}

/**
 * Pick (or, with an empty ref, un-pick) the photograph that represents a patrulje.
 *
 * Nothing is written into the cache here. The choice is an event, so the
 * projection's live signal is what updates every open view — the per-team list, the
 * thumbnail in the patrol list, and any other page showing this team — and doing it
 * that way means the screen shows what was actually recorded rather than what this
 * tab hoped for.
 */
export async function selectCoverPhoto(teamId: string, ref: string): Promise<void> {
  await http.put(`/patrulje/${teamId}/photos/cover`, { ref })
}

/**
 * The best URL for showing a photograph at roughly `targetWidth` CSS pixels.
 *
 * The display image is ~2000px wide, which is a wasteful 500 KB for a 64px
 * thumbnail. Each rendition carries its own dimensions precisely so a client can
 * choose without downloading one to measure it; the smallest rendition at least as
 * wide as the target wins, and the display image is the fallback when none is.
 */
export function photoUrlFor(
  photo: Pick<PatruljePhoto, 'url' | 'renditions'>,
  targetWidth: number,
): string {
  const candidates = (photo.renditions ?? [])
    .filter((r) => r.url && r.width >= targetWidth)
    .sort((a, b) => a.width - b.width)
  return candidates[0]?.url ?? photo.url
}
