import { describe, expect, it } from 'vitest'
import {
  STALE_AFTER_MS,
  formatClock,
  formatRelative,
  isStale,
} from './usePositionPresence'

// The pure half of the composable. `usePositionPresence` itself needs the live cache and axios, and
// is exercised through the views; these three functions decide what an operator actually reads, so
// they are worth pinning on their own.

describe('isStale', () => {
  const now = Date.parse('2026-09-03T14:00:00Z')

  it('treats a fresh position as fresh', () => {
    expect(isStale(now - 60_000, now)).toBe(false)
  })

  it('treats a position older than the threshold as stale', () => {
    expect(isStale(now - STALE_AFTER_MS - 1, now)).toBe(true)
  })

  // Exactly at the threshold is not yet stale: the boundary has to fall somewhere, and "older
  // than 30 minutes" is what the constant's name says.
  it('is not stale exactly at the threshold', () => {
    expect(isStale(now - STALE_AFTER_MS, now)).toBe(false)
  })
})

describe('formatRelative', () => {
  const now = Date.parse('2026-09-03T14:00:00Z')

  it('says "lige nu" under a minute', () => {
    expect(formatRelative(now - 5_000, now)).toBe('lige nu')
  })

  // A phone with a fast clock, or a point that arrived while the page sat open, can be marginally
  // in the future. "for -1 minutter siden" would look broken over something meaningless.
  it('does not produce a negative age for a future timestamp', () => {
    expect(formatRelative(now + 30_000, now)).toBe('lige nu')
  })

  it('uses the singular for one minute', () => {
    expect(formatRelative(now - 60_000, now)).toBe('for 1 minut siden')
  })

  it('pluralises minutes', () => {
    expect(formatRelative(now - 6 * 60_000, now)).toBe('for 6 minutter siden')
  })

  it('switches to hours, singular', () => {
    expect(formatRelative(now - 60 * 60_000, now)).toBe('for 1 time siden')
  })

  it('pluralises hours', () => {
    expect(formatRelative(now - 3 * 60 * 60_000, now)).toBe('for 3 timer siden')
  })

  // A 30-hour race plus the days after it: a gøgler who reported once on Friday is still
  // legitimately shown in the list on Sunday.
  it('switches to days', () => {
    expect(formatRelative(now - 48 * 60 * 60_000, now)).toBe('for 2 dage siden')
  })

  it('uses the singular for one day', () => {
    expect(formatRelative(now - 24 * 60 * 60_000, now)).toBe('for 1 dag siden')
  })
})

describe('formatClock', () => {
  // Date-less by design: during a race everyone is reasoning within the same 30 hours, and the
  // relative half of the tooltip disambiguates anything genuinely old.
  it('formats as a Danish 24-hour clock time with no date', () => {
    const formatted = formatClock(Date.parse('2026-09-03T14:32:00Z'))
    expect(formatted).toMatch(/^\d{2}[.:]\d{2}$/)
  })
})
