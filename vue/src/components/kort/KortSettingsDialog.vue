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
import { useDeferredApply } from '@/composables/useDeferredApply'
import {
  checkpointsWithoutMap,
  formatOptions,
  groupSelectionState,
  isDegenerate,
  orderPicks,
  sameExtent,
  teamTypeLabel,
  teamTypeOptions,
  toggleGroupSelection,
  type Extent,
  type Kort,
  type Kortsaet,
  type KortPayload,
  type TeamType,
} from '@/composables/kort'

/** A checkgroup with its checkpoints, as `/checkgroups` sends it. */
export interface PickerCheckgroup {
  id: string
  name: string
  checkpoints: { id: string; name: string; latitude: number | null; longitude: number | null }[]
}

const props = defineProps<{
  visible: boolean
  payload?: KortPayload
  /**
   * The year's checkgroups, for the picker.
   *
   * Passed in rather than fetched here: the view has them already, and a second fetch would be a
   * second cache entry to keep in step with the same live tokens.
   */
  checkgroups?: PickerCheckgroup[]
  /** The sheet whose checkpoints are highlighted on the map. */
  selectedId?: string
  /**
   * A rectangle the operator has just drawn on the map.
   *
   * The view owns the Leaflet map, so it does the picking and reports the result here. `seq`
   * increments per pick so that drawing the same rectangle twice still registers — comparing the
   * coordinates would swallow the second one.
   */
  pick?: { extent: Extent; seq: number } | null
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
  /** Ask the view to arm (or disarm) two-click corner picking. */
  (e: 'update:picking', value: boolean): void
  /** The rectangles to draw right now — the draft, so an unsaved extent is still visible. */
  (e: 'update:extentsPreview', value: Extent[]): void
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

/**
 * Has the operator changed something that is not saved?
 *
 * The name is compared **trimmed**, matching what the server stores. Without that, typing a trailing
 * space would leave the draft permanently unequal to the saved row: the save would succeed, the
 * server would trim, and the dialog would insist there were unsaved changes forever — which, now
 * that unsaved changes pause live updates, would also freeze the map. Client-side normalisation has
 * to match the server's for exactly the fields it touches.
 */
const dirty = computed(() => {
  const sheet = selected.value
  if (!sheet) return false
  return (
    draft.value.name.trim() !== sheet.name ||
    draft.value.format !== (sheet.format ?? '') ||
    draft.value.note !== sheet.note ||
    draft.value.kortsaetId !== sheet.kortsaetId
  )
})

// Reordering counts as dirty too: the list on screen is in an order the server does not have yet.
const reordering = ref(false)

const select = (sheet: Kort) => {
  if (anyDirty.value) {
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
        name: draft.value.name.trim(),
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

// --- the checkpoint picker (task 128) ---
//
// This is the half of the feature the hej-app actually needs: which checkpoints are drawn on which
// sheet. Extents are cosmetic by comparison — a sheet with no rectangle still reveals the right
// checkpoints, a sheet with no checkpoint list reveals nothing.

/** The selected sheet's checkpoints, as a local set the tick-boxes drive. */
const picked = ref<Set<string>>(new Set())

/** Unsaved tick-box changes. Compared as sets, since order carries no meaning. */
const picksDirty = computed(() => {
  const sheet = selected.value
  if (!sheet) return false
  if (picked.value.size !== sheet.checkpointIds.length) return true
  return sheet.checkpointIds.some((id) => !picked.value.has(id))
})

const togglePick = (checkpointId: string) => {
  const next = new Set(picked.value)
  if (next.has(checkpointId)) next.delete(checkpointId)
  else next.add(checkpointId)
  picked.value = next
}

const groupState = (group: PickerCheckgroup): 'all' | 'some' | 'none' => groupSelectionState(group, picked.value)

/**
 * Tick or untick a whole checkgroup.
 *
 * The thing that keeps data entry to minutes rather than an hour: a sheet almost always carries
 * whole checkgroups, because a checkgroup is revealed as a whole and splitting one across two
 * sheets is the mistake task 133 warns about. Partial selections stay possible — a skitse is
 * exactly that — they are just not the common case.
 */
const toggleGroup = (group: PickerCheckgroup) => {
  picked.value = toggleGroupSelection(group, picked.value)
}

const savePicks = async () => {
  const sheet = selected.value
  if (!sheet) return
  saving.value = true
  try {
    await http.put(
      `/kort/${sheet.id}/checkpoints`,
      // Sent in checkgroup order rather than tick order, so the stored list is stable and
      // re-saving an unchanged selection stays a no-op on the server.
      { checkpointIds: orderedPicks.value },
      { withCredentials: true },
    )
    emit('saved')
  } catch (error) {
    failed(error, 'Kunne ikke gemme posterne')
  } finally {
    saving.value = false
  }
}

const orderedPicks = computed(() => orderPicks(props.checkgroups ?? [], picked.value))

const cancelPicks = () => {
  picked.value = new Set(selected.value?.checkpointIds ?? [])
}

/** Checkpoints with no position cannot be drawn, so they are flagged rather than blocked. */
const hasPosition = (cp: PickerCheckgroup['checkpoints'][number]) => cp.latitude != null && cp.longitude != null

// --- what is on no sheet at all ---

const allCheckpointIds = computed(() => (props.checkgroups ?? []).flatMap((group) => group.checkpoints.map((cp) => cp.id)))

const checkpointName = (id: string) => {
  for (const group of props.checkgroups ?? []) {
    const found = group.checkpoints.find((cp) => cp.id === id)
    if (found) return found.name
  }
  return id
}

/**
 * Per set, the checkpoints that appear on none of its sheets.
 *
 * Per set and not overall, because the two mistakes are different: a checkpoint missing from the
 * crew maps is a driver with no way to find it, while one missing from the patrol maps is a patrol
 * that will never be sent there. An overall list could not tell an operator which they are looking
 * at.
 */
const unassignedBySet = computed(() =>
  sets.value.map((set) => ({
    set,
    missing: checkpointsWithoutMap(set, allCheckpointIds.value),
  })),
)

// --- map sets (task 129) ---
//
// Edited inline in the list rather than in a nested dialog: a set has two fields, and a modal over
// a modal over the map would bury the thing the whole screen is about.

/** The set being edited, or 'new' while creating one. */
const editingSetId = ref<string | 'new' | undefined>(undefined)
const setDraft = ref<{ name: string; teamType: TeamType | null }>({ name: '', teamType: null })

const editSet = (set: Kortsaet) => {
  editingSetId.value = set.id
  setDraft.value = { name: set.name, teamType: set.teamType }
  fieldErrors.value = {}
}

const newSet = () => {
  editingSetId.value = 'new'
  // Not defaulted to `patrulje`: an unmarked set is the commonest one, and a default here would
  // quietly mark a crew set as the spejder set — which the hej-app would then serve to patrols.
  setDraft.value = { name: '', teamType: null }
  fieldErrors.value = {}
}

const cancelSetEdit = () => {
  editingSetId.value = undefined
  fieldErrors.value = {}
}

/** Unsaved set edits, so they join the same dirty guard as everything else. */
const setDirty = computed(() => {
  if (editingSetId.value === 'new') return setDraft.value.name.trim() !== '' || setDraft.value.teamType !== null
  const set = sets.value.find((candidate) => candidate.id === editingSetId.value)
  if (!set) return false
  // Trimmed, for the same reason as a sheet's name: the server trims, and an untrimmed comparison
  // would leave the set editor stuck open with nothing actually unsaved.
  return setDraft.value.name.trim() !== set.name || setDraft.value.teamType !== set.teamType
})

const saveSet = async () => {
  saving.value = true
  fieldErrors.value = {}
  try {
    const body = { name: setDraft.value.name.trim(), teamType: setDraft.value.teamType }
    if (editingSetId.value === 'new') {
      await http.post('/kortsaet', body, { withCredentials: true })
    } else {
      // A whole-record PUT, matching the API: sending only what changed could not express clearing
      // the team type, because an absent field and "no team type" would look the same.
      await http.put(`/kortsaet/${editingSetId.value}`, body, { withCredentials: true })
    }
    editingSetId.value = undefined
    emit('saved')
  } catch (error) {
    failed(error, 'Kunne ikke gemme kortsættet')
  } finally {
    saving.value = false
  }
}

const deleteSet = async (set: Kortsaet) => {
  saving.value = true
  fieldErrors.value = {}
  try {
    await http.delete(`/kortsaet/${set.id}`, { withCredentials: true })
    editingSetId.value = undefined
    emit('saved')
  } catch (error) {
    // The API refuses a set that still holds sheets, with a Danish message naming what to do about
    // it. Surfaced as a toast rather than a field error, because it is about the set's contents
    // rather than anything the operator typed.
    const detail = (error as { response?: { data?: { error?: unknown } } })?.response?.data?.error
    const message = detail && typeof detail === 'object' ? Object.values(detail as Record<string, string>)[0] : undefined
    toast.add({
      severity: 'warn',
      summary: 'Kortsættet blev ikke slettet',
      detail: message ?? 'Kunne ikke slette kortsættet.',
      life: 6000,
    })
  } finally {
    saving.value = false
  }
}

/**
 * Persist the order of the sets after a drag.
 *
 * `PUT /api/kortsaet` — the collection, not a `/sorted` path: httprouter cannot have a static
 * segment beside `/:id`, and "kortsæt" is its own plural in Danish, so there was no plural to move
 * the route to.
 */
const saveSetOrder = async () => {
  reordering.value = false
  try {
    await http.put('/kortsaet', { kortsaetIds: sets.value.map((set) => set.id) }, { withCredentials: true })
  } catch (error) {
    failed(error, 'Kunne ikke gemme rækkefølgen')
  } finally {
    emit('saved')
  }
}

// --- extents (task 130) ---
//
// The ground a sheet shows: none for a skitse, one for a normal sheet, two for a double-sided one.
// The two are simply two areas — nothing here records which is the front, and the checkpoints are
// not split per side, because both sides are handed over at once.

const MAX_EXTENTS = 2

/** The extents being edited. A copy, like the fields: the cached value is drawn on the map. */
const draftExtents = ref<Extent[]>([])
/** Which slot the next drawn rectangle fills, or null when not picking. */
const pickingIndex = ref<number | null>(null)

const extentsDirty = computed(() => {
  const sheet = selected.value
  if (!sheet) return false
  if (draftExtents.value.length !== sheet.extents.length) return true
  return draftExtents.value.some((extent, i) => !sameExtent(extent, sheet.extents[i]))
})

// The view draws the *draft*, so an extent that has been drawn but not saved is still visible.
// Anything else would make "Gem områder" feel like it was what drew the rectangle.
watch(draftExtents, (extents) => emit('update:extentsPreview', extents), { deep: true, immediate: true })
watch(pickingIndex, (index) => emit('update:picking', index !== null))

/** Arm picking for a slot, adding one if this is a new area. */
const pickExtent = (index: number) => {
  pickingIndex.value = pickingIndex.value === index ? null : index
}

const addExtent = () => {
  if (draftExtents.value.length >= MAX_EXTENTS) return
  // Armed straight away for the slot that does not exist yet: "Tilføj område" means the operator is
  // about to draw one, so making them press a second button first would be ceremony.
  pickingIndex.value = draftExtents.value.length
}

const removeExtent = (index: number) => {
  draftExtents.value = draftExtents.value.filter((_, i) => i !== index)
  pickingIndex.value = null
}

// A rectangle arrived from the map.
watch(
  () => props.pick?.seq,
  () => {
    const pick = props.pick
    if (!pick || pickingIndex.value === null) return
    const next = [...draftExtents.value]
    next[pickingIndex.value] = pick.extent
    draftExtents.value = next
    // One rectangle per arming: staying armed would turn the next map click, meant for something
    // else, into a redrawn extent.
    pickingIndex.value = null
  },
)

const saveExtents = async () => {
  const sheet = selected.value
  if (!sheet) return
  // Caught here rather than after a round trip: the API refuses it too, but telling the operator
  // straight away means "vælg to forskellige hjørner" arrives while they still remember clicking.
  const flat = draftExtents.value.findIndex(isDegenerate)
  if (flat !== -1) {
    fieldErrors.value = { extents: 'Området har ingen udstrækning — vælg to forskellige hjørner' }
    return
  }
  saving.value = true
  fieldErrors.value = {}
  try {
    await http.put(`/kort/${sheet.id}`, { extents: draftExtents.value }, { withCredentials: true })
    emit('saved')
  } catch (error) {
    failed(error, 'Kunne ikke gemme områderne')
  } finally {
    saving.value = false
  }
}

const cancelExtents = () => {
  draftExtents.value = (selected.value?.extents ?? []).map((extent) => ({ ...extent }))
  pickingIndex.value = null
}

/** A short, readable rendering of a rectangle's corners. */
const extentLabel = (extent: Extent) =>
  `${extent.northWest.latitude.toFixed(4)}, ${extent.northWest.longitude.toFixed(4)} → ` +
  `${extent.southEast.latitude.toFixed(4)}, ${extent.southEast.longitude.toFixed(4)}`

// --- unsaved state, all of it ---
//
// Declared here, after every source, because the emit below runs immediately: a computed that
// referenced one of these before its `const` was initialised would throw during setup.

/** Anything unsaved: a field, a tick-box, an extent, a set being edited, or a drag not yet saved. */
const anyDirty = computed(
  () => dirty.value || picksDirty.value || extentsDirty.value || setDirty.value || reordering.value,
)

/**
 * Load every buffer from a sheet: the fields, the tick-boxes and the rectangles.
 *
 * One function because they are one decision — "show me this sheet" — and three separate watchers
 * on the same source is how two of them end up with different guards.
 */
const loadBuffers = (sheet: Kort) => {
  loadDraft(sheet)
  picked.value = new Set(sheet.checkpointIds)
  draftExtents.value = sheet.extents.map((extent) => ({ ...extent }))
  pickingIndex.value = null
}

/**
 * Follow the selected sheet — unless the operator has unsaved work.
 *
 * This is the bug that made task 131 worth doing, and it is not the one the task description
 * anticipated. The dialog reads the sheets straight from the live cache, so a payload arriving while
 * someone typed used to re-run the load and **wipe the field under the cursor**: another operator
 * renaming any sheet, or this operator's own save of a *different* sheet, was enough. Deferring only
 * the Leaflet markers would have left that untouched.
 *
 * The same composable as the map, for the same reason: watching the condition cannot miss an exit.
 * Selection changes come through here too, which is safe because `select()` refuses to change the
 * selection while dirty — so a selection change is always a clean moment by construction.
 */
useDeferredApply(selected, anyDirty, loadBuffers)

// The view pauses applying live payloads while this is true (task 131), and the guards above use it
// to refuse switching sheets or closing rather than discarding work.
watch(anyDirty, (value) => emit('update:dirty', value), { immediate: true })

const close = () => {
  if (anyDirty.value) {
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
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-600">Kortene vi printer og deler ud.</p>
        <Button label="Nyt sæt" icon="pi pi-plus" text size="small" :disabled="saving || anyDirty" @click="newSet" />
      </div>

      <!-- Creating a set. Same two fields as editing one; the team type deliberately starts empty,
           because the unmarked crew set is the commonest and a default would silently mark a set as
           the spejder set. -->
      <div v-if="editingSetId === 'new'" class="space-y-2 rounded border border-blue-200 bg-blue-50 p-2">
        <div>
          <label class="block text-xs text-gray-700">Navn</label>
          <InputText v-model="setDraft.name" class="w-full" placeholder="fx Patruljer" autofocus />
          <small v-if="fieldErrors.name" class="text-red-600">{{ fieldErrors.name }}</small>
        </div>
        <div>
          <label class="block text-xs text-gray-700">Holdtype</label>
          <Select
            v-model="setDraft.teamType"
            :options="teamTypeOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
          <small class="text-gray-500">Afgør hvilke kort hej-appen viser til fx patruljer.</small>
          <small v-if="fieldErrors.teamType" class="block text-red-600">{{ fieldErrors.teamType }}</small>
        </div>
        <div class="flex justify-end gap-2">
          <Button label="Annullér" severity="secondary" text size="small" :disabled="saving" @click="cancelSetEdit" />
          <Button label="Opret" size="small" :loading="saving" :disabled="saving || !setDraft.name.trim()" @click="saveSet" />
        </div>
      </div>

      <div v-if="sets.length === 0 && editingSetId !== 'new'" class="text-sm text-gray-500">
        Der er ingen kortsæt endnu. Opret fx „Patruljer” og „Crew”.
      </div>

      <draggable :list="sets" handle=".kortsaet-handle" item-key="id" @start="reordering = true" @end="saveSetOrder">
        <template #item="{ element: set }">
          <div class="mb-3 space-y-1">
            <!-- Editing this set. -->
            <div v-if="editingSetId === set.id" class="space-y-2 rounded border border-blue-200 bg-blue-50 p-2">
              <div>
                <label class="block text-xs text-gray-700">Navn</label>
                <InputText v-model="setDraft.name" class="w-full" autofocus />
                <small v-if="fieldErrors.name" class="text-red-600">{{ fieldErrors.name }}</small>
              </div>
              <div>
                <label class="block text-xs text-gray-700">Holdtype</label>
                <Select
                  v-model="setDraft.teamType"
                  :options="teamTypeOptions"
                  optionLabel="label"
                  optionValue="value"
                  class="w-full"
                />
                <small v-if="fieldErrors.teamType" class="block text-red-600">{{ fieldErrors.teamType }}</small>
              </div>
              <div class="flex items-center justify-between">
                <Button label="Slet sæt" severity="danger" text size="small" :disabled="saving" @click="deleteSet(set)" />
                <div class="flex gap-2">
                  <Button label="Annullér" severity="secondary" text size="small" :disabled="saving" @click="cancelSetEdit" />
                  <Button label="Gem" size="small" :loading="saving" :disabled="saving || !setDirty" @click="saveSet" />
                </div>
              </div>
            </div>

            <div v-else class="flex items-center justify-between">
              <div class="flex min-w-0 items-center gap-2">
                <i class="kortsaet-handle pi pi-bars cursor-move text-xs text-gray-400" />
                <div class="truncate font-medium">
                  {{ set.name }}
                  <!-- The team-type marking, shown because it is what the hej-app matches on: an
                       operator needs to see which set is “the spejder set” without opening it. -->
                  <span v-if="set.teamType" class="ml-1 text-xs text-gray-500">({{ teamTypeLabel(set.teamType) }})</span>
                </div>
              </div>
              <div class="flex items-center">
                <Button icon="pi pi-pencil" text size="small" :disabled="saving || anyDirty" @click="editSet(set)" />
                <Button icon="pi pi-plus" text size="small" :disabled="saving || anyDirty" @click="createSheet(set)" />
              </div>
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
        </template>
      </draggable>

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

        <!-- The ground this sheet shows. Zero areas is normal — a skitse has none worth recording;
             two means a double-sided sheet, and they are simply two areas with no front or back. -->
        <div class="space-y-1 border-t pt-3">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium">Områder på kortet</div>
            <div class="flex items-center gap-2">
              <Button label="Annullér" severity="secondary" text size="small" :disabled="saving || !extentsDirty" @click="cancelExtents" />
              <Button label="Gem områder" size="small" :loading="saving" :disabled="saving || !extentsDirty" @click="saveExtents" />
            </div>
          </div>

          <div v-if="draftExtents.length === 0 && pickingIndex === null" class="text-xs text-gray-400">
            Ingen områder — fx en skitse.
          </div>

          <div v-for="(extent, index) in draftExtents" :key="index" class="flex items-center gap-2 text-xs">
            <span class="flex-1 truncate text-gray-600">{{ extentLabel(extent) }}</span>
            <Button
              :label="pickingIndex === index ? 'Klik på kortet…' : 'Vælg på kort'"
              :severity="pickingIndex === index ? 'warn' : 'secondary'"
              text
              size="small"
              @click="pickExtent(index)"
            />
            <Button icon="pi pi-trash" severity="danger" text size="small" @click="removeExtent(index)" />
          </div>

          <div v-if="pickingIndex !== null && pickingIndex >= draftExtents.length" class="text-xs text-amber-700">
            Klik to modsatte hjørner på kortet.
          </div>

          <small v-if="fieldErrors.extents" class="block text-red-600">{{ fieldErrors.extents }}</small>

          <Button
            v-if="draftExtents.length < 2"
            label="Tilføj område"
            icon="pi pi-plus"
            text
            size="small"
            :disabled="saving"
            @click="addExtent"
          />
        </div>

        <!-- The checkpoints drawn on this sheet. The part the hej-app reads: this list is what may
             be revealed once the sheet is known to be in a team's hands. -->
        <div class="space-y-1 border-t pt-3">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium">Poster på kortet</div>
            <div class="flex items-center gap-2">
              <span class="text-xs text-gray-500">{{ picked.size }} valgt</span>
              <Button label="Annullér" severity="secondary" text size="small" :disabled="saving || !picksDirty" @click="cancelPicks" />
              <Button label="Gem poster" size="small" :loading="saving" :disabled="saving || !picksDirty" @click="savePicks" />
            </div>
          </div>

          <div v-if="(checkgroups ?? []).length === 0" class="text-xs text-gray-400">Ingen poster fundet</div>

          <div v-for="group in checkgroups ?? []" :key="group.id" class="space-y-0.5">
            <!-- Select-all per checkgroup. A sheet almost always carries whole checkgroups, because
                 a checkgroup is revealed as a whole — so this is the button that turns fifteen
                 sheets from an hour's work into a few minutes. -->
            <button class="flex w-full items-center gap-2 rounded px-1 py-0.5 text-left text-xs font-medium hover:bg-gray-50" @click="toggleGroup(group)">
              <i
                class="pi text-xs"
                :class="{
                  'pi-check-square text-blue-600': groupState(group) === 'all',
                  'pi-minus-circle text-blue-400': groupState(group) === 'some',
                  'pi-stop text-gray-300': groupState(group) === 'none'
                }"
              />
              <span class="flex-1 truncate">{{ group.name }}</span>
            </button>

            <button
              v-for="cp in group.checkpoints"
              :key="cp.id"
              class="flex w-full items-center gap-2 rounded py-0.5 pl-5 pr-1 text-left text-sm hover:bg-gray-50"
              @click="togglePick(cp.id)"
            >
              <i class="pi text-xs" :class="picked.has(cp.id) ? 'pi-check-square text-blue-600' : 'pi-stop text-gray-300'" />
              <span class="flex-1 truncate">{{ cp.name }}</span>
              <!-- A checkpoint with no position can be assigned; it just cannot be drawn. Flagged
                   rather than blocked, since the sheet may be drawn before the pin is placed. -->
              <i v-if="!hasPosition(cp)" class="pi pi-exclamation-triangle text-xs text-amber-500" title="Ingen placering endnu" />
            </button>
          </div>
        </div>
      </div>

      <!-- Checkpoints on no sheet of a set. Per set, because a post missing from the crew maps is a
           driver who cannot find it, while one missing from the patrol maps is a patrol that will
           never be sent there. -->
      <div v-if="unassignedBySet.some((entry) => entry.missing.length)" class="space-y-2 border-t pt-3">
        <div class="text-sm font-medium">Ikke på noget kort</div>
        <div v-for="entry in unassignedBySet" :key="entry.set.id">
          <div v-if="entry.missing.length" class="text-xs">
            <span class="font-medium">{{ entry.set.name }}:</span>
            <span class="text-amber-700">{{ entry.missing.map(checkpointName).join(', ') }}</span>
          </div>
        </div>
      </div>
    </div>
  </Dialog>
</template>
