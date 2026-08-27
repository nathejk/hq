<script setup lang="ts">
// One tour on the kørsel board (PRD 009 §7): a card per run, its stops in order.
//
// The card owns no data. It renders the tour it is given and emits what the operator did, so the
// view keeps one place where writes happen — which is what lets the whole board defer live
// updates while an arrangement is unsaved (PRD 004). A card that saved its own reorders would
// each need that dance separately, and one of them would eventually not do it.

import { computed } from 'vue'
import {
  type Tour,
  type Task,
  type Unit,
  type TourStop,
  formatUtsTime,
  kindIcon,
  placeLine,
  roleLabel,
  tourStateLabel,
} from '@/composables/dispatch'

const props = defineProps<{
  tour: Tour
  unit?: Unit
  /** Every task on the board, so a stop can show what it is actually carrying. */
  tasksById: Record<string, Task>
  nowMs: number
  busy?: boolean
}>()

const emit = defineEmits<{
  (e: 'drop-task', payload: { taskId: string; afterStopId?: string }): void
  (e: 'move-stop', payload: { stopId: string; direction: -1 | 1 }): void
  (e: 'remove-stop', stopId: string): void
  (e: 'visit-stop', stopId: string): void
  (e: 'start'): void
  (e: 'complete'): void
  (e: 'cancel'): void
  (e: 'edit'): void
}>()

const stateSeverity = computed(() => {
  switch (props.tour.state) {
    case 'underway':
      return 'warn'
    case 'completed':
      return 'success'
    case 'cancelled':
      return 'danger'
    default:
      return 'secondary'
  }
})

const open = computed(() => props.tour.state === 'planned' || props.tour.state === 'underway')

// A tour whose stops have all been visited offers to complete, rather than completing itself:
// the driver may still be talking, and the desk is the one who knows the run is over.
const canComplete = computed(
  () => open.value && props.tour.stops.length > 0 && props.tour.stops.every((s) => s.visitedUts),
)

/**
 * A stop whose planned time has passed without it being visited is overdue.
 *
 * That is the point of writing plans down: it makes broken ones visible (PRD 009 §5). Nothing is
 * overdue on a tour that has not set off — an un-departed plan is not yet a promise.
 */
const overdue = (stop: TourStop) =>
  props.tour.state === 'underway' &&
  !stop.visitedUts &&
  !!stop.plannedUts &&
  stop.plannedUts * 1000 < props.nowMs

const vehicle = computed(() => props.unit?.vehicles?.[0])

function onDrop(event: DragEvent, afterStopId?: string) {
  event.preventDefault()
  const taskId = event.dataTransfer?.getData('text/dispatch-task')
  if (taskId) emit('drop-task', { taskId, afterStopId })
}
</script>

<template>
  <div
    class="border rounded p-3 bg-white space-y-2"
    :class="{ 'opacity-60': !open }"
    @dragover.prevent
    @drop="onDrop($event)"
  >
    <header class="flex items-center gap-2">
      <i class="pi pi-truck text-primary-500" />
      <span class="font-semibold flex-1 truncate">
        {{ unit?.label ?? tour.sectionSlug }}
        <span v-if="vehicle" class="text-gray-500 font-normal">· {{ vehicle.licensePlate }}</span>
      </span>
      <Tag :value="tourStateLabel(tour.state)" :severity="stateSeverity" />
    </header>

    <div class="text-sm text-gray-600 flex items-center gap-3">
      <span v-if="tour.departureUts">Afgang {{ formatUtsTime(tour.departureUts) }}</span>
      <span v-else class="italic">Ingen afgangstid</span>
      <span v-if="tour.notes" class="truncate">{{ tour.notes }}</span>
    </div>

    <!--
      An unplanned tour with no stops is not an error state — a dispatcher builds the tour first
      and fills it from the queue — so it says what to do rather than looking broken.
    -->
    <p v-if="tour.stops.length === 0" class="text-sm text-gray-500 italic">
      Ingen stop. Træk en opgave herover.
    </p>

    <ol v-else class="space-y-1">
      <li
        v-for="(stop, index) in tour.stops"
        :key="stop.stopId"
        class="border rounded px-2 py-1 text-sm"
        :class="{
          'bg-gray-50': !!stop.visitedUts,
          'border-red-300': overdue(stop),
        }"
        @dragover.prevent
        @drop.stop="onDrop($event, stop.stopId)"
      >
        <div class="flex items-center gap-2">
          <span class="text-gray-400 w-4 text-right">{{ index + 1 }}</span>
          <span class="flex-1 truncate">{{ placeLine(stop.place) }}</span>

          <!-- An overridden time is marked, because a time somebody typed is a different fact
               from one the platform derived, and the following stops follow it. -->
          <span
            v-if="stop.plannedUts"
            class="tabular-nums"
            :class="overdue(stop) ? 'text-red-600 font-medium' : 'text-gray-600'"
            v-tooltip.top="stop.override ? 'Tid sat manuelt' : 'Beregnet tid'"
          >
            {{ formatUtsTime(stop.plannedUts) }}<span v-if="stop.override">*</span>
          </span>

          <template v-if="open && !stop.visitedUts">
            <Button
              icon="pi pi-arrow-up"
              size="small"
              text
              rounded
              severity="secondary"
              :disabled="busy || index === 0"
              v-tooltip.top="'Flyt op'"
              @click="emit('move-stop', { stopId: stop.stopId, direction: -1 })"
            />
            <Button
              icon="pi pi-arrow-down"
              size="small"
              text
              rounded
              severity="secondary"
              :disabled="busy || index === tour.stops.length - 1"
              v-tooltip.top="'Flyt ned'"
              @click="emit('move-stop', { stopId: stop.stopId, direction: 1 })"
            />
            <Button
              icon="pi pi-check"
              size="small"
              text
              rounded
              severity="success"
              :disabled="busy"
              v-tooltip.top="'Stoppet er nået'"
              @click="emit('visit-stop', stop.stopId)"
            />
            <Button
              icon="pi pi-times"
              size="small"
              text
              rounded
              severity="danger"
              :disabled="busy"
              v-tooltip.top="'Fjern stop'"
              @click="emit('remove-stop', stop.stopId)"
            />
          </template>
          <span v-else-if="stop.visitedUts" class="text-green-700 tabular-nums">
            <i class="pi pi-check" /> {{ formatUtsTime(stop.visitedUts) }}
          </span>
        </div>

        <ul v-if="stop.tasks.length" class="pl-6 text-xs text-gray-600">
          <li v-for="st in stop.tasks" :key="st.taskId + st.role" class="truncate">
            <i :class="kindIcon(tasksById[st.taskId]?.kind ?? '')" />
            {{ tasksById[st.taskId]?.description ?? st.taskId }}
            <span class="text-gray-400">({{ roleLabel(st.role) }})</span>
          </li>
        </ul>
      </li>
    </ol>

    <footer v-if="open" class="flex flex-wrap gap-2 pt-1">
      <Button
        v-if="tour.state === 'planned'"
        label="Kørt"
        icon="pi pi-play"
        size="small"
        severity="secondary"
        outlined
        :disabled="busy"
        @click="emit('start')"
      />
      <Button
        v-if="canComplete"
        label="Afslut tur"
        icon="pi pi-flag"
        size="small"
        severity="success"
        outlined
        :disabled="busy"
        @click="emit('complete')"
      />
      <Button
        label="Rediger"
        icon="pi pi-pencil"
        size="small"
        severity="secondary"
        text
        :disabled="busy"
        @click="emit('edit')"
      />
      <Button
        label="Aflys"
        icon="pi pi-ban"
        size="small"
        severity="danger"
        text
        :disabled="busy"
        @click="emit('cancel')"
      />
    </footer>
    <p v-else-if="tour.cancelReason" class="text-sm text-red-700">Aflyst: {{ tour.cancelReason }}</p>
  </div>
</template>
