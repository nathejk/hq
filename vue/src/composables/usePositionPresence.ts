// Telemetry presence: does this person's phone report positions, and when did it last (PRD 011)?
//
// # Why one composable rather than a field on every payload
//
// HQ shows a small glyph next to a person's name wherever people are listed. The obvious
// implementation adds `hasPosition`/`lastSeenAt` to every one of those endpoints; this one asks a
// single small endpoint which ids have ever reported and lets each row look itself up. That works
// because a `personId` is either a memberID or a crewmemberID — opaque, non-colliding — and every
// people-list row already carries the id it needs, so nothing else changes shape and a new list of
// people gets the glyph for free.
//
// Like `useKort`, this is the only way to read presence: the useLiveResource cache is keyed by
// string, so inlining the same fetch in eight views means eight chances to write a different key or
// a different `dependsOn`, and both failures are silent — a mismatched key gives each view its own
// copy, a wrong token gives a page that looks live and never updates.

import { computed, onScopeDispose, ref } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from './useLiveResource'

/** One person's presence, as `GET /api/telemetry/presence` reports it. */
export interface PresenceEntry {
  /**
   * Epoch **milliseconds** of the most recent reported position.
   *
   * Milliseconds, not seconds: this is the producer's unit, stored and served unchanged, so no
   * conversion happens anywhere between the phone and this file. Note `scan.uts` is *seconds* — the
   * two units meet on the track map, and only there (task 149).
   */
  ts: number

  /**
   * The role held when that position was reported — `gøgler`, `spejder`, `crewmember`, …
   *
   * Advisory. The glyph does not branch on it; it is here because a role can change while the
   * history is permanent, so this says what the person was then rather than what they are now.
   */
  personType: string
}

/** The endpoint's payload: an object keyed by personId, so a lookup needs no client-side index. */
export type Presence = Record<string, PresenceEntry>

/**
 * How long after a person's last position the glyph is shown as stale.
 *
 * 30 minutes, and the reasoning is worth keeping because both directions are tempting. Sampling is
 * ~30 s, so a *tight* threshold looks defensible — but gaps of an hour are entirely normal here
 * (phones lock, apps get killed, batteries die), so a tight threshold would leave most glyphs muted
 * most of the time and the state would stop carrying information. Too loose and "recent" stops
 * meaning anything on a race night.
 *
 * PRD 011 §11 leaves the value open pending real data; this is a defensible default in one place,
 * not a decision. Whatever it becomes, the muted state means **"we do not know"**, never "something
 * is wrong" — a phone in a pocket on battery-saver is indistinguishable from a phone at the bottom
 * of a lake, and this must not be read as a safety signal.
 */
export const STALE_AFTER_MS = 30 * 60 * 1000

/**
 * Shared presence for the whole app.
 *
 * `dependsOn: ['track']` — the **event subject's** entity token, from
 * `TELEMETRY.{year}.track.{personId}.reported`. Not `position`, not `telemetry`, and not the
 * projection's table names: the token is what `live.SignalFromSubject` derives, and a wrong one
 * fails silently. A *type* dependency rather than instances, because the whole point is that a
 * person who has never reported before appears — and their id was by definition never seen.
 */
export function usePositionPresence() {
  const { data, pending, error } = useLiveResource<Presence>(
    'telemetry:presence',
    async () => {
      const response = await http.get('/telemetry/presence')
      return response.data.presence ?? {}
    },
    { dependsOn: ['track'] },
  )

  const presence = computed<Presence>(() => data.value ?? {})

  /**
   * Has this person ever reported a position?
   *
   * Returns false while presence is still loading, which callers must respect: the *absence* of a
   * glyph is meaningful, so it must not be rendered before the answer is known. `loading` is
   * exposed for exactly that reason.
   */
  function hasPosition(personId: string | undefined | null): boolean {
    if (!personId) return false
    return personId in presence.value
  }

  /** Epoch ms of the last reported position, or undefined if this person has never reported. */
  function lastSeenAt(personId: string | undefined | null): number | undefined {
    if (!personId) return undefined
    return presence.value[personId]?.ts
  }

  /** The role held when the last position was reported, if any. */
  function personType(personId: string | undefined | null): string | undefined {
    if (!personId) return undefined
    return presence.value[personId]?.personType
  }

  return {
    presence,
    /** True only when nothing is cached yet — so a revisited page does not re-hide its glyphs. */
    loading: pending,
    error,
    hasPosition,
    lastSeenAt,
    personType,
  }
}

/**
 * A shared clock, ticking once a minute.
 *
 * Needed because staleness is a function of *now*, not of the data: a page left open overnight with
 * no new positions would otherwise keep rendering every glyph as fresh, which is the one failure
 * this feature must not have — claiming recency it does not have is worse than saying nothing. Live
 * signals refresh the data, but silence produces no signal, and silence is exactly the thing being
 * displayed.
 *
 * One interval for the whole app, reference-counted, so a table of 200 rows does not start 200
 * timers. A minute is the right granularity: the tooltip is rounded to minutes anyway, and the
 * threshold is half an hour.
 */
const now = ref(Date.now())
let tickers = 0
let timer: ReturnType<typeof setInterval> | undefined

export function useNow() {
  tickers += 1
  if (!timer) {
    timer = setInterval(() => {
      now.value = Date.now()
    }, 60_000)
  }
  onScopeDispose(() => {
    tickers -= 1
    if (tickers === 0 && timer) {
      clearInterval(timer)
      timer = undefined
    }
  })
  return now
}

/** Whether a timestamp counts as stale. Exported so the indicator and any future list share one rule. */
export function isStale(ts: number, now = Date.now()): boolean {
  return now - ts > STALE_AFTER_MS
}

/**
 * "14:32" — the clock time of a position, in Danish convention.
 *
 * Date-less on purpose: during a race every operator is reasoning within the same 30 hours, and a
 * full date in a tooltip is noise. The relative half of the tooltip disambiguates a timestamp that
 * is genuinely old.
 */
export function formatClock(ts: number): string {
  return new Date(ts).toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
}

/**
 * "for 6 minutter siden" — how long ago, in Danish.
 *
 * Hand-rolled rather than `Intl.RelativeTimeFormat` because the vocabulary is small and the output
 * needs to read naturally in a tooltip next to an absolute time. Coarse by design: nobody needs
 * seconds-precision staleness, and "for 2 timer siden" is more useful than "for 127 minutter siden".
 */
export function formatRelative(ts: number, now = Date.now()): string {
  const seconds = Math.round((now - ts) / 1000)

  // A phone with a slightly fast clock, or a point that arrived while the page sat open, can be
  // marginally in the future. Saying "for -1 minutter siden" would look broken over something
  // meaningless.
  if (seconds < 60) return 'lige nu'

  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `for ${minutes} ${minutes === 1 ? 'minut' : 'minutter'} siden`

  const hours = Math.round(minutes / 60)
  if (hours < 24) return `for ${hours} ${hours === 1 ? 'time' : 'timer'} siden`

  const days = Math.round(hours / 24)
  return `for ${days} ${days === 1 ? 'dag' : 'dage'} siden`
}
