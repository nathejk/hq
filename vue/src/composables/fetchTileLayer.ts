// Tile loading that survives a flaky tile server (task 161).
//
// # The problem
//
// The Dataforsyningen WMS drops the occasional request under the burst of ~30 tiles a pan fires off,
// and Leaflet has no retry: the tile stays an empty grey square until some later gesture happens to
// re-create it. Holes in the terrain are exactly the wrong failure for a map an operator is reading a
// route off.
//
// # Why this is a TileLayer subclass and not a `tileerror` handler
//
// The obvious fix — listen for `tileerror` and re-assign `img.src` — was tried and made things much
// worse: a blank map from zoom 12 up. Leaflet owns that img's lifecycle. Re-assigning `src` makes the
// load complete a second time, which re-enters `_tileReady`, which re-stamps `tile.loaded` and calls
// `setOpacity(el, 0)` to restart the fade-in. `_updateOpacity` then skips any tile that is no longer
// `current` — and a tile stops being current the moment the operator zooms — so the tile stays at
// opacity 0 for ever, including the previous level's tiles Leaflet retains to cover the new level.
//
// The rule that follows is the whole design of this file: **`done` is called exactly once per tile**,
// after we have either succeeded or given up. Leaflet sees one completion, its fade and prune
// bookkeeping is never re-entered, and retrying becomes invisible to it.
//
// # What this buys beyond retrying
//
// An `<img>` reports failure as a bare `error` event: no status, no reason, and — the case that
// matters most — **no event at all for a request that hangs**, until the browser's own multi-minute
// stall limit. Fetching the tile ourselves gives a status code and lets us impose a deadline, so a
// slow server is retried in seconds and a 404 (bad bbox, dead token) is not retried at all.
//
// The cost is that tiles must be CORS-readable. Dataforsyningen, Esri, OSM and OpenTopoMap all send
// permissive `Access-Control-Allow-Origin`, but it is a header we do not control, so a failure to read
// is treated as a network failure and retried like one rather than throwing.

import L from 'leaflet'
import {
  describeFailure,
  isRetryable,
  retryDelayMs,
  MAX_TILE_ATTEMPTS,
  TILE_TIMEOUT_MS,
  type TileFailure,
} from './tileRetryPolicy'

/** Per-tile bookkeeping, parked on the element because that is what Leaflet hands back to us. */
interface FetchedTile extends HTMLImageElement {
  _abort?: AbortController
  _objectUrl?: string
  _cancelled?: boolean
}

const releaseTile = (tile: FetchedTile) => {
  tile._cancelled = true
  tile._abort?.abort()
  if (tile._objectUrl) {
    // Without this, a long shift on /kort leaks every tile it ever scrolled past.
    URL.revokeObjectURL(tile._objectUrl)
    tile._objectUrl = undefined
  }
}

/**
 * One attempt. Resolves with a blob, or rejects with a `TileFailure`.
 *
 * `AbortSignal.timeout` is not used: it would be indistinguishable from our own abort on tile unload,
 * and the two must not be confused — one is worth retrying and the other means nobody is waiting any
 * more.
 */
const fetchTile = async (url: string, outer: AbortController): Promise<Blob> => {
  const controller = new AbortController()
  const onOuterAbort = () => controller.abort('unloaded')
  outer.signal.addEventListener('abort', onOuterAbort, { once: true })

  let timedOut = false
  const timer = window.setTimeout(() => {
    timedOut = true
    controller.abort('timeout')
  }, TILE_TIMEOUT_MS)

  try {
    const response = await fetch(url, { signal: controller.signal, credentials: 'omit' })
    if (!response.ok) throw { kind: 'http', status: response.status } satisfies TileFailure
    return await response.blob()
  } catch (error) {
    if (timedOut) throw { kind: 'timeout' } satisfies TileFailure
    if (outer.signal.aborted) throw { kind: 'network' } satisfies TileFailure
    // A thrown TileFailure passes through; anything else (TypeError from the network layer, a CORS
    // refusal, a DNS failure) is a network problem as far as a retry is concerned.
    if (error && typeof error === 'object' && 'kind' in error) throw error
    throw { kind: 'network' } satisfies TileFailure
  } finally {
    window.clearTimeout(timer)
    outer.signal.removeEventListener('abort', onOuterAbort)
  }
}

/**
 * The shared `createTile`, mixed into both the plain and the WMS layer.
 *
 * Written once as a mixin because `L.TileLayer.WMS` is a subclass of `L.TileLayer` with its own
 * `getTileUrl`, and everything here is expressed in terms of `getTileUrl` — so both layers get the
 * same loading behaviour and neither has a copy of it.
 */
const fetchTileMixin = {
  createTile(this: L.TileLayer, coords: L.Coords, done: L.DoneCallback): HTMLElement {
    const tile = document.createElement('img') as FetchedTile
    tile.alt = ''
    tile.setAttribute('role', 'presentation')

    const abort = new AbortController()
    tile._abort = abort

    // Called exactly once. This is the invariant the whole file exists to keep: two completions per
    // tile is what blanked the map when this was attempted as a `tileerror` retry.
    let settled = false
    const finish = (failure?: TileFailure) => {
      if (settled) return
      settled = true
      done(failure ? new Error(describeFailure(failure)) : undefined, tile)
    }

    const url = this.getTileUrl(coords)

    void (async () => {
      for (let attempt = 1; attempt <= MAX_TILE_ATTEMPTS; attempt += 1) {
        if (tile._cancelled) return
        try {
          const blob = await fetchTile(url, abort)
          if (tile._cancelled) return

          const objectUrl = URL.createObjectURL(blob)
          tile._objectUrl = objectUrl
          // `done` waits for the decode, so Leaflet's fade-in reveals a painted tile rather than an
          // empty box that fills in afterwards.
          tile.onload = () => finish()
          tile.onerror = () => finish({ kind: 'network' })
          tile.src = objectUrl
          return
        } catch (error) {
          const failure = (error ?? { kind: 'network' }) as TileFailure
          if (tile._cancelled) return
          if (attempt === MAX_TILE_ATTEMPTS || !isRetryable(failure)) {
            // One line per permanently failed tile: this is how a flaky provider or an expired token
            // gets noticed, and a grey square on its own says nothing.
            console.warn(`Tile failed (${describeFailure(failure)}) after ${attempt} attempt(s): ${url}`)
            finish(failure)
            return
          }
          await new Promise((resolve) => window.setTimeout(resolve, retryDelayMs(attempt + 1)))
        }
      }
    })()

    return tile
  },
}

// Leaflet's `extend` is untyped in @types/leaflet, so the constructors are asserted back into shape
// here. Contained to these four lines rather than leaking `any` into `createBaseLayers`.
const FetchTileLayer = L.TileLayer.extend(fetchTileMixin) as unknown as new (
  url: string,
  options?: L.TileLayerOptions,
) => L.TileLayer

const FetchTileLayerWMS = L.TileLayer.WMS.extend(fetchTileMixin) as unknown as new (
  url: string,
  options: L.WMSOptions,
) => L.TileLayer.WMS

/** `L.tileLayer`, with retrying. */
export const fetchTileLayer = (url: string, options?: L.TileLayerOptions): L.TileLayer => {
  const layer = new FetchTileLayer(url, options)
  // Both events mean "nobody is waiting for this tile any more": `tileunload` when it is pruned,
  // `tileabort` when Leaflet drops a stale-zoom tile mid-flight. Either way the fetch is cancelled and
  // the blob URL released.
  layer.on('tileunload tileabort', (e) => releaseTile((e as L.TileEvent).tile as FetchedTile))
  return layer
}

/** `L.tileLayer.wms`, with retrying. */
export const fetchTileLayerWms = (url: string, options: L.WMSOptions): L.TileLayer.WMS => {
  const layer = new FetchTileLayerWMS(url, options)
  layer.on('tileunload tileabort', (e) => releaseTile((e as L.TileEvent).tile as FetchedTile))
  return layer
}
