/**
 * The vagter roster as one timeline per unit: a multi-range axis spanning the whole race.
 *
 * # Why one axis and not a column per day
 *
 * The roster is read as a comparison. The question it answers is never "when is Bil 1 on?" but
 * **"who covers 02:00?"**, and the answer has to be legible across units *and* across midnight.
 * Split into a column per day, every night shift became two bars in two cells with a seam down
 * the middle of the thing being judged — and the seam fell exactly where the race is hardest to
 * staff. One axis from the start of the race to the end of it has no seam: a shift is one bar,
 * and a hole in the night is a gap you can point at.
 *
 * # Positions are absolute times; the axis is a percentage
 *
 * A window stays one absolute span (`startUts`–`endUts`), which is what the API stores and what
 * survives the axis being redrawn. Everything here converts between that and a percentage of the
 * track, snapping to a coarse step on the way in.
 *
 * # Pairs, and why their order is never touched
 *
 * A period is two knots with the bar between them marked. Pairs are appended — a new one goes at
 * the end of the list and at the end of the timeline — and nothing here ever re-sorts them. That
 * is load-bearing rather than tidy: a knot carries the id of the duty window it edits, and a
 * slider that reordered its handles when one was dragged past another would move the *other*
 * unit's shift, silently and plausibly. Overlaps are allowed for the same reason the API allows
 * them: a unit rostered twice over the same hour is on duty either way, and refusing it would
 * block an operator fixing a roster in whatever order makes sense to them.
 */

import { addDays, startOfDay } from '@/composables/daytime'
import { type Duty, formatUts, formatUtsTime } from '@/composables/dispatch'

/** Quarter-hour steps. A roster agreed over the phone is never to the minute, and a coarse step
 *  is what makes a drag land where the operator meant on an axis that is days wide. */
export const STEP_MINUTES = 15

/** How long a newly added period is. Two hours: long enough to grab and drag on a multi-day
 *  axis, short enough that leaving it unadjusted is obviously unfinished rather than plausible. */
export const NEW_PERIOD_MINUTES = 120

/** The shortest a window can be dragged to. Shorter is not a shift, and collapsing one to
 *  nothing is what the remove button is for. */
export const MIN_PERIOD_MINUTES = 15

/** A guard, not a limit: a malformed span must not render ten thousand gridlines and hang the
 *  screen. Three weeks is far beyond any Nathejk. */
const MAX_DAYS = 21

const weekdays = ['søn', 'man', 'tir', 'ons', 'tor', 'fre', 'lør']

/** One gridline on the axis: a midnight, with the day it starts. */
export interface DutyDay {
  /** Midnight starting the day. */
  startUts: number
  /** Midnight ending it — the next day's start, so the days tile without gaps. */
  endUts: number
  /** "lør 19/9". Weekday-bearing because "02:00" alone does not say which night. */
  label: string
  /** Stable key for `v-for`, across revalidations. */
  key: string
}

const dayLabel = (d: Date) => `${weekdays[d.getDay()]} ${d.getDate()}/${d.getMonth() + 1}`

/**
 * The event's days, as the axis's gridlines and labels.
 *
 * Empty for an unset span, which the editor renders as "the year has no dates" — an honest empty
 * state beats an axis on guessed dates that silently rosters the wrong weekend.
 */
export const eventDays = (startUts: number, endUts: number): DutyDay[] => {
  if (!startUts || !endUts || endUts <= startUts) return []
  const first = startOfDay(new Date(startUts * 1000))
  const days: DutyDay[] = []
  for (let i = 0; i < MAX_DAYS; i++) {
    const from = addDays(first, i)
    if (from.getTime() / 1000 >= endUts) break
    const to = addDays(first, i + 1)
    days.push({
      startUts: Math.floor(from.getTime() / 1000),
      endUts: Math.floor(to.getTime() / 1000),
      label: dayLabel(from),
      key: `${from.getFullYear()}-${from.getMonth() + 1}-${from.getDate()}`,
    })
  }
  return days
}

/** The axis a unit's row is drawn on. */
export interface Axis {
  startUts: number
  endUts: number
}

/** Where `uts` sits on the axis, as a percentage. Clamped, so a window that predates the axis —
 *  a roster kept while the year's dates were corrected — is drawn at the edge rather than off it. */
export const percentOf = (uts: number, axis: Axis): number => {
  const span = axis.endUts - axis.startUts
  if (span <= 0) return 0
  return Math.min(Math.max(((uts - axis.startUts) / span) * 100, 0), 100)
}

/** The time at a percentage of the axis, snapped to the step. */
export const utsAt = (percent: number, axis: Axis, stepMinutes = STEP_MINUTES): number => {
  const span = axis.endUts - axis.startUts
  const raw = axis.startUts + (Math.min(Math.max(percent, 0), 100) / 100) * span
  const step = stepMinutes * 60
  return Math.round(raw / step) * step
}

/** One period on the axis: a pair of knots and the bar between them. */
export interface Pair {
  duty: Duty
  /** Left edge as a percentage of the axis. */
  fromPercent: number
  /** Width as a percentage. Never zero: a bar nobody can see is a bar nobody can grab. */
  widthPercent: number
  /** True when the window falls partly outside the axis, so its bar is drawn clipped. Worth
   *  saying out loud on screen — the shift is real, the axis is what is wrong. */
  clipped: boolean
}

/**
 * A unit's windows as pairs on the axis, in the order they are stored.
 *
 * Not sorted by time. See the note on pairs above: the index is an identity here, and sorting
 * would hand a dragged knot somebody else's shift.
 */
export const pairsOn = (windows: Duty[], axis: Axis): Pair[] =>
  windows.map((duty) => {
    const fromPercent = percentOf(duty.startUts, axis)
    const toPercent = percentOf(duty.endUts, axis)
    return {
      duty,
      fromPercent,
      // A floor rather than the true width: a fifteen-minute window on a three-day axis is 0.3%
      // of it, which rounds to nothing on screen.
      widthPercent: Math.max(toPercent - fromPercent, 0.4),
      clipped: duty.startUts < axis.startUts || duty.endUts > axis.endUts,
    }
  })

/** A span being edited. Only the two times matter here, so a bare span is accepted as well as a
 *  whole `Duty` — the drag state is not a duty window until it is saved. */
interface Span {
  startUts: number
  endUts: number
}

/**
 * Where a dragged knot puts its window.
 *
 * The knot being dragged is the only edge that moves; the other stays where the payload says it
 * is, so adjusting the end of a shift cannot quietly move its start. Both edges are kept on the
 * axis and at least `MIN_PERIOD_MINUTES` apart — an edge pushed past its partner stops against
 * it rather than turning the window inside out, which would read as a different shift entirely.
 */
export const dragKnot = (
  span: Span,
  edge: 'start' | 'end',
  uts: number,
  axis: Axis,
): { startUts: number; endUts: number } => {
  const min = MIN_PERIOD_MINUTES * 60
  const clamped = Math.min(Math.max(uts, axis.startUts), axis.endUts)
  if (edge === 'start') {
    return { startUts: Math.min(clamped, span.endUts - min), endUts: span.endUts }
  }
  return { startUts: span.startUts, endUts: Math.max(clamped, span.startUts + min) }
}

/**
 * Where a dragged *bar* puts its window: the whole period moves, its length unchanged.
 *
 * Moving a shift is a distinct instruction from resizing one — "they start two hours later" is
 * not "they work two hours more" — and at the edges of the axis the length is what has to
 * survive, so the span is shifted back inside rather than squashed against the end.
 */
export const dragPair = (
  span: Span,
  startUts: number,
  axis: Axis,
): { startUts: number; endUts: number } => {
  const length = span.endUts - span.startUts
  const from = Math.min(
    Math.max(startUts, axis.startUts),
    Math.max(axis.endUts - length, axis.startUts),
  )
  return { startUts: from, endUts: from + length }
}

/**
 * The span a newly added period gets: at the end of the timeline, after everything already there.
 *
 * At the end for two reasons. A new pair is appended to the list, so putting it anywhere else
 * would make the visual order disagree with the stored order; and a period that cannot land on
 * top of an existing one leaves no doubt about which bar was just created and is now being
 * dragged.
 *
 * It is `NEW_PERIOD_MINUTES` long where there is room and shrinks to fit where there is not, down
 * to the minimum — so adding is always possible, and a unit already rostered to the end of the
 * race gets a sliver at the end to drag out rather than a button that does nothing.
 */
export const newPeriod = (
  windows: Duty[],
  axis: Axis,
): { startUts: number; endUts: number } => {
  const lastEnd = windows.reduce((max, w) => Math.max(max, w.endUts), axis.startUts)
  const min = MIN_PERIOD_MINUTES * 60
  const wanted = NEW_PERIOD_MINUTES * 60
  const from = Math.max(Math.min(lastEnd, axis.endUts - min), axis.endUts - wanted, axis.startUts)
  // Flush with the end of the race, always: that is what "at the end of the timeline" means, and
  // it is what makes the newest bar findable without looking for it.
  return { startUts: from, endUts: axis.endUts }
}

/**
 * Zoom levels for the timeline, as a multiple of the container's width.
 *
 * The roster has to work on a laptop in a tent and on the big screen in HQ, and those want opposite
 * things: the whole race at a glance, or a quarter-hour you can actually aim at. Multiples of the
 * *container* rather than fixed pixels-per-hour, so 1× always means "it fits" on either machine.
 *
 * Powers of two because the useful gesture is "about twice as close", and four steps span a
 * three-day race from whole-race overview to roughly an hour per centimetre.
 */
export const ZOOM_STEPS = [1, 2, 4, 8] as const

/** The next zoom level in `direction`, clamped at both ends. */
export const zoomBy = (current: number, direction: 1 | -1): number => {
  const index = ZOOM_STEPS.indexOf(current as (typeof ZOOM_STEPS)[number])
  const from = index === -1 ? 0 : index
  return ZOOM_STEPS[Math.min(Math.max(from + direction, 0), ZOOM_STEPS.length - 1)]
}

/**
 * Where *now* is on the axis, or null when now is not in the race at all.
 *
 * Null rather than clamped: a line pinned to the left edge in the week before the race does not say
 * "not yet", it says "the race started at midnight on Friday and it is now", which is a lie an
 * operator would plan against. The marker simply does not exist outside the race.
 *
 * Milliseconds in, because that is what `useNow` ticks.
 */
export const nowPercent = (nowMs: number, axis: Axis): number | null => {
  const uts = Math.floor(nowMs / 1000)
  if (!axis.startUts || !axis.endUts) return null
  if (uts < axis.startUts || uts > axis.endUts) return null
  return percentOf(uts, axis)
}

/**
 * The label that goes *inside* a bar: "21.40–02.00".
 *
 * The clock alone, deliberately. The weekday is what the bar's own position on the axis already
 * says, and dropping it is most of what makes the label fit inside a four-hour bar — which is the
 * point of putting it there rather than in a list underneath.
 */
export const barLabel = (span: { startUts: number; endUts: number }): string =>
  `${formatUtsTime(span.startUts)}–${formatUtsTime(span.endUts)}`

/**
 * The full label, for the hover card, the tooltip and the screen reader: "fre 21.40 – lør 02.00".
 *
 * Weekday-bearing, and on both ends when the shift crosses midnight — which on this screen most of
 * them do, and "21.40 til 02.00" alone does not say which night. Where it does not cross, the day is
 * said once: repeating it reads as a two-day shift.
 */
export const spanLabel = (span: { startUts: number; endUts: number }): string => {
  const sameDay =
    new Date(span.startUts * 1000).toDateString() === new Date(span.endUts * 1000).toDateString()
  return `${formatUts(span.startUts)}–${sameDay ? formatUtsTime(span.endUts) : formatUts(span.endUts)}`
}

/** Total hours a unit is rostered, for the row summary. Overlaps are counted once: two windows
 *  over the same hour is a roster mistake, not two hours of cover. */
export const rosteredHours = (windows: Duty[]): number => {
  const spans = [...windows].sort((a, b) => a.startUts - b.startUts)
  let seconds = 0
  let cursor = -Infinity
  for (const w of spans) {
    const from = Math.max(w.startUts, cursor)
    if (w.endUts > from) {
      seconds += w.endUts - from
      cursor = w.endUts
    }
  }
  return Math.round((seconds / 3600) * 10) / 10
}

/**
 * The hours of the race nobody at all is rostered for.
 *
 * The number the screen exists to drive to zero, and the reason the axis spans the whole race
 * rather than a day at a time: a gap is only visible against the hours either side of it.
 */
export const uncoveredHours = (windows: Duty[], axis: Axis): number => {
  const span = axis.endUts - axis.startUts
  if (span <= 0) return 0
  const spans = [...windows]
    .map((w) => ({
      startUts: Math.max(w.startUts, axis.startUts),
      endUts: Math.min(w.endUts, axis.endUts),
    }))
    .filter((w) => w.endUts > w.startUts)
    .sort((a, b) => a.startUts - b.startUts)
  let covered = 0
  let cursor = axis.startUts
  for (const w of spans) {
    const from = Math.max(w.startUts, cursor)
    if (w.endUts > from) {
      covered += w.endUts - from
      cursor = w.endUts
    }
  }
  return Math.round(((span - covered) / 3600) * 10) / 10
}
