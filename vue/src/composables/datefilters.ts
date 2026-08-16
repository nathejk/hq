const format = (opt: Intl.DateTimeFormatOptions) => new Intl.DateTimeFormat('da-DK', opt)

function isValidDate(d: Date): boolean {
  return !isNaN(d.getTime()) && d.getFullYear() > 1970
}

// The API serves some timestamps as Go's time.Time text form rather than ISO 8601,
// because they are stored as VARCHAR and handed back verbatim — order.createdAt,
// order.changedAt, payment.createdAt, payment.changedAt and signup.createdAt all
// look like:
//
//	2026-07-12 21:25:07.299012709 +0000 UTC
//
// which is a space instead of T, nine fractional digits, and a trailing zone name
// after the offset. Parsing that is implementation-defined: V8 accepts it, so it
// renders fine in Chrome, while Safari rejects it and every such column showed
// "NaN. Invalid Date Invalid Date".
//
// Hence an explicit parse rather than trusting the engine. Anything already ISO
// still goes through the native parser, which is well defined for that.
const timestamp =
  /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?(?:\.(\d+))?(?:\s*(Z|[+-]\d{2}:?\d{2}))?(?:\s+(\S+))?$/

/**
 * parseApiDate turns a value from the API into a Date, or null when it cannot be
 * read as one. Timestamps before 1971 count as absent: an unset Go time marshals
 * as year 1, and showing "1. jan. 00:00" for "never" is worse than showing nothing.
 */
export function parseApiDate(value: Date | string | number | null | undefined): Date | null {
  if (value === null || value === undefined || value === '') return null

  if (value instanceof Date) return isValidDate(value) ? value : null
  if (typeof value === 'number') {
    const d = new Date(value)
    return isValidDate(d) ? d : null
  }
  // Anything that is not a string cannot be pattern-matched; hand it to the native
  // parser and let the validity check reject it.
  if (typeof value !== 'string') {
    const d = new Date(value as never)
    return isValidDate(d) ? d : null
  }

  const m = timestamp.exec(value.trim())
  if (m) {
    const [, year, month, day, hour, minute, second, fraction, zone, zoneName] = m
    // Only the first three fractional digits are milliseconds; Go writes nine.
    // Truncated rather than rounded, so .999999999 stays inside the same second.
    const ms = fraction ? Number(fraction.slice(0, 3).padEnd(3, '0')) : 0
    const parts = [
      Number(year),
      Number(month) - 1,
      Number(day),
      Number(hour),
      Number(minute),
      Number(second ?? 0),
      ms,
    ] as const

    // A named UTC zone with no numeric offset still means UTC. Any other zone name
    // is ignored in favour of the offset, which is what Go puts in front of it.
    const named = zoneName && /^(UTC|GMT|Z)$/i.test(zoneName)

    let d: Date
    if (zone || named) {
      let stamp = Date.UTC(...parts)
      if (zone && zone !== 'Z') {
        const [zh, zm] = (zone.slice(1).match(/\d{2}/g) ?? ['0', '0']).map(Number)
        const offset = (zh * 60 + zm) * 60_000
        stamp += zone[0] === '-' ? offset : -offset
      }
      d = new Date(stamp)
    } else {
      // No zone at all: local time, matching what the native parser does for an ISO
      // date-time without an offset.
      d = new Date(...parts)
    }
    return isValidDate(d) ? d : null
  }

  const fallback = new Date(value)
  return isValidDate(fallback) ? fallback : null
}

export function ddddhhmm(date: Date | string | number) {
  const d = parseApiDate(date)
  if (!d) return ''
  return format({ weekday: 'long', hour: '2-digit', minute: '2-digit' }).format(d)
}
export function dddhhmm(date: Date | string | number) {
  const d = parseApiDate(date)
  if (!d) return ''
  return format({ weekday: 'short', hour: '2-digit', minute: '2-digit' }).format(d)
}
export function hhmm(date: Date | string | number) {
  const d = parseApiDate(date)
  if (!d) return ''
  return format({ hour: '2-digit', minute: '2-digit' }).format(d)
}

/**
 * daymonthhhmm renders "12. jul. 21:25" — the format the payment and order tables
 * use for a timestamp. Shared so the two views cannot drift apart, and so the
 * parsing above is not reimplemented per view, which is how the Safari bug came to
 * exist in two places at once.
 */
export function daymonthhhmm(date: Date | string | number | null | undefined) {
  const d = parseApiDate(date)
  if (!d) return ''
  const day = d.getDate()
  const month = format({ month: 'short' }).format(d)
  const time = format({ hour: '2-digit', minute: '2-digit', hour12: false }).format(d)
  return `${day}. ${month} ${time}`
}
