import { describe, expect, it } from 'vitest'
import { daymonthhhmm, hhmm, parseApiDate } from './datefilters'

// These tests run on V8, which parses Go's time.Time text form out of leniency —
// so a test that merely says "the prod string renders" would pass even with the
// broken implementation. The assertions below therefore pin the exact instant, and
// one of them feeds a string V8 cannot parse at all, which only succeeds if the
// explicit parse is doing the work.
describe('parseApiDate', () => {
  // The literal shape served by payment.createdAt, order.changedAt and
  // signup.createdAt: space separator, nine fractional digits, offset, zone name.
  const goForm = '2026-07-12 21:25:07.299012709 +0000 UTC'

  it('parses the Go time.Time text form to the right instant', () => {
    expect(parseApiDate(goForm)?.toISOString()).toBe('2026-07-12T21:25:07.299Z')
  })

  it('does not rely on the engine being lenient', () => {
    // Go writes the offset *and* a zone name. V8 tolerates "+0000 UTC", which is
    // why Chrome renders these today, but rejects any other zone name outright —
    // so a payment recorded outside UTC would break in Chrome as well. This can
    // only pass through the explicit parse.
    const cest = '2026-07-12 23:25:07.299012709 +0200 CEST'
    expect(new Date(cest).getTime()).toBeNaN()
    expect(parseApiDate(cest)?.toISOString()).toBe('2026-07-12T21:25:07.299Z')
  })

  it('applies a non-zero offset', () => {
    expect(parseApiDate('2026-07-12 23:25:07 +0200 CEST')?.toISOString()).toBe(
      '2026-07-12T21:25:07.000Z',
    )
    expect(parseApiDate('2026-07-12 17:25:07 -0400 EDT')?.toISOString()).toBe(
      '2026-07-12T21:25:07.000Z',
    )
    expect(parseApiDate('2026-07-12T23:25:07+02:00')?.toISOString()).toBe(
      '2026-07-12T21:25:07.000Z',
    )
  })

  it('reads a zone name with no offset as UTC', () => {
    expect(parseApiDate('2026-07-12 21:25:07.299012709 UTC')?.toISOString()).toBe(
      '2026-07-12T21:25:07.299Z',
    )
  })

  it('reads a timestamp with no zone at all as local, like the native parser', () => {
    // Asserted as an equivalence rather than a fixed instant, so the test holds
    // whatever timezone it runs in.
    expect(parseApiDate('2026-07-12 21:25:07')?.getTime()).toBe(
      new Date('2026-07-12T21:25:07').getTime(),
    )
  })

  it('still handles ISO 8601, which most of the API serves', () => {
    expect(parseApiDate('2026-07-12T21:25:07Z')?.toISOString()).toBe('2026-07-12T21:25:07.000Z')
    expect(parseApiDate('2026-07-12T21:25:07.299Z')?.toISOString()).toBe(
      '2026-07-12T21:25:07.299Z',
    )
  })

  it('truncates rather than rounds sub-millisecond precision', () => {
    // 999999999ns is 999ms and change; rounding up would roll into the next second.
    expect(parseApiDate('2026-07-12 21:25:07.999999999 +0000 UTC')?.toISOString()).toBe(
      '2026-07-12T21:25:07.999Z',
    )
    expect(parseApiDate('2026-07-12 21:25:07.5 +0000 UTC')?.toISOString()).toBe(
      '2026-07-12T21:25:07.500Z',
    )
  })

  it('treats an unset Go timestamp as absent', () => {
    // The zero time marshals as year 1; "1. jan. 00:00" would be a lie.
    expect(parseApiDate('0001-01-01 00:00:00 +0000 UTC')).toBeNull()
  })

  it('returns null for empty and unreadable values', () => {
    for (const value of [null, undefined, '', 'not a date', {} as never]) {
      expect(parseApiDate(value as never)).toBeNull()
    }
  })

  it('passes Date and epoch values through', () => {
    const d = new Date('2026-07-12T21:25:07Z')
    expect(parseApiDate(d)).toBe(d)
    expect(parseApiDate(d.getTime())?.toISOString()).toBe('2026-07-12T21:25:07.000Z')
    expect(parseApiDate(new Date('nonsense'))).toBeNull()
  })
})

describe('daymonthhhmm', () => {
  // The regression: /betalinger rendered "NaN. Invalid Date Invalid Date" in Safari
  // for every row, because each view parsed this string itself with new Date().
  it('formats the Go text form instead of producing NaN', () => {
    const out = daymonthhhmm('2026-07-12 21:25:07.299012709 +0000 UTC')
    expect(out).not.toContain('NaN')
    expect(out).not.toContain('Invalid')
    expect(out).toMatch(/^12\. jul\.? \d{2}[.:]\d{2}$/)
  })

  it('renders nothing for a missing timestamp', () => {
    expect(daymonthhhmm('')).toBe('')
    expect(daymonthhhmm(null)).toBe('')
    expect(daymonthhhmm('0001-01-01 00:00:00 +0000 UTC')).toBe('')
  })
})

describe('hhmm', () => {
  it('accepts the Go text form too, not just ISO', () => {
    expect(hhmm('2026-07-12 21:25:07.299012709 +0000 UTC')).toMatch(/^\d{2}[.:]\d{2}$/)
    expect(hhmm('nonsense')).toBe('')
  })
})
