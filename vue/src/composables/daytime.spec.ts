import { describe, it, expect } from 'vitest'
import { parseOffset, dayIndex, composeDayTime, dayOptions, dayDiff } from './daytime'

// The suite pins the picker's behaviour for a real year edition: 2026 starts
// Friday 2026-09-18 and runs three days (fre/lør/søn).
const offset = parseOffset('2026-09-18')
const dayCount = 3

describe('parseOffset', () => {
  it('treats a date-only year.dateStart as a local date, not UTC midnight', () => {
    const d = parseOffset('2026-09-18')
    expect(d.getFullYear()).toBe(2026)
    expect(d.getMonth()).toBe(8)
    expect(d.getDate()).toBe(18)
    expect(d.getHours()).toBe(0)
  })

  it('passes full timestamps through', () => {
    expect(parseOffset('2026-09-18T21:00:00Z').toISOString()).toBe('2026-09-18T21:00:00.000Z')
  })

  it('falls back to the epoch without an offset', () => {
    expect(parseOffset(undefined).getTime()).toBe(0)
    expect(parseOffset(null).getTime()).toBe(0)
  })
})

describe('dayOptions', () => {
  it('lists consecutive weekdays from the offset', () => {
    expect(dayOptions(offset, dayCount).map((o) => o.shortName)).toEqual(['fre', 'lør', 'søn'])
  })

  it('wraps across the week boundary', () => {
    expect(dayOptions(parseOffset('2026-09-19'), 3).map((o) => o.shortName)).toEqual(['lør', 'søn', 'man'])
  })

  it('numbers positions from zero so the value is an event-day index', () => {
    expect(dayOptions(offset, dayCount).map((o) => o.value)).toEqual([0, 1, 2])
  })
})

describe('dayIndex', () => {
  it('uses the calendar day, not elapsed time since the offset', () => {
    // The regression: 01:00 on day 2 is only ~25h after a midnight offset and
    // was previously rounded down to day 0, pinning every small hour to Friday.
    expect(dayIndex(composeDayTime(offset, 1, 1, 0), offset, dayCount)).toBe(1)
    expect(dayIndex(composeDayTime(offset, 2, 1, 0), offset, dayCount)).toBe(2)
  })

  it('clamps to the available positions', () => {
    expect(dayIndex(composeDayTime(offset, 0, 0, 0), offset, dayCount)).toBe(0)
    expect(dayIndex(new Date(2026, 8, 15), offset, dayCount)).toBe(0)
    expect(dayIndex(new Date(2026, 8, 30), offset, dayCount)).toBe(2)
  })
})

describe('composeDayTime', () => {
  it('keeps the picked wall-clock time on the picked day', () => {
    const d = composeDayTime(offset, 1, 1, 30)
    expect(d.getDate()).toBe(19)
    expect(d.getHours()).toBe(1)
    expect(d.getMinutes()).toBe(30)
  })

  it('zeroes seconds so repeated edits compare equal', () => {
    const d = composeDayTime(parseOffset('2026-09-18T21:00:45Z'), 0, 9, 0)
    expect(d.getSeconds()).toBe(0)
    expect(d.getMilliseconds()).toBe(0)
  })

  it('round-trips every position at every hour', () => {
    for (let day = 0; day < dayCount; day++) {
      for (let hour = 0; hour < 24; hour++) {
        const ts = composeDayTime(offset, day, hour, 0)
        expect(dayIndex(ts, offset, dayCount)).toBe(day)
        expect(ts.getHours()).toBe(hour)
      }
    }
  })

  it('stays ordered when an end time falls past midnight', () => {
    // Post 1A: open Friday 23:00, close Saturday 01:00.
    const from = composeDayTime(offset, 0, 23, 0)
    const until = composeDayTime(offset, 1, 1, 0)
    expect(until.getTime()).toBeGreaterThan(from.getTime())
  })
})

describe('DST', () => {
  it('advances one calendar day across the autumn shift', () => {
    // 2026-10-25 is the CEST -> CET change in Europe/Copenhagen.
    const dst = parseOffset('2026-10-24')
    const d = composeDayTime(dst, 2, 1, 0)
    expect(d.getDate()).toBe(26)
    expect(d.getHours()).toBe(1)
    expect(dayDiff(d, dst)).toBe(2)
  })
})
