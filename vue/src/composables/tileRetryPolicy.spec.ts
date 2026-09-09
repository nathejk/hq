import { describe, it, expect } from 'vitest'
import {
  describeFailure,
  isRetryable,
  retryDelayMs,
  MAX_TILE_ATTEMPTS,
  TILE_TIMEOUT_MS,
  type TileFailure,
} from './tileRetryPolicy'

// The Leaflet glue is not unit-testable without a DOM and a map, but the *policy* is, and the policy is
// where the judgement lives: which failures deserve another request, and how long to wait.

describe('isRetryable', () => {
  // Transient: the server is busy or the connection blinked. These are the whole reason the layer
  // exists — the Dataforsyningen WMS drops requests under the burst a pan produces.
  it('retries the transient failures', () => {
    expect(isRetryable({ kind: 'timeout' })).toBe(true)
    expect(isRetryable({ kind: 'network' })).toBe(true)
    expect(isRetryable({ kind: 'http', status: 429 })).toBe(true)
    expect(isRetryable({ kind: 'http', status: 500 })).toBe(true)
    expect(isRetryable({ kind: 'http', status: 503 })).toBe(true)
  })

  // Settled: a malformed bbox or a rejected token fails identically on the third attempt, so retrying
  // only adds load to a server already refusing us.
  it('does not retry a request that will never succeed', () => {
    expect(isRetryable({ kind: 'http', status: 400 })).toBe(false)
    expect(isRetryable({ kind: 'http', status: 401 })).toBe(false)
    expect(isRetryable({ kind: 'http', status: 403 })).toBe(false)
    expect(isRetryable({ kind: 'http', status: 404 })).toBe(false)
  })

  // A tile server answering 200 with an error page is not this function's problem: the image decode
  // fails and that is reported as a network failure, which is retried. Pinned so the 2xx branch is not
  // "tidied" into the retryable list and made to loop on a broken response.
  it('treats a 2xx as not a failure at all', () => {
    expect(isRetryable({ kind: 'http', status: 200 })).toBe(false)
  })
})

describe('retryDelayMs', () => {
  // 1-based on the attempt being *scheduled*, so the second attempt waits 300 ms, the third 600 ms.
  it('backs off, starting at 300 ms for the second attempt', () => {
    expect(retryDelayMs(2)).toBe(300)
    expect(retryDelayMs(3)).toBe(600)
  })

  // The point of backing off: thirty tiles that all failed because thirty arrived at once must not all
  // come back immediately.
  it('grows with each attempt', () => {
    expect(retryDelayMs(3)).toBeGreaterThan(retryDelayMs(2))
  })

  // Total added latency for a tile that never arrives: bounded and small enough that an operator sees a
  // grey square rather than a frozen map.
  it('gives up well inside a second of waiting', () => {
    let total = 0
    for (let attempt = 2; attempt <= MAX_TILE_ATTEMPTS; attempt += 1) total += retryDelayMs(attempt)
    expect(total).toBeLessThan(1000)
  })
})

describe('budgets', () => {
  it('makes a bounded number of attempts', () => {
    expect(MAX_TILE_ATTEMPTS).toBe(3)
  })

  // Without a deadline there is no timeout at all: a hung request fires no event on an <img> until the
  // browser's own multi-minute stall limit, which is the failure the previous img-based attempt could
  // never even see.
  it('imposes a deadline short enough to replace a hung request while the map is still relevant', () => {
    expect(TILE_TIMEOUT_MS).toBeGreaterThan(2000)
    expect(TILE_TIMEOUT_MS).toBeLessThanOrEqual(10_000)
  })
})

describe('describeFailure', () => {
  // This string is the one line a permanently failed tile writes to the console, so it has to carry the
  // status: "a grey square appeared" and "the token is rejected" need different responses.
  it('names the status for an HTTP failure', () => {
    expect(describeFailure({ kind: 'http', status: 503 })).toBe('HTTP 503')
  })

  it('names the kind otherwise', () => {
    const cases: TileFailure[] = [{ kind: 'timeout' }, { kind: 'network' }]
    expect(cases.map(describeFailure)).toEqual(['timeout', 'network'])
  })
})
