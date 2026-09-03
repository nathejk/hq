<script setup lang="ts">
/**
 * A small glyph next to a person's name, saying whether their phone reports positions (PRD 011).
 *
 * # Three states, and one of them is nothing at all
 *
 * - **Absent** — this person has never reported. No glyph, no tooltip, no placeholder. The absence
 *   is the information, and it only works if nothing occupies the space: a greyed-out icon for
 *   "never" and one for "a while ago" would be two shades of the same thing at a glance.
 * - **Normal** — reported recently.
 * - **Muted** — reported, but not for a while (`STALE_AFTER_MS`).
 *
 * Nothing renders while presence is still loading, which is deliberate rather than tidy: since
 * absence means "never reported", showing it before the data arrives would state something false on
 * every first paint.
 *
 * # What the muted state does not mean
 *
 * It does not mean anything is wrong. Gaps of an hour are normal — phones lock, apps get killed,
 * batteries die — so a stale glyph is weak evidence of nothing at all: a phone in a pocket on
 * battery-saver looks identical to a phone at the bottom of a lake. This answers "can I expect
 * location data from this person?" and nothing more, and the wording is chosen so it cannot be read
 * as a safety signal.
 */
import { computed } from 'vue'
import {
  formatClock,
  formatRelative,
  isStale,
  useNow,
  usePositionPresence,
} from '@/composables/usePositionPresence'
import { useTrackViewer } from '@/composables/useTrackViewer'

const props = defineProps<{
  /**
   * The person's id — a memberID for a spejder or senior, a userId for crew, gøgler, friend or
   * bandit. Both spaces are opaque and do not collide, so the caller passes whichever id its row
   * already has and needs no type hint.
   */
  personId: string | undefined | null

  /**
   * The team this person is a member of, when the caller knows it.
   *
   * This is what decides which map opens: with a team, clicking shows the **patrol's** track — every
   * member who has ever been on it plus its scans — because for a spejder the patrol is the unit that
   * matters. Without one, it shows this person's own track, which is the right answer for crew and
   * gøglere. The rule itself lives in `useTrackViewer`, so a call site only states what it knows.
   */
  teamId?: string | null

  /** A name for the dialog's heading, when the caller has one. */
  label?: string | null

  /**
   * Show the timestamp as text beside the glyph instead of only in a tooltip.
   *
   * For detail views, where there is room and where the timestamp should not require a hover to
   * reach — a tooltip is unreachable by touch and awkward by keyboard.
   */
  showText?: boolean
}>()

const { hasPosition, lastSeenAt, loading } = usePositionPresence()
const { show } = useTrackViewer()
const now = useNow()

const ts = computed(() => lastSeenAt(props.personId))

// Loading is checked as well as presence: `hasPosition` is false both for "never reported" and for
// "not known yet", and those must not render the same way.
const visible = computed(() => !loading.value && hasPosition(props.personId) && ts.value !== undefined)

const stale = computed(() => (ts.value === undefined ? false : isStale(ts.value, now.value)))

const label = computed(() => {
  if (ts.value === undefined) return ''
  return `Sidst set ${formatClock(ts.value)} · ${formatRelative(ts.value, now.value)}`
})

const title = computed(() => `${label.value} — klik for at se spor`)

function openTrack() {
  if (!props.personId) return
  show(props.personId, {
    teamId: props.teamId ?? undefined,
    label: props.label ?? undefined,
  })
}
</script>

<template>
  <!--
    aria-label carries the whole sentence, so a screen reader gets the timestamp rather than the
    fact that there is an icon. `pi-map-marker` and v-tooltip follow SosTeamCard's inline-glyph
    pattern rather than introducing FontAwesome here.

    Sized to the surrounding text (text-xs, no margins of its own beyond a small gap) so dropping it
    into a dense DataTable cell cannot shift a row's height.

    A <button> rather than a clickable <i>: this opens a dialog, so it must be reachable by keyboard
    and announced as an action. `type="button"` because these sit inside forms in some views, and a
    default-type button there would submit. `@click.stop` because several call sites put this next to
    a name that is itself a row-opening control — clicking the glyph should not also open the member.
  -->
  <span v-if="visible" class="inline-flex items-center gap-1 align-baseline">
    <button
      type="button"
      class="inline-flex items-center bg-transparent border-0 p-0 cursor-pointer leading-none"
      v-tooltip.top="title"
      :aria-label="title"
      @click.stop="openTrack"
    >
      <i
        class="pi pi-map-marker text-xs"
        :class="stale ? 'text-gray-400' : 'text-gray-600'"
        aria-hidden="true"
      />
    </button>
    <span v-if="showText" class="text-xs" :class="stale ? 'text-gray-400' : 'text-gray-600'">
      {{ label }}
    </span>
  </span>
</template>
