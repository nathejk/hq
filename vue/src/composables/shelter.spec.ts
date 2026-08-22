import { describe, it, expect } from 'vitest'
import { formatClock, formatElapsed, formatSince, minutesSince } from './shelter'

// The durations on the Hønsegården screen are what tell the crew something has gone on too
// long, so the arithmetic is worth pinning — including the awkward inputs, which is where a
// display like this actually fails at 3am.

const at = (iso: string) => new Date(iso).getTime()

describe('formatElapsed', () => {
  it('reports minutes below the hour', () => {
    expect(formatElapsed('2026-09-26T00:42:00Z', at('2026-09-26T01:00:00Z'))).toBe('18m')
  })

  it('reports hours and minutes above the hour', () => {
    expect(formatElapsed('2026-09-25T21:40:00Z', at('2026-09-25T23:54:00Z'))).toBe('2t 14m')
  })

  it('reports a whole hour without stray minutes', () => {
    expect(formatElapsed('2026-09-25T21:40:00Z', at('2026-09-25T22:40:00Z'))).toBe('1t 0m')
  })

  // Server and browser clocks disagree by a second or two, and a status stamped "just now"
  // can land in the future. "-1m" would look broken; explaining machine clock skew to a
  // volunteer is not worth a pixel.
  it('clamps a future timestamp to zero rather than going negative', () => {
    expect(formatElapsed('2026-09-26T01:00:00Z', at('2026-09-26T00:42:00Z'))).toBe('0m')
  })

  // A missing or unparseable timestamp renders as nothing at all. A row with no "siden" is
  // odd; a row saying "NaNm" makes the crew distrust the whole screen.
  it('renders nothing for a missing or unparseable value', () => {
    expect(formatElapsed(undefined, Date.now())).toBe('')
    expect(formatElapsed('not a date', Date.now())).toBe('')
  })
})

describe('formatSince', () => {
  // Both halves, because they are used differently: the clock time gets written on paper and
  // read out over the radio, the elapsed span triggers a decision.
  it('gives the clock time and the elapsed span together', () => {
    const text = formatSince('2026-09-25T21:40:00Z', at('2026-09-25T23:54:00Z'))
    expect(text).toContain('2t 14m')
    expect(text).toMatch(/^siden \d{2}[:.]\d{2} /)
  })

  it('renders nothing without a timestamp', () => {
    expect(formatSince(undefined, Date.now())).toBe('')
  })
})

describe('formatClock', () => {
  it('renders a two-digit da-DK time', () => {
    expect(formatClock('2026-09-25T21:40:00Z')).toMatch(/^\d{2}[:.]\d{2}$/)
  })

  it('renders nothing for rubbish', () => {
    expect(formatClock('')).toBe('')
    expect(formatClock('not a date')).toBe('')
  })
})

describe('minutesSince', () => {
  it('counts whole minutes', () => {
    expect(minutesSince('2026-09-26T00:00:00Z', at('2026-09-26T00:45:30Z'))).toBe(45)
  })

  // Drives the overdue highlight, so a missing timestamp must read as "not overdue" rather
  // than lighting up a row for no reason.
  it('treats a missing timestamp as zero', () => {
    expect(minutesSince(undefined, Date.now())).toBe(0)
  })
})
