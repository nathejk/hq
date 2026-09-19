<script setup lang="ts">
// One unit's duty roster as a multi-range slider across the whole race.
//
// # Why this is hand-built rather than a slider library
//
// The obvious candidate is `vue-slider-component`, which does have N handles and a `process`
// callback for marking the bars between pairs. It was not used, for reasons that are specific
// rather than aesthetic:
//
//   * Its Vue 3 support exists only on the `next` tag — `4.1.0-beta.7`, last published in 2022,
//     with `vue-property-decorator` (Vue 2 class components) still in its dependencies. The
//     `latest` tag is the Vue 2 line. Pinning a three-year-old beta for a screen that runs the
//     race is a poor trade against ~150 lines here.
//   * Its handles are one flat ordered array. Preventing a dragged handle from reordering the
//     array means `order: false`, which switches off crossing prevention wholesale — so the
//     per-pair constraints would have to be written anyway, and they are the only hard part.
//   * The things this needs that a generic slider does not have: a bar draggable *as a whole*
//     (moving a shift is a different instruction from resizing it), its times written inside it,
//     a minimum bar width so a 15-minute window stays grabbable on a 72-hour track, and a now
//     marker underneath the bars.
//
// The arithmetic all lives in `composables/dutyTimeline`, under test. What is here is pointer
// plumbing, and it is kept deliberately dull.
//
// # Condensed: the times are in the bar
//
// They used to be a list of chips under the track, which cost a second line per unit and put the
// reading of a shift somewhere other than the shift. Inside the bar there is often not room, so the
// label is clipped and the full one appears on hover — a shift you can read at a glance when it is
// long, and on demand when it is short, in one line per unit either way.
//
// # Pairs are appended, never reordered
//
// A knot carries the id of the duty window it edits. Nothing here sorts the pairs: a slider that
// reordered its handles when one was dragged past another would move a *different* shift, which
// would be both silent and plausible. New periods go at the end of the list and at the end of the
// timeline. Overlaps are allowed, as the API allows them.
//
// # It does not write
//
// It emits. The board owns the PUT, the toast and the `saving` flag that pauses live payloads
// (PRD 004).

import { computed, ref } from 'vue'
import type { Duty } from '@/composables/dispatch'
import { formatUts } from '@/composables/dispatch'
import {
  type Axis,
  type DutyDay,
  barLabel,
  dragKnot,
  dragPair,
  pairsOn,
  percentOf,
  spanLabel,
  utsAt,
} from '@/composables/dutyTimeline'

const props = defineProps<{
  windows: Duty[]
  axis: Axis
  /** Midnights, for the gridlines inside the track. The labels are drawn once by the roster, above
   *  every row; the lines are per row because judging whether a shift covers the small hours means
   *  reading it against the night, not against a caption three rows up. */
  days: DutyDay[]
  /** Where now is on the axis, or null when now is outside the race. Passed in rather than computed
   *  here so every row's marker is the same instant — one clock, not one per unit. */
  nowPercent?: number | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  save: [{ dutyId: string; startUts: number; endUts: number }]
  remove: [string]
  /** A knot or bar is being held. The board pauses live payloads while it is true: a gesture in
   *  progress is unsaved state, and the house rule (PRD 004) is not to redraw underneath one. */
  dragging: [boolean]
}>()

const track = ref<HTMLElement | null>(null)

/** The gesture in progress. Null between gestures — the bars then show what the payload says. */
const drag = ref<{
  dutyId: string
  /** Which knot, or the bar itself. */
  grab: 'start' | 'end' | 'bar'
  startUts: number
  endUts: number
  /** Where in the bar it was grabbed, as seconds from its start: without this a bar jumps so its
   *  start lands under the cursor, which on a 72-hour axis throws the shift hours off. */
  offsetSeconds: number
} | null>(null)

/** The bar under the cursor, which is the one whose full label is shown. */
const hovered = ref<string | null>(null)

/** What to draw: the payload, with the dragged window replaced by where it is being dragged to. */
const pairs = computed(() =>
  pairsOn(
    props.windows.map((w) =>
      drag.value && drag.value.dutyId === w.id
        ? { ...w, startUts: drag.value.startUts, endUts: drag.value.endUts }
        : w,
    ),
    props.axis,
  ),
)

const utsFromEvent = (event: PointerEvent): number | null => {
  const element = track.value
  if (!element) return null
  const box = element.getBoundingClientRect()
  if (box.width <= 0) return null
  // Against the track's own box, so zoom and horizontal scroll need no arithmetic of their own:
  // both change the box, and the pointer is measured in the same coordinates.
  return utsAt(((event.clientX - box.left) / box.width) * 100, props.axis)
}

const onPointerDown = (duty: Duty, grab: 'start' | 'end' | 'bar', event: PointerEvent) => {
  if (props.disabled) return
  event.preventDefault()
  const at = utsFromEvent(event)
  drag.value = {
    dutyId: duty.id,
    grab,
    startUts: duty.startUts,
    endUts: duty.endUts,
    offsetSeconds: at === null ? 0 : Math.max(at - duty.startUts, 0),
  }
  // Captured on the element that was grabbed, so the gesture survives the cursor leaving the
  // track — a drag to the end of the axis inevitably does.
  ;(event.target as HTMLElement).setPointerCapture?.(event.pointerId)
  emit('dragging', true)
}

const onPointerMove = (event: PointerEvent) => {
  const gesture = drag.value
  if (!gesture) return
  const at = utsFromEvent(event)
  if (at === null) return
  const next =
    gesture.grab === 'bar'
      ? dragPair(gesture, at - gesture.offsetSeconds, props.axis)
      : dragKnot(gesture, gesture.grab, at, props.axis)
  drag.value = { ...gesture, ...next }
}

/**
 * Commit on release, not while moving.
 *
 * One PUT per quarter-hour crossed would be dozens of writes for one drag, each of them a
 * projection, a live signal and a revalidation of every open board.
 */
const onPointerUp = () => {
  const gesture = drag.value
  drag.value = null
  if (!gesture) return
  emit('dragging', false)
  const before = props.windows.find((w) => w.id === gesture.dutyId)
  // A click that moved nothing, or a drag and a change of mind: a no-op PUT still costs every open
  // board a revalidation.
  if (!before || (before.startUts === gesture.startUts && before.endUts === gesture.endUts)) return
  emit('save', { dutyId: gesture.dutyId, startUts: gesture.startUts, endUts: gesture.endUts })
}

/**
 * Arrow keys move a knot by one step, so the roster is reachable without a mouse.
 *
 * Not a nicety on this screen: the operator is often on a laptop in a tent with no room for a
 * mouse, and a 15-minute adjustment by key is more precise than aiming at two pixels anyway.
 */
const onKey = (duty: Duty, grab: 'start' | 'end', event: KeyboardEvent) => {
  if (props.disabled) return
  const step = event.shiftKey ? 3600 : 15 * 60
  const delta = event.key === 'ArrowLeft' ? -step : event.key === 'ArrowRight' ? step : 0
  if (!delta) return
  event.preventDefault()
  const from = grab === 'start' ? duty.startUts : duty.endUts
  const next = dragKnot(duty, grab, from + delta, props.axis)
  if (next.startUts === duty.startUts && next.endUts === duty.endUts) return
  emit('save', { dutyId: duty.id, ...next })
}

/** What a knot reads as to a screen reader: its own edge, matching `aria-valuenow`. The whole span
 *  belongs on the bar, not on one of its two handles. */
const knotLabel = (duty: Duty, grab: 'start' | 'end') =>
  `${grab === 'start' ? 'Fra' : 'Til'} ${formatUts(grab === 'start' ? duty.startUts : duty.endUts)}`

/** The full label is shown for the bar being hovered, and for the one being dragged — during a drag
 *  it is the number being chosen, and it has to be legible without being clipped by its own bar. */
const showFullLabel = (dutyId: string) => hovered.value === dutyId || drag.value?.dutyId === dutyId

/** Left offset of a midnight, for the gridlines. */
const dayPercent = (day: DutyDay) => percentOf(day.startUts, props.axis)
</script>

<template>
  <div
    ref="track"
    data-track
    class="duty-track relative h-full select-none"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
  >
    <!-- The track's own stripe, as an element rather than a background on the track: the now marker
         has to sit above it and below the bars, and explicit layers in one parent are the only way to
         say that which does not depend on what an ancestor happens to paint. See the style block. -->
    <div class="duty-bg absolute inset-x-0 inset-y-1 rounded" :class="{ 'duty-bg-disabled': disabled }" />

    <div
      v-for="day in days"
      :key="day.key"
      class="duty-gridline absolute top-0 bottom-0 w-px"
      :style="{ left: `${dayPercent(day)}%` }"
    />

    <!--
      Now, under the bars.

      Drawn per row rather than once across the table so that it is genuinely *behind* the shifts:
      between the track and the bars there is no z-index a single element spanning all rows could
      take. The rows are contiguous, so it reads as one line through the roster — which is the point
      of it. See the stacking note in the style block.
    -->
    <div
      v-if="nowPercent !== null && nowPercent !== undefined"
      data-now
      class="duty-now absolute top-0 bottom-0"
      :style="{ left: `${nowPercent}%` }"
    />

    <template v-for="pair in pairs" :key="pair.duty.id">
      <!-- The bar: draggable as a whole, because moving a shift is its own instruction. The times
           live in it; `overflow-hidden` clips them when the shift is too short to hold them, and the
           hover card below says the whole thing. -->
      <div
        data-bar
        class="duty-bar absolute top-1 bottom-1 overflow-hidden rounded px-1 flex items-center"
        :class="[
          disabled ? 'cursor-default' : 'cursor-grab',
          drag?.dutyId === pair.duty.id ? 'duty-bar-active' : '',
          pair.clipped ? 'duty-bar-clipped' : '',
        ]"
        :style="{ left: `${pair.fromPercent}%`, width: `${pair.widthPercent}%` }"
        @pointerdown="onPointerDown(pair.duty, 'bar', $event)"
        @pointerenter="hovered = pair.duty.id"
        @pointerleave="hovered = null"
      >
        <span data-bar-label class="duty-bar-label whitespace-nowrap tabular-nums">
          {{ barLabel(pair.duty) }}
        </span>
      </div>

      <!--
        The full label, outside the bar so nothing clips it.

        This is what makes the condensed row honest: a two-hour shift on a three-day axis cannot show
        its own times, and an operator who cannot read a time cannot check a roster. It carries the
        weekday (which the in-bar label drops) and the remove button, which had nowhere else to go
        once the chips were gone.
      -->
      <div
        v-if="showFullLabel(pair.duty.id)"
        data-bar-card
        class="duty-card absolute top-0 z-30 flex items-center gap-1 whitespace-nowrap rounded px-1.5 text-xs tabular-nums"
        :style="{ left: `${pair.fromPercent}%` }"
        @pointerenter="hovered = pair.duty.id"
        @pointerleave="hovered = null"
      >
        <i v-if="pair.clipped" class="pi pi-exclamation-triangle text-[10px] text-amber-600" />
        {{ spanLabel(pair.duty) }}
        <button
          type="button"
          data-remove
          class="duty-remove pi pi-times"
          :disabled="disabled"
          :aria-label="`Fjern vagten ${spanLabel(pair.duty)}`"
          title="Fjern vagten"
          @pointerdown.stop
          @click.stop="emit('remove', pair.duty.id)"
        />
      </div>

      <!-- The two knots. Wider than they look: on a three-day axis the difference between grabbing
           an edge and grabbing the bar is a couple of pixels, so the hit area is padded beyond the
           visible handle. -->
      <button
        v-for="grab in (['start', 'end'] as const)"
        :key="grab"
        data-knot
        type="button"
        class="duty-knot-hit absolute top-0 bottom-0 flex w-3 -ml-1.5 items-center justify-center focus:outline-none"
        :class="disabled ? 'cursor-default' : 'cursor-ew-resize'"
        :style="{
          left: `${grab === 'start' ? pair.fromPercent : pair.fromPercent + pair.widthPercent}%`,
        }"
        role="slider"
        :aria-valuemin="axis.startUts"
        :aria-valuemax="axis.endUts"
        :aria-valuenow="grab === 'start' ? pair.duty.startUts : pair.duty.endUts"
        :aria-valuetext="knotLabel(pair.duty, grab)"
        :aria-label="knotLabel(pair.duty, grab)"
        :disabled="disabled"
        @pointerdown="onPointerDown(pair.duty, grab, $event)"
        @keydown="onKey(pair.duty, grab, $event)"
      >
        <span
          class="duty-knot h-5 w-1.5 rounded-sm"
          :class="{ 'duty-knot-active': drag?.dutyId === pair.duty.id }"
        />
      </button>
    </template>
  </div>
</template>

<style scoped>
/*
 * # Colours
 *
 * Taken from the PrimeVue theme *by its own variable names*, with a literal fallback on every one.
 *
 * Not belt and braces — this control was invisible in the browser while every test passed. Tailwind
 * is configured (`tailwind.config.js`) with `--surface-200` / `--primary-400`, which are PrimeVue
 * **3** token names; PrimeVue 4 emits them prefixed (`--p-surface-200`, see `prefix: "p"` in
 * @primeuix/styled). So a utility class like bg-surface-* compiles to `rgb(var(--surface-200))`
 * against a variable nothing defines, and resolves to transparent. A track, a bar and two knots all
 * rendered, all correct, all invisible.
 *
 * Hence the fallbacks: a slider nobody can see is not degraded, it is broken, so it must not depend
 * on a variable name being right. The literals are the emerald the rest of the app uses for "on
 * time" (see the postoverblik meters).
 *
 * The same dead tokens are used elsewhere in the app (text-primary-* and text-surface-* in several
 * views) and are equally inert there — worth fixing, but not from here.
 *
 * # Stacking, which is load-bearing
 *
 * The now marker has to sit *above* the track and *below* the bars, so the track's stripe is a
 * positioned element with an explicit z-index rather than a background on the track itself.
 *
 * The tempting alternative — painting the stripe from a `::before` at `z-index: -1` — works until it
 * does not: a negative-z child is painted below the background of the nearest ancestor *stacking
 * context*, which inside a PrimeVue Dialog is several elements up, and any opaque background in
 * between would swallow the stripe. Four explicit layers in one parent cannot be undone by an
 * ancestor.
 */
.duty-bg {
  z-index: 0;
  background: var(--p-surface-200, #e5e7eb);
}

.duty-bg-disabled {
  background: var(--p-surface-100, #f3f4f6);
}

.duty-gridline {
  z-index: 1;
  background: var(--p-surface-400, #9ca3af);
}

/* Now. Orange because nothing else on this screen is: it is the one mark that is not a decision
   somebody made, and it must not be mistaken for a shift. Under the bars, so it cannot obscure the
   thing being read. */
.duty-now {
  z-index: 2;
  width: 2px;
  margin-left: -1px;
  background: #f97316;
}

.duty-bar {
  z-index: 3;
  background: var(--p-primary-400, #34d399);
}

.duty-bar:hover {
  background: var(--p-primary-500, #10b981);
}

.duty-bar-active {
  background: var(--p-primary-500, #10b981);
  box-shadow: 0 0 0 2px var(--p-primary-700, #047857);
}

/* A shift that falls outside the race can only be drawn clamped to an edge. Marked rather than
   hidden: the hours are real and somebody has to deal with them. */
.duty-bar-clipped {
  outline: 1px solid #f59e0b;
}

.duty-bar-label {
  font-size: 10px;
  line-height: 1;
  color: #fff;
}

.duty-card {
  background: #fff;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.25);
  border: 1px solid var(--p-surface-300, #d1d5db);
  /* Lifted clear of its own bar, so the bar it describes stays visible while being dragged. */
  transform: translateY(-115%);
}

.duty-remove {
  font-size: 10px;
  color: #dc2626;
  padding: 0.125rem;
}

.duty-remove:disabled {
  color: var(--p-surface-400, #9ca3af);
}

.duty-knot-hit {
  z-index: 4;
}

.duty-knot {
  background: var(--p-primary-700, #047857);
  /* Against the bar as well as against the track: a knot that vanishes into its own bar cannot be
     aimed at, which is the whole gesture. */
  box-shadow: 0 0 0 1px #fff;
}

.duty-knot-active {
  box-shadow: 0 0 0 2px #fff;
}

/* Keyboard focus has to be visible on the knot itself: the button is a hit area with no box of its
   own, so the browser's default outline lands nowhere useful. */
[data-knot]:focus-visible .duty-knot {
  box-shadow: 0 0 0 2px #fff, 0 0 0 4px var(--p-primary-700, #047857);
}
</style>
