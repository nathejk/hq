<script setup lang="ts">
// Define the sheets we print and hand out (PRD 010, task 127).
//
// # Why a side dialog and not a page
//
// The map behind it is the feedback surface: selecting a sheet highlights exactly the checkpoints
// drawn on it, which is how an operator sees a mistake *before* the sheets go to the printer. A
// full-screen editor would hide the one thing being described. So this is a narrow modal on the
// right, and the view it sits over does the highlighting.
//
// # It writes, the view reloads
//
// Every write goes straight to the API and the owner refreshes the shared cache entry, following
// `DispatchTaskDialog`. There is no local copy of the payload to keep in step, which matters here
// because the same data is on screen twice — once as markers, once as this list.

import { computed, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import draggable from 'vuedraggable'
import { http } from '@/plugins/axios'
import { formatOptions, teamTypeLabel, type Kort, type Kortsaet, type KortPayload } from '@/composables/kort'

const props = defineProps<{
  visible: boolean
  payload?: KortPayload
  /** The sheet whose checkpoints are highlighted on the map. */
  selectedId?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:selectedId', value: string | undefined): void
  /** A write succeeded; the owner refreshes the cache entry. */
  (e: 'saved'): void
  /**
   * Whether the operator has unsaved work in here.
   *
   * The view pauses applying live payloads while this is true (task 131). Emitted rather than
   * inspected, because only this component knows what "half-edited" means.
   */
  (e: 'update:dirty', value: boolean): void
}>()

const toast = useToast()

const sets = computed(() => props.payload?.kortsaet ?? [])
const orphans = computed(() => props.payload?.orphanKort ?? [])

const selected = computed(() => {
  const id = props.selectedId
  if (!id) return undefined
  for (const set of sets.value) {
    const found = set.kort.find((sheet) => sheet.id === id)
    if (found) return found
  }
  return orphans.value.find((sheet) => sheet.id === id)
})

// --- the edit buffer ---
//
// The form edits a copy, not the cached row. Two reasons, and the second is the one that bites:
// the cached value is shared with the markers on the map, so typing into it would rename a marker
// letter by letter; and a live payload arriving mid-edit must not rewrite the field under the
// cursor.

const draft = ref<{ name: string; format: string; note: string; kortsaetId: string }>({
  name: '',
  format: '',
  note: '',
  kortsaetId: '',
})
const saving = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const loadDraft = (sheet?: Kort) => {
  draft.value = {
    name: sheet?.name ?? '',
    format: sheet?.format ?? '',
    note: sheet?.note ?? '',
    kortsaetId: sheet?.kortsaetId ?? '',
  }
  fieldErrors.value = {}
}

watch(selected, (sheet) => loadDraft(sheet), { immediate: true })

/** Has the operator changed something that is not saved? */
const dirty = computed(() => {
  const sheet = selected.value
  if (!sheet) return false
  return (
    draft.value.name !== sheet.name ||
    draft.value.format !== (sheet.format ?? '') ||
    draft.value.note !== sheet.note ||
    draft.value.kortsaetId !== sheet.kortsaetId
  )
})

// Reordering counts as dirty too: the list on screen is in an order the server does not have yet.
const reordering = ref(false)
watch(
  [dirty, reordering],
  ([isDirty, isReordering]) => emit('update:dirty', isDirty || isReordering),
  { immediate: true },
)

const select = (sheet: Kort) => {
  if (dirty.value) {
    // Refuse rather than discard: switching sheets with unsaved text would throw away work with
    // no warning, and this list is exactly where a mis-click is easy.
    toast.add({
      severity: 'warn',
      summary: 'Gem eller annullér først',
      detail: 'Der er ugemte ændringer på det valgte kort.',
      life: 4000,
    })
    return
  }
  emit('update:selectedId', sheet.id === props.selectedId ? undefined : sheet.id)
}

const failed = (error: unknown, fallback: string) => {
  const detail = (error as { response?: { data?: { error?: unknown } } })?.response?.data?.error
  if (detail && typeof detail === 'object') {
    // The API answers 422 with a field map; showing it beside the field beats a toast that
    // disappears before an operator has finished reading it.
    fieldErrors.value = detail as Record<string, string>
    return
  }
  toast.add({
    severity: 'error',
    summary: fallback,
    detail: typeof detail === 'string' ? detail : undefined,
    life: 5000,
  })
}

/**
 * Suggest the next sheet name in a set.
 *
 * "Kort 1", "Kort 2", … because a sheet must be named to exist, and an operator adding fifteen of
 * them should not have to invent fifteen names before drawing anything. Renaming is one field away.
 */
const nextSheetName = (set: Kortsaet) => `Kort ${set.kort.length + 1}`

const createSheet = async (set: Kortsaet) => {
  saving.value = true
  try {
    const response = await http.post('/kort', { kortsaetId: set.id, name: nextSheetName(set) }, { withCredentials: true })
    emit('saved')
    // Select the new sheet, so the operator lands in the editor they were reaching for.
    emit('update:selectedId', response.data?.kortId)
  } catch (error) {
    failed(error, 'Kunne ikke oprette kortet')
  } finally {
    saving.value = false
  }
}

const saveSheet = async () => {
  const sheet = selected.value
  if (!sheet) return
  saving.value = true
  fieldErrors.value = {}
  try {
    await http.put(
      `/kort/${sheet.id}`,
      {
        name: draft.value.name,
        // An empty format is omitted rather than sent: "" is not one of the four values and the
        // API would refuse it, and a sheet whose format is not yet decided is normal.
        ...(draft.value.format ? { format: draft.value.format } : {}),
        note: draft.value.note,
        kortsaetId: draft.value.kortsaetId,
      },
      { withCredentials: true },
    )
    emit('saved')
  } catch (error) {
    failed(error, 'Kunne ikke gemme kortet')
  } finally {
    saving.value = false
  }
}

const deleteSheet = async () => {
  const sheet = selected.value
  if (!sheet) return
  saving.value = true
  try {
    await http.delete(`/kort/${sheet.id}`, { withCredentials: true })
    emit('update:selectedId', undefined)
    emit('saved')
  } catch (error) {
    failed(error, 'Kunne ikke slette kortet')
  } finally {
    saving.value = false
  }
}

/**
 * Persist a set's sheet order after a drag.
 *
 * vuedraggable has already reordered the array, so the screen is in the new order; this tells the
 * server. One request for the whole set, because a drag is one gesture.
 */
const saveOrder = async (set: Kortsaet) => {
  reordering.value = false
  try {
    await http.put(`/kortsaet/${set.id}/kort`, { kortIds: set.kort.map((sheet) => sheet.id) }, { withCredentials: true })
  } catch (error) {
    failed(error, 'Kunne ikke gemme rækkefølgen')
  } finally {
    // Refresh either way: on success to pick up the server's order, and on failure so the list
    // snaps back to what was actually saved rather than showing an order that never was.
    emit('saved')
  }
}

const cancelEdit = () => loadDraft(selected.value)

const close = () => {
  if (dirty.value) {
    toast.add({
      severity: 'warn',
      summary: 'Ugemte ændringer',
      detail: 'Gem eller annullér ændringerne først.',
      life: 4000,
    })
    return
  }
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    :modal="false"
    position="right"
    header="Kort"
    :style="{ width: '26rem', maxHeight: '92vh' }"
    :draggable="false"
    @update:visible="close"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-600">Kortene vi printer og deler ud. Vælg et kort for at se dets poster på kortet.</p>

      <div v-if="sets.length === 0" class="text-sm text-gray-500">Der er ingen kortsæt endnu.</div>

      <div v-for="set in sets" :key="set.id" class="space-y-1">
        <div class="flex items-center justify-between">
          <div class="font-medium">
            {{ set.name }}
            <!-- The team-type marking, shown because it is what the hej-app matches on: an
                 operator needs to see which set is "the spejder set" without opening it. -->
            <span v-if="set.teamType" class="ml-1 text-xs text-gray-500">({{ teamTypeLabel(set.teamType) }})</span>
          </div>
          <Button icon="pi pi-plus" text size="small" :disabled="saving || dirty" @click="createSheet(set)" />
        </div>

        <draggable :list="set.kort" handle=".kort-handle" item-key="id" @start="reordering = true" @end="saveOrder(set)">
          <template #item="{ element: sheet }">
            <div
              class="flex items-center gap-2 rounded px-2 py-1 text-sm"
              :class="sheet.id === selectedId ? 'bg-blue-50 ring-1 ring-blue-300' : 'hover:bg-gray-50'"
            >
              <i class="kort-handle pi pi-bars cursor-move text-xs text-gray-400" />
              <button class="flex-1 truncate text-left" @click="select(sheet)">{{ sheet.name }}</button>
              <span class="text-xs text-gray-400">{{ sheet.checkpointIds.length }}</span>
            </div>
          </template>
        </draggable>

        <div v-if="set.kort.length === 0" class="px-2 text-xs text-gray-400">Ingen kort i sættet</div>
      </div>

      <!-- Sheets whose set is gone. Normally absent; shown so a mis-assigned sheet cannot become
           invisible, which is the whole reason the API returns them separately. -->
      <div v-if="orphans.length" class="space-y-1">
        <div class="font-medium text-amber-700">Uden kortsæt</div>
        <div
          v-for="sheet in orphans"
          :key="sheet.id"
          class="flex items-center gap-2 rounded px-2 py-1 text-sm"
          :class="sheet.id === selectedId ? 'bg-blue-50 ring-1 ring-blue-300' : 'hover:bg-gray-50'"
        >
          <button class="flex-1 truncate text-left" @click="select(sheet)">{{ sheet.name }}</button>
        </div>
      </div>

      <!-- The editor for the selected sheet. -->
      <div v-if="selected" class="space-y-3 border-t pt-3">
        <div>
          <label class="block text-sm text-gray-700">Navn</label>
          <InputText v-model="draft.name" class="w-full" />
          <small v-if="fieldErrors.name" class="text-red-600">{{ fieldErrors.name }}</small>
        </div>

        <div>
          <label class="block text-sm text-gray-700">Format</label>
          <Select
            v-model="draft.format"
            :options="formatOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Ikke valgt"
            class="w-full"
          />
          <small v-if="fieldErrors.format" class="text-red-600">{{ fieldErrors.format }}</small>
        </div>

        <div>
          <label class="block text-sm text-gray-700">Kortsæt</label>
          <Select v-model="draft.kortsaetId" :options="sets" optionLabel="name" optionValue="id" class="w-full" />
        </div>

        <div>
          <label class="block text-sm text-gray-700">Note</label>
          <Textarea v-model="draft.note" rows="2" class="w-full" autoResize />
        </div>

        <div class="flex items-center justify-between">
          <Button label="Slet" severity="danger" text size="small" :disabled="saving" @click="deleteSheet" />
          <div class="flex gap-2">
            <Button label="Annullér" severity="secondary" text size="small" :disabled="saving || !dirty" @click="cancelEdit" />
            <Button label="Gem" size="small" :loading="saving" :disabled="saving || !dirty" @click="saveSheet" />
          </div>
        </div>
      </div>
    </div>
  </Dialog>
</template>
