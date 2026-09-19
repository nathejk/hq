<script setup lang="ts">
// The vagter roster (PRD 009 §6): one row per dispatch unit, one shared timeline.
//
// # Why one axis, and why one row per unit
//
// The roster is read as a comparison. The question it answers is never "when is Bil 1 on?" but
// **"who covers 02:00?"**, so every unit's shifts have to sit on the *same* axis, spanning the whole
// race. A column per day split every night shift into two bars in two cells, putting a seam down the
// middle of the thing being judged — and the seam fell exactly where the race is hardest to staff.
//
// The layout is condensed for the same reason: the value is in seeing *many* units at once, so a
// unit costs one line. The times live inside their bars, the day scale is drawn once above all rows
// instead of once per row, and `+ Vagt` sits beside the timeline rather than under it.
//
// # Three columns, one scroller
//
// Names | timeline | add. The timeline is the only part that scrolls, and it is **one** scroll
// container holding every row: a scroller per row would let two units drift to different hours,
// which is precisely the comparison this screen exists to make. The other two columns are outside
// it, so a name and its add button cannot scroll away from the row they belong to.
//
// That shared scroller is also what forces fixed row heights (`.duty-row`): three separate columns
// have to agree on where each row is, and nothing but a height can make them.
//
// # Zoom
//
// The race is 72 hours. On HQ's big screen that fits; on a laptop in a tent, a quarter-hour is
// sub-pixel. Zoom is a multiple of the container width — see ZOOM_STEPS — so 1× always means "the
// whole race fits" on whatever machine is in front of the operator, and nothing needs to know about
// pixels per hour.
//
// # It does not write
//
// It emits. The board owns the PUT, the toast and the `saving` flag that pauses live payloads
// (PRD 004) — a component that wrote on its own could have its own row replaced underneath a drag
// in flight.

import { computed, ref } from 'vue'
import type { Duty, Unit } from '@/composables/dispatch'
import DutyRangeSlider from '@/components/DutyRangeSlider.vue'
import {
  eventDays,
  newPeriod,
  nowPercent,
  percentOf,
  rosteredHours,
  uncoveredHours,
  zoomBy,
  ZOOM_STEPS,
} from '@/composables/dutyTimeline'

const props = defineProps<{
  units: Unit[]
  duty: Duty[]
  /** The race's span, from the board payload. Absent or unset means the year has no dates yet,
   *  which the roster reports rather than drawing a guessed axis. */
  event?: { startUts: number; endUts: number }
  /** True while a write is in flight: the controls go inert rather than queueing a second edit
   *  onto a row whose payload has not landed. */
  busy?: boolean
  /** The board's clock, for the now marker. Passed in rather than taken from `useNow` here, matching
   *  DispatchTourCard and DispatchCapacityStrip: one clock per screen, so nothing on it can disagree
   *  about what time it is. */
  nowMs?: number
}>()

const emit = defineEmits<{
  save: [{ dutyId?: string; sectionSlug: string; startUts: number; endUts: number }]
  remove: [string]
  /** Forwarded from the slider: a gesture is in progress somewhere in the roster, so the board must
   *  not redraw underneath it. */
  dragging: [boolean]
}>()

const axis = computed(() => ({
  startUts: props.event?.startUts ?? 0,
  endUts: props.event?.endUts ?? 0,
}))

const days = computed(() => eventDays(axis.value.startUts, axis.value.endUts))

/** One marker for the whole roster, not one per row: rows a few milliseconds apart would draw a
 *  ragged line, and the marker's whole job is to be one straight edge through the units. Null when
 *  now is outside the race, or when the board has not passed a clock at all. */
const nowPct = computed(() => (props.nowMs ? nowPercent(props.nowMs, axis.value) : null))

/** One row per unit, whether or not it has any windows: a unit with an empty timeline is the
 *  point — that is the gap somebody has to go and fill. */
const rows = computed(() =>
  props.units.map((unit) => ({
    unit,
    // Stored order, deliberately not sorted by time: a knot carries the id of the window it edits,
    // and the slider's pairs are positional. See DutyRangeSlider.
    windows: props.duty.filter((w) => w.sectionSlug === unit.sectionSlug),
  })),
)

/** The hours of the race no unit at all covers. The number this screen exists to drive to zero,
 *  and the reason the axis is the whole race rather than a day at a time. */
const uncovered = computed(() => uncoveredHours(props.duty, axis.value))

const addPeriod = (unit: Unit, windows: Duty[]) =>
  emit('save', { sectionSlug: unit.sectionSlug, ...newPeriod(windows, axis.value) })

// --- zoom and scroll ---

const zoom = ref<number>(ZOOM_STEPS[0])
const scroller = ref<HTMLElement | null>(null)

const canvasStyle = computed(() => ({ width: `${zoom.value * 100}%` }))

/**
 * Zoom about the middle of what is on screen.
 *
 * Anchoring matters more than it sounds: zooming from the left edge throws away the operator's place
 * on a 72-hour axis, and finding Saturday 02:00 again by scrollbar is the kind of small tax that
 * stops a screen being used during a race.
 */
const changeZoom = (direction: 1 | -1) => {
  const next = zoomBy(zoom.value, direction)
  if (next === zoom.value) return
  const element = scroller.value
  const anchor = element ? (element.scrollLeft + element.clientWidth / 2) / (element.clientWidth * zoom.value) : 0.5
  zoom.value = next
  if (!element) return
  // After the width has been applied, or the new scrollLeft is clamped against the old canvas.
  requestAnimationFrame(() => {
    element.scrollLeft = anchor * element.clientWidth * next - element.clientWidth / 2
  })
}

/** Scroll the now marker into the middle. Zoomed in, "where are we" is the first question, and the
 *  scrollbar is a poor way to answer it. */
const scrollToNow = () => {
  const element = scroller.value
  if (!element || nowPct.value === null) return
  element.scrollLeft = (nowPct.value / 100) * element.clientWidth * zoom.value - element.clientWidth / 2
}
</script>

<template>
  <div>
    <template v-if="days.length && rows.length">
      <div class="flex items-center justify-end gap-1 pb-1">
        <Button
          v-if="nowPct !== null"
          label="Nu"
          icon="pi pi-clock"
          text
          size="small"
          class="!py-0.5 mr-2"
          v-tooltip.bottom="'Rul til nu'"
          @click="scrollToNow()"
        />
        <Button
          icon="pi pi-search-minus"
          text
          size="small"
          class="!py-0.5"
          :disabled="zoom === ZOOM_STEPS[0]"
          v-tooltip.bottom="'Zoom ud'"
          aria-label="Zoom ud"
          @click="changeZoom(-1)"
        />
        <span class="w-8 text-center text-xs tabular-nums text-gray-600">{{ zoom }}×</span>
        <Button
          icon="pi pi-search-plus"
          text
          size="small"
          class="!py-0.5"
          :disabled="zoom === ZOOM_STEPS[ZOOM_STEPS.length - 1]"
          v-tooltip.bottom="'Zoom ind'"
          aria-label="Zoom ind"
          @click="changeZoom(1)"
        />
      </div>

      <div class="flex">
        <!-- Names. Outside the scroller, so a unit's label cannot scroll away from its own bars. -->
        <div class="w-36 shrink-0 pr-2" data-names>
          <div class="duty-scale" />
          <div
            v-for="row in rows"
            :key="row.unit.sectionSlug"
            class="duty-row flex items-baseline gap-1"
          >
            <span class="truncate font-medium">{{ row.unit.label }}</span>
            <small v-if="row.windows.length" class="ml-auto text-gray-500 tabular-nums">
              {{ rosteredHours(row.windows) }}t
            </small>
            <!-- Said once per row, where it can be acted on. A unit with no roster is never on duty,
                 which the board then reports as no capacity at all. -->
            <small v-else class="ml-auto text-amber-600">ingen</small>
          </div>
        </div>

        <div ref="scroller" data-scroller class="min-w-0 flex-1 overflow-x-auto">
          <div class="relative" :style="canvasStyle">
            <!-- The day scale, once for the whole roster rather than once per unit: repeating it per
                 row was a third of the vertical space and said nothing new. -->
            <div class="duty-scale relative text-[10px] text-gray-500">
              <span
                v-for="day in days"
                :key="day.key"
                class="absolute whitespace-nowrap pl-1"
                :style="{ left: `${percentOf(day.startUts, axis)}%` }"
              >
                {{ day.label }}
              </span>
            </div>

            <div
              v-for="row in rows"
              :key="row.unit.sectionSlug"
              class="duty-row"
            >
              <DutyRangeSlider
                :windows="row.windows"
                :axis="axis"
                :days="days"
                :nowPercent="nowPct"
                :disabled="busy"
                @dragging="emit('dragging', $event)"
                @remove="emit('remove', $event)"
                @save="emit('save', { sectionSlug: row.unit.sectionSlug, ...$event })"
              />
            </div>
          </div>
        </div>

        <!-- Add, immediately after the timeline and outside the scroller: every row needs it, and a
             button that scrolls out of reach at 8× is a button that is not there. New periods land at
             the end of the timeline, which is what the tooltip says. -->
        <div class="shrink-0 pl-1" data-add>
          <div class="duty-scale" />
          <div
            v-for="row in rows"
            :key="row.unit.sectionSlug"
            class="duty-row flex items-center"
          >
            <Button
              icon="pi pi-plus"
              label="Vagt"
              text
              size="small"
              class="!py-0.5 whitespace-nowrap"
              :disabled="busy"
              v-tooltip.bottom="'Ny vagt i slutningen af tidslinjen'"
              @click="addPeriod(row.unit, row.windows)"
            />
          </div>
        </div>
      </div>

      <p class="pt-2 text-sm" :class="uncovered > 0 ? 'text-amber-700' : 'text-gray-600'">
        <template v-if="uncovered > 0">
          <i class="pi pi-exclamation-triangle pr-1" />
          {{ uncovered }} timer af løbet er ingen enhed på vagt.
        </template>
        <template v-else>Hele løbet er dækket.</template>
      </p>
    </template>

    <!-- The two empty states say which thing is missing, because the fix is in a different place
         for each. -->
    <Message v-else-if="!rows.length" severity="warn" :closable="false">
      Ingen kørsels-enheder endnu. Marker en underafdeling som kørsels-enhed på Organisation.
    </Message>
    <Message v-else severity="warn" :closable="false">
      Årets datoer mangler, så der er ingen tidslinje at lægge vagter på. Sæt start- og slutdato
      under År.
    </Message>
  </div>
</template>

<style scoped>
/*
 * One height for a row, and one for the scale, shared by all three columns.
 *
 * Not styling: the columns are siblings, not cells, because the middle one has to be the only thing
 * that scrolls. Nothing then aligns a name with its own bars except both being exactly this tall, so
 * these two numbers are the layout. A row that grew with its content would shear the columns apart.
 */
.duty-row {
  height: 2rem;
}

.duty-scale {
  height: 1rem;
}
</style>
