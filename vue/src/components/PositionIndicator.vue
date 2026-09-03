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

const props = defineProps<{
  /**
   * The person's id — a memberID for a spejder or senior, a userId for crew, gøgler, friend or
   * bandit. Both spaces are opaque and do not collide, so the caller passes whichever id its row
   * already has and needs no type hint.
   */
  personId: string | undefined | null

  /**
   * Show the timestamp as text beside the glyph instead of only in a tooltip.
   *
   * For detail views, where there is room and where the timestamp should not require a hover to
   * reach — a tooltip is unreachable by touch and awkward by keyboard.
   */
  showText?: boolean
}>()

const { hasPosition, lastSeenAt, loading } = usePositionPresence()
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
</script>

<template>
  <!--
    aria-label carries the whole sentence, so a screen reader gets the timestamp rather than the
    fact that there is an icon. `pi-map-marker` and v-tooltip follow SosTeamCard's inline-glyph
    pattern rather than introducing FontAwesome here.

    Sized to the surrounding text (text-xs, no margins of its own beyond a small gap) so dropping it
    into a dense DataTable cell cannot shift a row's height.
  -->
  <span v-if="visible" class="inline-flex items-center gap-1 align-baseline">
    <i
      class="pi pi-map-marker text-xs"
      :class="stale ? 'text-gray-400' : 'text-gray-600'"
      v-tooltip.top="label"
      :aria-label="label"
      role="img"
    />
    <span v-if="showText" class="text-xs" :class="stale ? 'text-gray-400' : 'text-gray-600'">
      {{ label }}
    </span>
  </span>
</template>
