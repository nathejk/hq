<script setup lang="ts">
// Kørsel — the dispatch desk (PRD 009).
//
// Two panes: what is not planned, and what the cars are doing. The screen exists to answer the
// one question the desk cannot answer today — **when?** — so every task shows either the planned
// time of the stop it sits on, or how long it has already waited. Never a fabricated ETA: an
// invented number gets read down a phone to a patrol standing in the dark, who then stop making
// their own plans.
//
// # Live, and deliberately paused sometimes
//
// The board is an editor. A half-built tour or a stop reorder mid-flight must not be replaced by
// an incoming payload, so payloads are deferred while an arrangement is unsaved and the screen
// says so (PRD 004; `KlanListView` and `KortView` are the precedent). Everything else updates
// itself, because a screen a volunteer leaves open all night must not quietly go stale.
//
// # What is not here yet
//
// The capacity strip and the queued wait-time estimate (115/116), deadline warnings (117) and
// `Bestil kørsel` from a case (119).

import { computed, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useDeferredApply } from '@/composables/useDeferredApply'
import { useNow } from '@/composables/shelter'
import DispatchTourCard from '@/components/DispatchTourCard.vue'
import DispatchTaskDialog, { type PlaceOption } from '@/components/DispatchTaskDialog.vue'
import {
  type Board,
  type Duty,
  type Task,
  type Tour,
  type TourStop,
  type Unit,
  formatUts,
  formatUtsTime,
  kindIcon,
  kindLabel,
  placeLine,
  priorityLabel,
  priorityTagSeverity,
  stateLabel,
  untilUts,
  waitedFor,
} from '@/composables/dispatch'

const toast = useToast()
const now = useNow()

// The whole board is one live resource with one key.
//
// One key rather than one per pane: a task dragged into a tour changes both, and two keys could
// leave the panes disagreeing about where it is — showing the same task queued *and* on a tour.
//
// dependsOn is entity **types**, and they are the types on the event *subjects*, not the
// projections' names: `dispatch` and `tour` come from
// NATHEJK.{year}.dispatch.{id}.* and NATHEJK.{year}.tour.{id}.*. A new task's id was never seen
// by this client, so only a type-level dependency can make it appear at all.
//
// `dispatchduty` is deliberately absent until task 115 creates it: a dependency nothing can emit
// is exactly what the dev-console warning exists to catch, and adding it early would train
// whoever sees the warning to ignore it.
const { data, pending, error, refresh } = useLiveResource(
  'dispatch',
  async () => {
    const response = await http.get('/dispatch')
    return response.data as Board
  },
  { dependsOn: ['dispatch', 'tour', 'section', 'crewmember', 'crew', 'vehicle', 'spejder', 'sos'] },
)

// --- what the view renders ---

const tasks = ref<Task[]>([])
const tours = ref<Tour[]>([])
const units = ref<Unit[]>([])
const places = ref<PlaceOption[]>([])
const duty = ref<Duty[]>([])

// True while the board must not be redrawn underneath the operator: a write in flight, a local
// arrangement not yet saved, or a dialog holding a half-typed form. A drag is short, but the round
// trip after it is not, and a payload landing mid-save would show the pre-save order for a moment
// and then jump.
const saving = ref(false)
const dragging = ref(false)
const taskDialogOpen = ref(false)
const editingTask = ref<Task | undefined>(undefined)
// Declared here rather than beside the handlers that use them, because `paused` reads them and
// `useDeferredApply` evaluates that immediately — a later `const` would be a temporal-dead-zone
// error on mount, which is a blank screen rather than a warning.
const cancelling = ref<{ kind: 'task' | 'tour'; id: string; label: string } | null>(null)
const boarding = ref<Task | null>(null)
const dutyDialog = ref(false)
const paused = computed(
  () =>
    saving.value ||
    dragging.value ||
    taskDialogOpen.value ||
    dutyDialog.value ||
    !!cancelling.value ||
    !!boarding.value,
)

const { updatesWaiting } = useDeferredApply(data, paused, (board: Board) => {
  tasks.value = board.tasks ?? []
  tours.value = board.tours ?? []
  units.value = board.units ?? []
  places.value = (board as Board & { places?: PlaceOption[] }).places ?? []
  duty.value = board.duty ?? []
})

// --- duty windows (task 115) ---
//
// A roster agreed in advance with the logistics crew, recorded per unit. The editor is a dialog
// rather than a panel on the board: it is set up once an evening and then read all night, and a
// permanent form would take space from the two things that are read constantly. (`dutyDialog`
// itself is declared with the other pause flags above, because `paused` reads it.)
const dutyDraft = ref<{ unit: string; from: Date | null; to: Date | null }>({
  unit: '',
  from: null,
  to: null,
})

const dutyByUnit = computed<Record<string, Duty[]>>(() => {
  const map: Record<string, Duty[]> = {}
  for (const window of [...duty.value].sort((a, b) => a.startUts - b.startUts)) {
    ;(map[window.sectionSlug] ??= []).push(window)
  }
  return map
})

async function saveDuty() {
  const draft = dutyDraft.value
  if (!draft.unit || !draft.from || !draft.to) return
  saving.value = true
  try {
    await http.put('/dispatchduty', {
      sectionSlug: draft.unit,
      startUts: Math.floor(draft.from.getTime() / 1000),
      endUts: Math.floor(draft.to.getTime() / 1000),
    })
    dutyDraft.value = { unit: draft.unit, from: null, to: null }
    await refresh()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke gemme vagten',
      detail: errorDetail(err),
      life: 6000,
    })
  } finally {
    saving.value = false
  }
}

async function removeDuty(id: string) {
  saving.value = true
  try {
    await http.delete(`/dispatchduty/${id}`)
    await refresh()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke fjerne vagten',
      detail: errorDetail(err),
      life: 6000,
    })
  } finally {
    saving.value = false
  }
}

function newTask() {
  editingTask.value = undefined
  taskDialogOpen.value = true
}

function editTask(task: Task) {
  editingTask.value = task
  taskDialogOpen.value = true
}

async function onTaskSaved() {
  await refresh()
}

const tasksById = computed<Record<string, Task>>(() => {
  const map: Record<string, Task> = {}
  for (const task of tasks.value) map[task.id] = task
  return map
})

const unitsBySlug = computed<Record<string, Unit>>(() => {
  const map: Record<string, Unit> = {}
  for (const unit of units.value) map[unit.sectionSlug] = unit
  return map
})

/**
 * The queue: what nobody has planned yet, oldest first.
 *
 * Oldest first because this is a queue and the question asked of it is "who has waited longest",
 * not "what came in last". The API already orders it; the sort is here so a locally-mutated list
 * cannot drift out of order between payloads.
 */
const queued = computed(() =>
  tasks.value.filter((t) => t.state === 'queued').sort((a, b) => a.createdUts - b.createdUts),
)

/** Tours worth showing: the ones still happening, plus tonight's finished runs at the bottom. */
const openTours = computed(() =>
  tours.value.filter((t) => t.state === 'planned' || t.state === 'underway'),
)
const closedTours = computed(() =>
  tours.value.filter((t) => t.state === 'completed' || t.state === 'cancelled'),
)

/** Tasks underway, so the desk can see what is in a car right now. */
const underway = computed(() => tasks.value.filter((t) => t.state === 'underway'))

// The four numbers the board answers at a glance (PRD 009 §6). The oldest wait is the one that
// matters most, and it is the reason it is a headline rather than a column somebody scrolls to.
const oldestWait = computed(() => {
  const first = queued.value[0]
  return first ? waitedFor(first, now.value) : ''
})

// --- writes ---

/**
 * Send a tour's stops.
 *
 * `plannedUts` is sent **only** for a stop somebody actually overrode. Sending the derived times
 * back would mark every stop as manually set, and then nothing would ever re-derive — a
 * departure change would silently stop moving the plan. Visited stops keep their identity by id.
 */
async function saveStops(tour: Tour, stops: TourStop[]) {
  saving.value = true
  try {
    const { data: response } = await http.put(`/dispatch/tour/${tour.id}/stops`, {
      stops: stops.map((s) => ({
        stopId: s.stopId || undefined,
        place: s.place,
        plannedUts: s.override ? s.plannedUts : undefined,
        tasks: s.tasks,
      })),
    })
    for (const warning of response?.warnings ?? []) {
      // Warnings are things the desk should know and must not be blocked by: seats fold down,
      // and a platform that refuses the real world gets worked around.
      toast.add({ severity: 'warn', summary: warning.message, life: 6000 })
    }
    await refresh()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke gemme turens stop',
      detail: errorDetail(err),
      life: 6000,
    })
    await refresh()
  } finally {
    saving.value = false
  }
}

/**
 * Put a queued task on a tour.
 *
 * A pickup or transport becomes **two** stops — where it is loaded and where it is unloaded —
 * because that is what it is: a task that moves something occupies two places, and a single stop
 * would make "when will they be collected" and "when will they arrive" the same number.
 */
function stopsForTask(task: Task): TourStop[] {
  const blank = (label: string, place: Task['pickup'], role: 'load' | 'unload'): TourStop => ({
    stopId: '',
    sortOrder: 0,
    place: place?.label ? place : { kind: 'text', label },
    plannedUts: null,
    override: false,
    visitedUts: null,
    tasks: [{ taskId: task.id, role }],
  })
  return [blank('Hentes', task.pickup, 'load'), blank('Afleveres', task.dropoff, 'unload')]
}

function onDropTask(tour: Tour, payload: { taskId: string; afterStopId?: string }) {
  dragging.value = false
  const task = tasksById.value[payload.taskId]
  if (!task) return
  const stops = [...tour.stops]
  const added = stopsForTask(task)
  if (payload.afterStopId) {
    const at = stops.findIndex((s) => s.stopId === payload.afterStopId)
    stops.splice(at + 1, 0, ...added)
  } else {
    stops.push(...added)
  }
  void saveStops(tour, stops)
}

function onMoveStop(tour: Tour, payload: { stopId: string; direction: -1 | 1 }) {
  const stops = [...tour.stops]
  const from = stops.findIndex((s) => s.stopId === payload.stopId)
  const to = from + payload.direction
  if (from < 0 || to < 0 || to >= stops.length) return
  // A visited stop is fixed, so refuse the swap here as well as server-side: the operator gets an
  // explanation instead of a 422 they have to interpret.
  if (stops[to].visitedUts) {
    toast.add({ severity: 'warn', summary: 'Et besøgt stop kan ikke flyttes', life: 4000 })
    return
  }
  ;[stops[from], stops[to]] = [stops[to], stops[from]]
  void saveStops(tour, stops)
}

function onRemoveStop(tour: Tour, stopId: string) {
  void saveStops(
    tour,
    tour.stops.filter((s) => s.stopId !== stopId),
  )
}

async function post(url: string, body?: unknown, failure = 'Handlingen mislykkedes') {
  saving.value = true
  try {
    await http.post(url, body ?? {})
    await refresh()
  } catch (err: any) {
    toast.add({ severity: 'error', summary: failure, detail: errorDetail(err), life: 6000 })
  } finally {
    saving.value = false
  }
}

const visitStop = (tour: Tour, stopId: string) =>
  post(`/dispatch/tour/${tour.id}/stop/${stopId}/visited`, {}, 'Kunne ikke registrere stoppet')

const startTour = (tour: Tour) =>
  post(`/dispatch/tour/${tour.id}/underway`, {}, 'Kunne ikke starte turen')

const completeTour = (tour: Tour) =>
  post(`/dispatch/tour/${tour.id}/completed`, {}, 'Kunne ikke afslutte turen')

// --- the new-tour form ---

const newTourUnit = ref<string>('')
const newTourDeparture = ref<Date | null>(null)

async function createTour() {
  if (!newTourUnit.value) return
  saving.value = true
  try {
    await http.post('/dispatch/tour', {
      sectionSlug: newTourUnit.value,
      departureUts: newTourDeparture.value
        ? Math.floor(newTourDeparture.value.getTime() / 1000)
        : undefined,
    })
    newTourDeparture.value = null
    await refresh()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke oprette turen',
      detail: errorDetail(err),
      life: 6000,
    })
  } finally {
    saving.value = false
  }
}

// --- cancelling, which always needs a reason ---

const cancelReason = ref('')

function askToCancelTask(task: Task) {
  cancelling.value = { kind: 'task', id: task.id, label: task.description }
  cancelReason.value = ''
}

function askToCancelTour(tour: Tour) {
  cancelling.value = {
    kind: 'tour',
    id: tour.id,
    label: unitsBySlug.value[tour.sectionSlug]?.label ?? tour.sectionSlug,
  }
  cancelReason.value = ''
}

async function confirmCancel() {
  const target = cancelling.value
  if (!target || !cancelReason.value.trim()) return
  const url =
    target.kind === 'task'
      ? `/dispatch/task/${target.id}/cancelled`
      : `/dispatch/tour/${target.id}/cancelled`
  cancelling.value = null
  await post(url, { reason: cancelReason.value.trim() }, 'Kunne ikke aflyse')
}

// --- people aboard ---

const boardingUnit = ref<string>('')

function askWhoCollected(task: Task) {
  boarding.value = task
  // Pre-selected from the tour the task is on when there is exactly one: the dispatcher already
  // said which car this was, and asking again is a click for nothing.
  const tour = tours.value.find((t) => t.stops.some((s) => s.tasks.some((st) => st.taskId === task.id)))
  boardingUnit.value = tour?.sectionSlug ?? units.value[0]?.sectionSlug ?? ''
}

async function confirmBoarding() {
  const task = boarding.value
  if (!task) return
  boarding.value = null
  await post(
    `/dispatch/task/${task.id}/pickedup`,
    { sectionSlug: boardingUnit.value },
    'Kunne ikke registrere at folk er med',
  )
}

const unitOptions = computed(() =>
  units.value.map((u) => ({ value: u.sectionSlug, label: u.label })),
)

function errorDetail(err: any) {
  const payload = err?.response?.data?.error
  if (!payload) return String(err)
  if (typeof payload === 'string') return payload
  return Object.values(payload).join(', ')
}
</script>

<template>
  <div class="p-4 space-y-4">
    <header class="flex flex-wrap items-center gap-3">
      <h1 class="text-xl font-semibold">Kørsel</h1>

      <!-- The two numbers that decide what the desk does next, before any scrolling. -->
      <Tag :value="`${queued.length} ikke planlagt`" :severity="queued.length ? 'warn' : 'secondary'" />
      <Tag v-if="oldestWait" :value="`ældste ${oldestWait}`" severity="secondary" />
      <Tag :value="`${openTours.length} ture ude`" severity="secondary" />

      <div class="flex-1" />
      <Button label="Ny opgave" icon="pi pi-plus" size="small" @click="newTask()" />
      <Button
        label="Vagter"
        icon="pi pi-clock"
        size="small"
        severity="secondary"
        outlined
        @click="dutyDialog = true"
      />
      <Button
        icon="pi pi-refresh"
        label="Opdater"
        size="small"
        severity="secondary"
        text
        :loading="pending"
        @click="refresh()"
      />
    </header>

    <!--
      A page that has taught its operator to trust it is live owes them a word the one time it is
      deliberately not.
    -->
    <Message v-if="updatesWaiting" severity="info" :closable="false">
      Opdateringer er sat på pause, mens du redigerer. De vises, når du er færdig.
    </Message>
    <Message v-if="error" severity="error" :closable="false">
      Kunne ikke hente kørselstavlen.
    </Message>

    <div class="grid gap-4 md:grid-cols-2">
      <!-- Ikke planlagt -->
      <section class="space-y-2">
        <h2 class="font-semibold">Ikke planlagt ({{ queued.length }})</h2>
        <DataTable
          :value="queued"
          :loading="pending"
          dataKey="id"
          size="small"
          class="border rounded"
        >
          <template #empty>
            <span class="text-gray-500 italic">Ingen opgaver venter.</span>
          </template>
          <Column header="Opgave">
            <template #body="{ data: task }">
              <div
                class="cursor-move"
                :draggable="true"
                @dragstart="
                  dragging = true;
                  $event.dataTransfer?.setData('text/dispatch-task', task.id)
                "
                @dragend="dragging = false"
              >
                <div class="flex items-center gap-2">
                  <i :class="kindIcon(task.kind)" v-tooltip.top="kindLabel(task.kind)" />
                  <span class="font-medium truncate">{{ task.description }}</span>
                  <Tag
                    v-if="task.priority"
                    :value="priorityLabel(task.priority)"
                    :severity="priorityTagSeverity(task.priority)"
                  />
                </div>
                <div class="text-xs text-gray-600">
                  {{ placeLine(task.pickup) }} → {{ placeLine(task.dropoff) }}
                  <span v-if="task.spaceNeeds"> · {{ task.spaceNeeds }}</span>
                </div>
              </div>
            </template>
          </Column>
          <Column header="Ventet" style="width: 8rem">
            <template #body="{ data: task }">
              <!-- Always shown, from oprettet: the number that needs no model and is never
                   wrong. It advances because the whole screen shares one clock. -->
              <span class="tabular-nums">{{ waitedFor(task, now) }}</span>
              <div v-if="task.deadlineUts" class="text-xs text-gray-600">
                senest {{ formatUtsTime(task.deadlineUts) }} ({{ untilUts(task.deadlineUts, now) }})
              </div>
              <div v-if="task.notBeforeUts" class="text-xs text-gray-500">
                tidligst {{ formatUtsTime(task.notBeforeUts) }}
              </div>
            </template>
          </Column>
          <Column style="width: 5rem">
            <template #body="{ data: task }">
              <Button
                icon="pi pi-pencil"
                size="small"
                text
                rounded
                severity="secondary"
                :disabled="saving"
                v-tooltip.top="'Rediger opgaven'"
                @click="editTask(task)"
              />
              <Button
                icon="pi pi-ban"
                size="small"
                text
                rounded
                severity="danger"
                :disabled="saving"
                v-tooltip.top="'Aflys opgaven'"
                @click="askToCancelTask(task)"
              />
            </template>
          </Column>
        </DataTable>

        <!-- What is in a car right now: the answer to "where is the car I sent out an hour ago". -->
        <template v-if="underway.length">
          <h2 class="font-semibold pt-2">Undervejs ({{ underway.length }})</h2>
          <ul class="space-y-1">
            <li
              v-for="task in underway"
              :key="task.id"
              class="border rounded px-2 py-1 text-sm flex items-center gap-2 bg-white"
            >
              <i :class="kindIcon(task.kind)" />
              <span class="flex-1 truncate">{{ task.description }}</span>
              <span v-if="task.pickedUpUts" class="text-xs text-green-700">
                hentet {{ formatUtsTime(task.pickedUpUts) }}
              </span>
              <Button
                v-else-if="task.kind === 'pickup'"
                label="Hentet"
                icon="pi pi-user-plus"
                size="small"
                severity="secondary"
                outlined
                :disabled="saving"
                @click="askWhoCollected(task)"
              />
              <Tag :value="stateLabel(task.state)" severity="secondary" />
            </li>
          </ul>
        </template>
      </section>

      <!-- Ture -->
      <section class="space-y-2">
        <h2 class="font-semibold">Ture</h2>

        <div class="border rounded p-2 flex flex-wrap items-end gap-2 bg-gray-50">
          <div>
            <label class="block text-xs text-gray-600">Enhed</label>
            <Select
              v-model="newTourUnit"
              :options="unitOptions"
              optionLabel="label"
              optionValue="value"
              placeholder="Vælg enhed…"
              size="small"
              class="w-44"
            />
          </div>
          <div>
            <label class="block text-xs text-gray-600">Afgang</label>
            <DatePicker v-model="newTourDeparture" showTime hourFormat="24" size="small" class="w-44" />
          </div>
          <Button
            label="Ny tur"
            icon="pi pi-plus"
            size="small"
            :disabled="!newTourUnit || saving"
            @click="createTour()"
          />
          <!-- No unit configured is a normal state, not a bug: it means nobody has said which
               subsections hold a car, and the Organisation page is where that is done. -->
          <small v-if="unitOptions.length === 0" class="text-gray-600">
            Ingen kørsels-enheder. Marker en underafdeling som kørsels-enhed på Organisation.
          </small>
        </div>

        <p v-if="openTours.length === 0" class="text-sm text-gray-500 italic">Ingen ture planlagt.</p>

        <DispatchTourCard
          v-for="tour in openTours"
          :key="tour.id"
          :tour="tour"
          :unit="unitsBySlug[tour.sectionSlug]"
          :tasksById="tasksById"
          :nowMs="now"
          :busy="saving"
          @drop-task="onDropTask(tour, $event)"
          @move-stop="onMoveStop(tour, $event)"
          @remove-stop="onRemoveStop(tour, $event)"
          @visit-stop="visitStop(tour, $event)"
          @start="startTour(tour)"
          @complete="completeTour(tour)"
          @cancel="askToCancelTour(tour)"
          @edit="newTourUnit = tour.sectionSlug"
        />

        <template v-if="closedTours.length">
          <h3 class="font-medium text-gray-600 pt-2">Afsluttede ({{ closedTours.length }})</h3>
          <DispatchTourCard
            v-for="tour in closedTours"
            :key="tour.id"
            :tour="tour"
            :unit="unitsBySlug[tour.sectionSlug]"
            :tasksById="tasksById"
            :nowMs="now"
          />
        </template>
      </section>
    </div>

    <!--
      The roster (PRD 009 §6). Per unit, and per unit only: the unit is what is available or
      asleep, and a window per person would have to be intersected with the co-driver's to answer
      the one question the board asks of it.
    -->
    <Dialog
      v-model:visible="dutyDialog"
      modal
      header="Vagter"
      :style="{ width: '34rem' }"
    >
      <div class="flex flex-wrap items-end gap-2 pb-3">
        <div>
          <label class="block text-xs text-gray-600">Enhed</label>
          <Select
            v-model="dutyDraft.unit"
            :options="unitOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Vælg enhed…"
            size="small"
            class="w-40"
          />
        </div>
        <div>
          <label class="block text-xs text-gray-600">Fra</label>
          <DatePicker v-model="dutyDraft.from" showTime hourFormat="24" size="small" class="w-40" />
        </div>
        <div>
          <label class="block text-xs text-gray-600">Til</label>
          <DatePicker v-model="dutyDraft.to" showTime hourFormat="24" size="small" class="w-40" />
        </div>
        <Button
          label="Tilføj"
          icon="pi pi-plus"
          size="small"
          :disabled="!dutyDraft.unit || !dutyDraft.from || !dutyDraft.to || saving"
          @click="saveDuty()"
        />
      </div>

      <div v-for="unit in units" :key="unit.sectionSlug" class="border-t py-2">
        <div class="font-medium">{{ unit.label }}</div>
        <ul v-if="dutyByUnit[unit.sectionSlug]?.length" class="text-sm">
          <li
            v-for="window in dutyByUnit[unit.sectionSlug]"
            :key="window.id"
            class="flex items-center gap-2"
          >
            <!-- Weekday-bearing, because the race runs through a night and "21.40 til 02.00"
                 alone does not say which evening. -->
            <span class="flex-1 tabular-nums">
              {{ formatUts(window.startUts) }} – {{ formatUts(window.endUts) }}
            </span>
            <Button
              icon="pi pi-trash"
              size="small"
              text
              rounded
              severity="danger"
              :disabled="saving"
              @click="removeDuty(window.id)"
            />
          </li>
        </ul>
        <small v-else class="text-gray-500">Ingen vagter aftalt.</small>
      </div>
      <small v-if="units.length === 0" class="text-gray-600">
        Ingen kørsels-enheder endnu. Marker en underafdeling som kørsels-enhed på Organisation.
      </small>
    </Dialog>

    <DispatchTaskDialog
      v-model:visible="taskDialogOpen"
      :task="editingTask"
      :places="places"
      @saved="onTaskSaved()"
    />

    <!-- Cancelling always asks why: a cancelled job with no explanation is the one thing a shift
         handover cannot recover from. -->
    <Dialog
      :visible="!!cancelling"
      modal
      header="Aflys"
      :style="{ width: '28rem' }"
      @update:visible="cancelling = null"
    >
      <p class="pb-2">{{ cancelling?.label }}</p>
      <InputText v-model="cancelReason" class="w-full" placeholder="Årsag" autofocus />
      <template #footer>
        <Button label="Annuller" severity="secondary" text @click="cancelling = null" />
        <Button
          label="Aflys"
          severity="danger"
          :disabled="!cancelReason.trim()"
          @click="confirmCancel()"
        />
      </template>
    </Dialog>

    <!-- Which unit took them. A section slug, not a car: the unit is who has the scout, and it
         survives a vehicle being swapped mid-night. -->
    <Dialog
      :visible="!!boarding"
      modal
      header="Folk med i bilen"
      :style="{ width: '28rem' }"
      @update:visible="boarding = null"
    >
      <p class="pb-2">{{ boarding?.description }}</p>
      <Select
        v-model="boardingUnit"
        :options="unitOptions"
        optionLabel="label"
        optionValue="value"
        placeholder="Hvilken enhed?"
        class="w-full"
      />
      <template #footer>
        <Button label="Annuller" severity="secondary" text @click="boarding = null" />
        <Button label="Hentet" :disabled="!boardingUnit" @click="confirmBoarding()" />
      </template>
    </Dialog>
  </div>
</template>
