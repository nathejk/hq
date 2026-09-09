// When to retry a map tile, and how long to wait (task 161).
//
// Separate from `fetchTileLayer.ts`, which does the Leaflet part, for one reason: importing Leaflet
// needs a `window`, and the unit tests run in a plain node environment on purpose (see
// vitest.config.ts). The judgement in this file — which failures deserve another request — is exactly
// the part worth testing, so it lives where it can be.

/** Why a tile did not arrive. Data rather than an exception, so the decision below stays pure. */
export type TileFailure = { kind: 'timeout' } | { kind: 'network' } | { kind: 'http'; status: number }

/** Attempts in total, not retries after the first. Three is two retries. */
export const MAX_TILE_ATTEMPTS = 3

/**
 * How long one attempt may take before it counts as a timeout.
 *
 * Eight seconds: long enough that a slow-but-working WMS is not abandoned mid-render, short enough that
 * a hung request is replaced while the operator is still looking at the same part of the map.
 *
 * There is no timeout at all without this. A hung request fires no event on an `<img>` until the
 * browser's own multi-minute stall limit, which is why the earlier `tileerror`-based attempt could not
 * even detect this case, let alone recover from it.
 */
export const TILE_TIMEOUT_MS = 8000

/**
 * Whether this failure is worth another go.
 *
 * The distinction is *transient* versus *settled*. A timeout, a dropped connection, a 429 or a 5xx say
 * "not now"; a 400 or a 404 say "not ever" — a malformed bbox or a rejected token will fail identically
 * on the third attempt, and retrying only adds load to a server already refusing us.
 */
export const isRetryable = (failure: TileFailure): boolean => {
  switch (failure.kind) {
    case 'timeout':
    case 'network':
      return true
    case 'http':
      return failure.status === 429 || failure.status >= 500
  }
}

/**
 * Backoff before attempt `n`, 1-based on the attempt being scheduled: `retryDelayMs(2)` precedes the
 * second attempt.
 *
 * Doubling, from 300 ms. A server dropping requests because thirty arrived at once is not helped by
 * thirty immediate retries, and an operator would rather have the tile late than never.
 */
export const retryDelayMs = (attempt: number): number => 300 * 2 ** (attempt - 2)

/** A human-readable reason, for the one console warning a permanently failed tile produces. */
export const describeFailure = (failure: TileFailure): string =>
  failure.kind === 'http' ? `HTTP ${failure.status}` : failure.kind
