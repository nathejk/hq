/**
 * Day/time arithmetic for DayTimePicker.
 *
 * The picker splits a timestamp into an *event day index* (0 = first day of the
 * year edition) plus a wall-clock time. The day index must be derived from the
 * calendar day, not from the elapsed time since the offset: `year.dateStart`
 * carries no meaningful time of day, so a raw millisecond delta puts e.g.
 * Saturday 01:00 less than 24h after the offset (day 0 = Friday) and makes an
 * early hour on the first day negative — which is how checkpoints ended up
 * stored with an `openUntil` before their `openFrom`.
 */

/** Midnight, local time, of the day `d` falls on. */
export const startOfDay = (d: Date): Date => {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  return x
}

/** `d` shifted by `n` calendar days, keeping wall-clock time across DST shifts. */
export const addDays = (d: Date, n: number): Date => {
  const x = new Date(d)
  x.setDate(x.getDate() + n)
  return x
}

/** Whole calendar days from `b` to `a`, ignoring time of day. */
export const dayDiff = (a: Date, b: Date): number => Math.round((startOfDay(a).getTime() - startOfDay(b).getTime()) / 86400000)

/**
 * A date-only offset such as `year.dateStart` ("2026-09-18") is parsed as UTC
 * midnight by `Date`, which resolves to the previous calendar day in any zone
 * behind UTC. Treat bare dates as local dates; pass anything else through.
 */
export const parseOffset = (v?: string | null): Date => {
  if (!v) return new Date(0)
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(v)
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : new Date(v)
}

/** Day index of `ts` relative to `offset`, limited to the switch's positions. */
export const dayIndex = (ts: Date, offset: Date, dayCount: number): number => {
  const max = Math.max(dayCount - 1, 0)
  return Math.min(Math.max(dayDiff(ts, offset), 0), max)
}

/** The timestamp for day `day` (relative to `offset`) at wall-clock `hh:mm`. */
export const composeDayTime = (offset: Date, day: number, hh: number, mm: number): Date => {
  const base = addDays(startOfDay(offset), day)
  base.setHours(hh, mm, 0, 0)
  return base
}

const week = [
  { name: 'Søndag', shortName: 'søn' },
  { name: 'Mandag', shortName: 'man' },
  { name: 'Tirsdag', shortName: 'tirs' },
  { name: 'Onsdag', shortName: 'ons' },
  { name: 'Torsdag', shortName: 'tors' },
  { name: 'Fredag', shortName: 'fre' },
  { name: 'Lørdag', shortName: 'lør' }
]

/** Switch options: `dayCount` consecutive weekdays starting at `offset`. */
export const dayOptions = (offset: Date, dayCount: number): { name: string; shortName: string; value: number }[] => {
  const out: { name: string; shortName: string; value: number }[] = []
  for (let i = 0; i < Math.max(dayCount, 0); i++) {
    const w = week[(offset.getDay() + i) % 7]
    out.push({ name: w.name, shortName: w.shortName, value: i })
  }
  return out
}
