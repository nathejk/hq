<script setup lang="ts">
// The bingo curve: how many patruljer still hold a full card, across the night.
//
// A bingo-patrulje has started, has been through every obligatorisk postlinje inside its
// opening hours, and has never been caught by a bandit. The graph is the *field thinning
// out*: everyone starts with a clean card, a post closes and the teams that did not make
// it lose theirs, a bandit catches two more, and whoever is left on Sunday morning is the
// answer. See go/cmd/api/bingo.go for where each team leaves the curve.
//
// Hand-drawn SVG rather than a charting library: this is one step line against a time
// axis, and chart.js — the peer dependency PrimeVue's Chart needs — is a bigger download
// than the rest of the dashboard put together. It also has to be talked out of
// interpolating between points, which for a step function is the one thing that matters.

import { computed, ref } from 'vue'
import { dddhhmm, hhmm } from '@/composables/datefilters'
import { areaPath, countAt, hourTicks, niceMax, stepPath, tickStepHours, type BingoSeries } from '@/composables/bingo'

const props = defineProps<{
  series: BingoSeries
  loading?: boolean
}>()

// The graph answers "how many"; the table behind this answers "which, and where did it go
// wrong". Owned by the parent so this component stays presentation only, and so the dialog's
// resource is not created until somebody asks for it.
const emit = defineEmits<{ 'open-teams': [] }>()

// A fixed user-space canvas scaled by the browser, so the component needs no width
// measurement and no resize observer. Wide and short because the x-axis carries thirty
// hours and the y-axis at most a hundred teams.
const W = 900
const H = 260
const plot = { left: 34, right: W - 16, top: 16, bottom: H - 30 }

/** Is there a window to plot against at all? A year still being planned has none. */
const hasAxis = computed(() => props.series.untilUts > props.series.fromUts)

/**
 * Zero obligatoriske postlinjer is the trap this guards.
 *
 * With nothing flagged obligatorisk there is nothing to fail, so every started team reads
 * as bingo and the curve is a flat line at the number of starters — a graph that looks
 * like it works and means nothing. Said out loud instead of drawn.
 */
const hasCriteria = computed(() => props.series.checkgroupCount > 0)

const yMax = computed(() => niceMax(Math.max(props.series.startedCount, ...props.series.points.map((p) => p.count))))

const scale = computed(() => {
  const { fromUts, untilUts } = props.series
  const span = Math.max(1, untilUts - fromUts)
  return {
    x: (uts: number) => plot.left + ((uts - fromUts) / span) * (plot.right - plot.left),
    y: (count: number) => plot.bottom - (count / yMax.value) * (plot.bottom - plot.top)
  }
})

const line = computed(() => stepPath(props.series.points, scale.value))
const area = computed(() => areaPath(props.series.points, scale.value, plot.bottom))

const weekday = new Intl.DateTimeFormat('da-DK', { weekday: 'short' })

const xTicks = computed(() => {
  const { fromUts, untilUts } = props.series
  return hourTicks(fromUts, untilUts, tickStepHours(fromUts, untilUts)).map((uts) => {
    const at = new Date(uts * 1000)
    return {
      uts,
      x: scale.value.x(uts),
      label: hhmm(at),
      // The day is named only where it changes — at midnight — so the axis reads
      // "21.00 … 00.00 lør … 04.00" rather than repeating "fre" eight times.
      day: at.getHours() === 0 ? weekday.format(at) : ''
    }
  })
})

const dayTicks = computed(() => xTicks.value.filter((t) => t.day))

/** Four gridlines plus the baseline: enough to read a count off, few enough to stay quiet. */
const yTicks = computed(() => {
  const step = yMax.value / 4
  return [0, 1, 2, 3, 4].map((i) => ({ count: Math.round(i * step), y: scale.value.y(i * step) }))
})

/** Where the line stops, while the race is still running. Nothing to mark once it is over. */
const nowX = computed(() => {
  const { nowUts, fromUts, untilUts } = props.series
  if (nowUts <= fromUts || nowUts >= untilUts) return null
  return scale.value.x(nowUts)
})

/** The count in force right now — the headline number above the graph. */
const currentCount = computed(() => countAt(props.series.points, props.series.nowUts) ?? 0)

// Hover readout. An SVG overlay rather than a tooltip component: the question is "how
// many, and when", and both numbers are already in the series.
const hover = ref<{ x: number; uts: number; count: number } | null>(null)

function onMove(event: MouseEvent) {
  const box = (event.currentTarget as SVGSVGElement).getBoundingClientRect()
  if (box.width === 0) return
  // Back out of the browser's scaling into the fixed user space the paths live in.
  const x = ((event.clientX - box.left) / box.width) * W
  const { fromUts, untilUts } = props.series
  if (x < plot.left || x > plot.right) {
    hover.value = null
    return
  }
  const uts = Math.round(fromUts + ((x - plot.left) / (plot.right - plot.left)) * (untilUts - fromUts))
  const count = countAt(props.series.points, uts)
  // Undefined before the curve starts: it makes no claim about a time it does not cover.
  hover.value = count === undefined ? null : { x: scale.value.x(uts), uts, count }
}
</script>

<template>
  <div class="card font-nathejk">
    <div class="flex flex-wrap items-baseline justify-between gap-2 mb-2">
      <div>
        <span class="block uppercase text-teal-600">Bingo-patruljer</span>
        <span class="text-xs text-gray-500"> Stadig til tiden på alle obligatoriske postlinjer, der er lukket, og aldrig fanget af banditter </span>
        <!--
          A real button, not a styled span: this opens a dialog, so it has to be reachable by
          keyboard and announced as an action.
        -->
        <button type="button" class="block mt-1 text-sm text-teal-700 underline bg-transparent border-0 p-0 cursor-pointer" @click="emit('open-teams')">Se alle patruljer og postlinjer</button>
      </div>
      <div v-if="hasAxis && hasCriteria" class="text-right">
        <span class="block text-4xl font-bold text-teal-600">{{ currentCount }}</span>
        <span class="text-xs text-gray-500">af {{ series.startedCount }} startede</span>
      </div>
    </div>

    <!--
      The three empty states are different facts and read differently: still loading, no
      route planned yet, and a route with nothing obligatorisk on it. Only the last is
      something to go and fix.
    -->
    <p v-if="loading" class="text-sm text-gray-500 py-8 text-center">Henter …</p>
    <p v-else-if="!hasAxis" class="text-sm text-gray-500 py-8 text-center">Ingen åbningstider på posterne endnu, så der er ikke noget tidsrum at vise.</p>
    <p v-else-if="!hasCriteria" class="text-sm text-gray-500 py-8 text-center">Ingen postlinjer er markeret obligatoriske, så alle startede patruljer ville tælle med. Markér de obligatoriske postlinjer for at få grafen.</p>

    <svg v-else :viewBox="`0 0 ${W} ${H}`" class="w-full h-auto" role="img" aria-label="Antal bingo-patruljer over tid" @mousemove="onMove" @mouseleave="hover = null">
      <g>
        <line v-for="t in yTicks" :key="`y${t.y}`" :x1="plot.left" :x2="plot.right" :y1="t.y" :y2="t.y" stroke="#e5e7eb" stroke-width="1" />
        <text v-for="t in yTicks" :key="`yl${t.y}`" :x="plot.left - 6" :y="t.y + 4" text-anchor="end" font-size="11" fill="#6b7280">
          {{ t.count }}
        </text>
      </g>

      <g>
        <line v-for="t in xTicks" :key="`x${t.uts}`" :x1="t.x" :x2="t.x" :y1="plot.top" :y2="plot.bottom" stroke="#f3f4f6" stroke-width="1" />
        <text v-for="t in xTicks" :key="`xl${t.uts}`" :x="t.x" :y="plot.bottom + 14" text-anchor="middle" font-size="11" fill="#6b7280">
          {{ t.label }}
        </text>
        <text v-for="t in dayTicks" :key="`xd${t.uts}`" :x="t.x" :y="plot.bottom + 25" text-anchor="middle" font-size="9" fill="#9ca3af">
          {{ t.day }}
        </text>
      </g>

      <path :d="area" fill="#14b8a6" fill-opacity="0.12" stroke="none" />
      <path :d="line" fill="none" stroke="#0d9488" stroke-width="2" stroke-linejoin="miter" />

      <!-- Where the known part of the night ends. -->
      <g v-if="nowX !== null">
        <line :x1="nowX" :x2="nowX" :y1="plot.top" :y2="plot.bottom" stroke="#ef4444" stroke-width="1" stroke-dasharray="3 3" />
        <text :x="nowX + 4" :y="plot.top + 10" font-size="10" fill="#ef4444">nu</text>
      </g>

      <g v-if="hover">
        <line :x1="hover.x" :x2="hover.x" :y1="plot.top" :y2="plot.bottom" stroke="#9ca3af" stroke-width="1" />
        <circle :cx="hover.x" :cy="scale.y(hover.count)" r="3.5" fill="#0d9488" />
        <!-- Flipped to the left near the right edge so the readout never runs off the plot. -->
        <text :x="hover.x > plot.right - 160 ? hover.x - 8 : hover.x + 8" :text-anchor="hover.x > plot.right - 160 ? 'end' : 'start'" :y="plot.top + 12" font-size="12" fill="#111827">{{ hover.count }} bingo · {{ dddhhmm(new Date(hover.uts * 1000)) }}</text>
      </g>
    </svg>
  </div>
</template>
