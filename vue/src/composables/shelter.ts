// Shared bits of the Hønsegården UI (PRD 007): the ticking clock the durations are
// measured against, and the formatting the rows use.
//
// A composable rather than a store, matching the rest of the SPA. There is one piece of
// genuinely shared *state* here — `now` — and it is shared on purpose: see below.

import { computed, ref, onScopeDispose, type Ref } from 'vue'

/** How often the elapsed durations are recomputed. */
const TICK_MS = 30_000

/**
 * A clock that advances, shared by every row on the screen.
 *
 * The durations on this screen ("venter siden 21:40 (2t 14m)") are the crew's cue to do
 * something, and a duration computed once at render is wrong a minute later and wronger all
 * night. Vue will not recompute it either, because nothing it depends on has changed — the
 * status did not move, only time passed. So the passage of time has to be a reactive value.
 *
 * One interval for the whole screen rather than one per row: forty rows would otherwise mean
 * forty timers waking up out of step, and the columns would visibly disagree about what time
 * it is.
 *
 * Thirty seconds because the smallest unit displayed is the minute; a per-second tick would
 * re-render the table 120 times more often to show the same string.
 */
let subscribers = 0
let timer: ReturnType<typeof setInterval> | undefined
const now = ref(Date.now())

export function useNow(): Readonly<Ref<number>> {
  subscribers += 1
  if (!timer) {
    timer = setInterval(() => {
      now.value = Date.now()
    }, TICK_MS)
  }
  onScopeDispose(() => {
    subscribers -= 1
    // The timer stops with the last view that wanted it. A screen nobody is looking at has
    // no business waking the browser up every half minute.
    if (subscribers === 0 && timer) {
      clearInterval(timer)
      timer = undefined
    }
  })
  return computed(() => now.value)
}

/**
 * "lør 21.40" — the weekday and the clock time a status changed.
 *
 * The weekday is there because the race runs through a night into the next day: a scout waiting
 * "since 21.40" is a very different problem depending on whether that was four hours ago or
 * yesterday evening, and a bare time silently hides which. The elapsed span beside it answers
 * the same question, but the crew reads out and writes down the clock time, so it has to stand
 * on its own.
 *
 * Short weekday rather than "lørdag": three characters scan faster in a narrow column, and there
 * are only ever two or three distinct days in play. Some ICU versions abbreviate with a trailing
 * period ("lør.") and some do not, so it is normalised away either way — a stray period is noise
 * in a table cell, and a format that differs between the dev container and a browser is worse.
 *
 * Times render as da-DK writes them, which means a dot rather than a colon (21.40). That is the
 * Danish convention and matches the rest of the SPA (see composables/sos.ts).
 */
export const formatClock = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const weekday = date.toLocaleDateString('da-DK', { weekday: 'short' }).replace(/\.$/, '')
  const time = date.toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
  return `${weekday} ${time}`
}

/**
 * "2t 14m" — how long ago, in Danish.
 *
 * Minutes below the hour, hours and minutes above it. Days are deliberately not a unit: this
 * screen exists for one night, and "1d 3t" would mean something had gone very wrong — in which
 * case a large hour count is the clearer alarm.
 *
 * A future timestamp (clock skew between server and browser) reads as "0m" rather than a
 * negative span. It is not worth explaining a machine's disagreement to a volunteer at 3am.
 */
export const formatElapsed = (value: string | undefined, nowMs: number) => {
  if (!value) return ''
  const then = new Date(value).getTime()
  if (Number.isNaN(then)) return ''
  const minutes = Math.max(0, Math.floor((nowMs - then) / 60_000))
  if (minutes < 60) return `${minutes}m`
  return `${Math.floor(minutes / 60)}t ${minutes % 60}m`
}

/** "siden lør 21.40 (2t 14m)", the two together, as the rows show it. */
export const formatSince = (value: string | undefined, nowMs: number) => {
  const clock = formatClock(value)
  if (!clock) return ''
  return `siden ${clock} (${formatElapsed(value, nowMs)})`
}

/**
 * "lør 21.40 (2t 14m)" — the same two facts without the "siden" prefix.
 *
 * For a column whose header already says what the timestamp is ("Ankommet"). "Ankommet: siden
 * lør 21.40" reads as a mistake, and the elapsed span is still what tells the crew how long the
 * scout has been asleep.
 */
export const formatAt = (value: string | undefined, nowMs: number) => {
  const clock = formatClock(value)
  if (!clock) return ''
  return `${clock} (${formatElapsed(value, nowMs)})`
}

/** Minutes since a timestamp, for deciding whether something has gone on too long. */
export const minutesSince = (value: string | undefined, nowMs: number) => {
  if (!value) return 0
  const then = new Date(value).getTime()
  if (Number.isNaN(then)) return 0
  return Math.max(0, Math.floor((nowMs - then) / 60_000))
}

/**
 * How long a scout may wait before the row is highlighted.
 *
 * The real threshold is still unsettled (PRD 006 §11, task 082) and belongs to the dispatch
 * dashboard, not here. 45 minutes is a placeholder chosen to be useful rather than correct: it
 * highlights the handful of rows worth asking about without lighting up the whole table.
 *
 * Deliberately one constant in one place, so adopting the real value is an edit here and
 * nowhere else.
 */
export const WAITING_ALARM_MINUTES = 45
