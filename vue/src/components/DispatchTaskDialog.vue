<script setup lang="ts">
// Create or edit a kørsel task (PRD 009 §7, task 114).
//
// The form's shape follows one finding from PRD 009 §8: **the board is only as good as the desk's
// discipline**, and the mitigation is that writing a job down has to be faster than not writing
// it down. So only the kind and a description are required, the kind defaults the two places to
// what that kind almost always means, and every time field is optional.
//
// It emits rather than reloads: the board owns the payload and the cache key, so the dialog's job
// ends when the write succeeds.

import { computed, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import {
  type Place,
  type PlaceKind,
  type Task,
  type TaskKind,
  kindLabel,
  priorityOptions,
} from '@/composables/dispatch'

export interface PlaceOption {
  kind: PlaceKind
  refId: string
  label: string
}

const props = defineProps<{
  visible: boolean
  /** The task being edited, or undefined to create one. */
  task?: Task
  places: PlaceOption[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved', taskId: string): void
}>()

const toast = useToast()

const kindOptions: { value: TaskKind; label: string }[] = (
  ['pickup', 'transport', 'collection', 'delivery'] as TaskKind[]
).map((value) => ({ value, label: kindLabel(value) }))

const kind = ref<TaskKind>('pickup')
const priority = ref<string>('')
const description = ref('')
const spaceNeeds = ref('')
const pickup = ref<Place>({ kind: 'text', label: '' })
const dropoff = ref<Place>({ kind: 'hq', label: 'HQ' })
const notBefore = ref<Date | null>(null)
const deadline = ref<Date | null>(null)
const saving = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const editing = computed(() => !!props.task)
const hq = computed<Place>(
  () => props.places.find((p) => p.kind === 'hq') ?? { kind: 'hq', refId: '', label: 'HQ' },
)

/**
 * What each kind means for the two places.
 *
 * The kinds exist *because* they read differently on a board and default differently (PRD 009
 * §6); this is that second half. A delivery leaves HQ, a collection returns to it, a pickup
 * brings people in — so the dispatcher types one end, not two.
 */
function applyKindDefaults(next: TaskKind) {
  const blank: Place = { kind: 'text', label: '' }
  switch (next) {
    case 'pickup':
      pickup.value = { ...blank }
      dropoff.value = { ...hq.value }
      break
    case 'delivery':
      pickup.value = { ...hq.value }
      dropoff.value = { ...blank }
      break
    case 'collection':
      pickup.value = { ...blank }
      dropoff.value = { ...hq.value }
      break
    case 'transport':
      pickup.value = { ...blank }
      dropoff.value = { ...blank }
      break
  }
}

// Only when the operator changes the kind, and never while editing an existing task: rewriting
// the places of a task somebody already filled in is how a form loses work.
watch(kind, (next, previous) => {
  if (previous === undefined || next === previous) return
  if (editing.value) return
  applyKindDefaults(next)
})

const utsToDate = (uts?: number | null) => (uts ? new Date(uts * 1000) : null)
const dateToUts = (date: Date | null) => (date ? Math.floor(date.getTime() / 1000) : null)

// Reset on open, from the task if there is one. Watching `visible` rather than doing this on
// mount because PrimeVue keeps the component alive between openings, so a stale draft would
// otherwise be waiting the next time the dialog appears.
watch(
  () => props.visible,
  (open) => {
    if (!open) return
    fieldErrors.value = {}
    const task = props.task
    if (task) {
      kind.value = task.kind
      priority.value = task.priority ?? ''
      description.value = task.description
      spaceNeeds.value = task.spaceNeeds ?? ''
      pickup.value = { ...task.pickup }
      dropoff.value = { ...task.dropoff }
      notBefore.value = utsToDate(task.notBeforeUts)
      deadline.value = utsToDate(task.deadlineUts)
      return
    }
    kind.value = 'pickup'
    priority.value = ''
    description.value = ''
    spaceNeeds.value = ''
    notBefore.value = null
    deadline.value = null
    applyKindDefaults('pickup')
  },
)

async function save() {
  if (!description.value.trim()) {
    fieldErrors.value = { description: 'opgaven skal have en beskrivelse' }
    return
  }
  saving.value = true
  fieldErrors.value = {}
  try {
    if (props.task) {
      // A PATCH: an absent field is left alone and an explicit null clears it, which is why the
      // times are sent as `null` rather than omitted when the operator emptied them.
      await http.patch(`/dispatch/task/${props.task.id}`, {
        kind: kind.value,
        priority: priority.value,
        description: description.value.trim(),
        spaceNeeds: spaceNeeds.value.trim(),
        pickup: pickup.value,
        dropoff: dropoff.value,
        notBeforeUts: dateToUts(notBefore.value),
        deadlineUts: dateToUts(deadline.value),
      })
      emit('saved', props.task.id)
    } else {
      const { data } = await http.post('/dispatch/task', {
        kind: kind.value,
        priority: priority.value,
        description: description.value.trim(),
        spaceNeeds: spaceNeeds.value.trim(),
        pickup: pickup.value,
        dropoff: dropoff.value,
        notBeforeUts: dateToUts(notBefore.value) ?? undefined,
        deadlineUts: dateToUts(deadline.value) ?? undefined,
      })
      emit('saved', data?.taskId ?? '')
    }
    emit('update:visible', false)
  } catch (err: any) {
    // Field-level refusals go beside the field they are about; anything else is a toast. The API
    // answers 422 with a map of field → Danish message, which is exactly this shape.
    const payload = err?.response?.data?.error
    if (payload && typeof payload === 'object') {
      fieldErrors.value = payload
    } else {
      toast.add({
        severity: 'error',
        summary: 'Kunne ikke gemme opgaven',
        detail: String(err),
        life: 6000,
      })
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="editing ? 'Rediger opgave' : 'Ny opgave'"
    :style="{ width: '34rem' }"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="space-y-3">
      <div class="flex gap-3">
        <div class="flex-1">
          <label class="block text-sm text-gray-700">Type</label>
          <Select
            v-model="kind"
            :options="kindOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
          <small v-if="fieldErrors.kind" class="text-red-600">{{ fieldErrors.kind }}</small>
        </div>
        <div class="flex-1">
          <!-- The nødtelefon's own vocabulary, shared rather than copied (task 112): a pickup
               created from a red case should be able to arrive red. -->
          <label class="block text-sm text-gray-700">Prioritet</label>
          <Select
            v-model="priority"
            :options="priorityOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Ingen"
            showClear
            class="w-full"
          />
        </div>
      </div>

      <div>
        <label class="block text-sm text-gray-700">Hvad skal der ske?</label>
        <Textarea v-model="description" rows="2" class="w-full" autoResize autofocus />
        <small v-if="fieldErrors.description" class="text-red-600">
          {{ fieldErrors.description }}
        </small>
      </div>

      <div class="flex gap-3">
        <div class="flex-1">
          <label class="block text-sm text-gray-700">Hentes</label>
          <DispatchPlacePicker v-model="pickup" :places="places" />
        </div>
        <div class="flex-1">
          <label class="block text-sm text-gray-700">Afleveres</label>
          <DispatchPlacePicker v-model="dropoff" :places="places" />
        </div>
      </div>

      <div>
        <label class="block text-sm text-gray-700">Pladsbehov</label>
        <!-- Words, not an inventory: PRD 009 §4 refuses to track how many maps exist. -->
        <InputText v-model="spaceNeeds" class="w-full" placeholder="fx “fylder næsten hele bagagerummet”" />
      </div>

      <div class="flex gap-3">
        <div class="flex-1">
          <label class="block text-sm text-gray-700">Tidligst</label>
          <DatePicker v-model="notBefore" showTime hourFormat="24" class="w-full" showButtonBar />
        </div>
        <div class="flex-1">
          <label class="block text-sm text-gray-700">Skal leveres</label>
          <DatePicker v-model="deadline" showTime hourFormat="24" class="w-full" showButtonBar />
          <small v-if="fieldErrors.deadlineUts" class="text-red-600">
            {{ fieldErrors.deadlineUts }}
          </small>
        </div>
      </div>
    </div>

    <template #footer>
      <Button label="Annuller" severity="secondary" text @click="emit('update:visible', false)" />
      <Button
        :label="editing ? 'Gem' : 'Opret'"
        icon="pi pi-check"
        :loading="saving"
        @click="save()"
      />
    </template>
  </Dialog>
</template>
