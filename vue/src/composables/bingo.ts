/**
 * The maths behind the bingo curve on the dashboard.
 *
 * Kept out of the component so the parts that can be wrong silently — a tick that
 * lands mid-hour, an axis that rounds to a max nobody can read, a step path that
 * interpolates — are testable without mounting anything. See bingo.spec.ts.
 */

/** One step of the curve. The count holds until the next point; never interpolate. */
export type BingoPoint = {
  uts: number
  count: number
}

/** The payload of `GET /api/bingo`. */
export type BingoSeries = {
  /** The x-axis: the first post's opening hour and the last post's closing hour. */
  fromUts: number
  untilUts: number
  /** Where the line stops while the race is still running. */
  nowUts: number
  points: BingoPoint[]
  startedCount: number
  /** How many obligatoriske postlinjer the card is measured over. Zero means the curve is meaningless. */
  checkgroupCount: number
}

export const emptyBingoSeries: BingoSeries = {
  fromUts: 0,
  untilUts: 0,
  nowUts: 0,
  points: [],
  startedCount: 0,
  checkgroupCount: 0
}

/**
 * A y-axis maximum a human can read off.
 *
 * The count of patruljer is small and integral, so the axis is rounded up to a 1/2/5
 * multiple rather than to the exact maximum: an axis topping out at 73 makes every
 * gridline a number nobody can divide by eye, and it also makes the curve's shape
 * change whenever one more team signs up. Never below 1, so an empty race still has
 * an axis to draw on instead of dividing by zero.
 */
export function niceMax(value: number): number {
  if (!Number.isFinite(value) || value <= 1) return 1
  const magnitude = 10 ** Math.floor(Math.log10(value))
  for (const step of [1, 2, 5, 10]) {
    const candidate = step * magnitude
    if (value <= candidate) return candidate
  }
  return 10 * magnitude
}

/**
 * How many hours apart the x-axis gridlines should be.
 *
 * Chosen from the span rather than fixed at three hours, because the window is read
 * from the route's opening hours and a year that runs a shorter night would otherwise
 * get two gridlines. Aiming at roughly eight labels: enough to read a time off the
 * curve, few enough that they do not collide.
 */
export function tickStepHours(fromUts: number, untilUts: number): number {
  const hours = (untilUts - fromUts) / 3600
  for (const step of [1, 2, 3, 4, 6, 12]) {
    if (hours / step <= 8) return step
  }
  return 24
}

/**
 * Gridline positions, on whole hours in local time.
 *
 * Aligned to the clock and not to the start of the window: the window begins at the
 * first post's opening hour, which is a round time this year and need not be next
 * year, and a label reading "20.47" is not a time anybody is working against.
 *
 * Local time on purpose — the operator reads the same clock as the posts do, and
 * Copenhagen changes offset between some Nathejk weekends and others.
 */
export function hourTicks(fromUts: number, untilUts: number, stepHours: number): number[] {
  if (!(untilUts > fromUts) || stepHours <= 0) return []

  const first = new Date(fromUts * 1000)
  first.setMinutes(0, 0, 0)
  // Stepping through Date rather than adding stepHours*3600 to a timestamp, so a
  // daylight-saving change inside the window keeps the ticks on whole hours.
  const cursor = new Date(first)
  // Advance in whole steps from the hour boundary until inside the window.
  while (cursor.getTime() / 1000 < fromUts) cursor.setHours(cursor.getHours() + stepHours)

  const ticks: number[] = []
  // A guard, not a limit anybody should reach: a malformed window must not spin here.
  for (let i = 0; i < 500 && cursor.getTime() / 1000 <= untilUts; i++) {
    ticks.push(Math.round(cursor.getTime() / 1000))
    cursor.setHours(cursor.getHours() + stepHours)
  }
  return ticks
}

/** Maps a timestamp and a count to SVG user units. */
export type BingoScale = {
  x: (uts: number) => number
  y: (count: number) => number
}

/**
 * The curve as an SVG path: horizontal runs and vertical steps, no diagonals.
 *
 * A straight line between two points would claim the count changed gradually, and
 * every reading taken off the slope would be a number that never held. The count is
 * a step function — it holds until something happens — so the path says so.
 */
export function stepPath(points: BingoPoint[], scale: BingoScale): string {
  if (points.length === 0) return ''
  const parts = [`M ${scale.x(points[0].uts)} ${scale.y(points[0].count)}`]
  for (let i = 1; i < points.length; i++) {
    // Along at the old level, then straight up or down to the new one.
    parts.push(`L ${scale.x(points[i].uts)} ${scale.y(points[i - 1].count)}`)
    parts.push(`L ${scale.x(points[i].uts)} ${scale.y(points[i].count)}`)
  }
  return parts.join(' ')
}

/** The same shape closed down to the baseline, for the fill under the line. */
export function areaPath(points: BingoPoint[], scale: BingoScale, baselineY: number): string {
  const line = stepPath(points, scale)
  if (!line) return ''
  const last = points[points.length - 1]
  return `${line} L ${scale.x(last.uts)} ${baselineY} L ${scale.x(points[0].uts)} ${baselineY} Z`
}

/**
 * The count in force at a given moment — what the hover readout shows.
 *
 * The last point at or before the time, because that is what a step function means.
 * Undefined before the first point rather than clamped to it: the curve makes no
 * claim about a time it does not cover.
 */
export function countAt(points: BingoPoint[], uts: number): number | undefined {
  let found: number | undefined
  for (const p of points) {
    if (p.uts > uts) break
    found = p.count
  }
  return found
}
