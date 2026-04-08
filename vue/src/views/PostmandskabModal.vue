<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'

import DayTimePicker from '@/components/DayTimePicker.vue'
import { dddhhmm, hhmm } from '@/composables/datefilters'

const props = withDefaults(
  defineProps<{
    checkgroupId?: string
  }>(),
  {
    checkgroupId: ''
  }
)
const emit = defineEmits<{
  saved: [id: String]
  canceled: null
}>()

const toast = useToast()

onMounted(() => load())

watch(
  () => props.checkgroupId,
  () => {
    // Clear local state from previous session
    newRows.value = new Map()
    markedForDeletion.value = new Set()
    timeEdits.value = new Map()
    editingExisting.value = new Set()
    load()
  }
)

const checkgroup = ref<any>({})
const checkpoints = ref<any[]>([])
const availablePersonnel = ref<any[]>([])
const assignedPersonnel = ref<any[]>([])
const year = ref<any>({})

let newRowCounter = 0

// Pending additions: checkpoint id -> array of { key, userId, editing, start, end }
const newRows = ref<Map<string, any[]>>(new Map())

// IDs of existing assigned personnel marked for deletion
const markedForDeletion = ref<Set<string>>(new Set())

// Time edits for existing rows: checkpersonnel id -> { start, end }
const timeEdits = ref<Map<string, { start: Date | null; end: Date | null }>>(new Map())

// IDs of existing rows currently in edit mode
const editingExisting = ref<Set<string>>(new Set())

const saving = ref(false)

const personById = computed(() => {
  const map = new Map()
  for (const p of availablePersonnel.value) {
    map.set(p.id, p)
  }
  return map
})

const checkpointById = computed(() => {
  const map = new Map()
  for (const cp of checkpoints.value) {
    map.set(cp.id, cp)
  }
  return map
})

const personName = (userId: string) => {
  const p = personById.value.get(userId)
  return p ? p.name : userId
}

// Personnel not yet assigned to any checkpoint in this checkgroup.
// Accepts the current pending row's key so that row's own selection stays in the list.
const unassignedPersonnel = (currentRowKey: number | null = null) => {
  const assignedUserIds = new Set(assignedPersonnel.value.filter((ap: any) => !markedForDeletion.value.has(ap.id)).map((ap: any) => ap.userId))
  for (const pendingList of newRows.value.values()) {
    for (const pending of pendingList) {
      if (pending.userId && pending.key !== currentRowKey) {
        assignedUserIds.add(pending.userId)
      }
    }
  }
  return availablePersonnel.value.filter((p: any) => !assignedUserIds.has(p.id))
}

// Whether there are any unsaved changes
const hasChanges = computed(() => {
  if (markedForDeletion.value.size > 0) return true
  if (timeEdits.value.size > 0) return true
  for (const list of newRows.value.values()) {
    if (list.length > 0) return true
  }
  return false
})

// Default start/end from checkpoint opening hours
const defaultStart = (checkpointId: string): Date | null => {
  const cp = checkpointById.value.get(checkpointId)
  if (cp && cp.openFrom) {
    const d = new Date(cp.openFrom)
    if (!isNaN(d.getTime()) && d.getTime() > 0) return d
  }
  return null
}

const defaultEnd = (checkpointId: string): Date | null => {
  const cp = checkpointById.value.get(checkpointId)
  if (cp && cp.openUntil) {
    const d = new Date(cp.openUntil)
    if (!isNaN(d.getTime()) && d.getTime() > 0) return d
  }
  return null
}

// Get the effective start/end for an existing row (with possible local edits)
const effectiveStart = (row: any): Date | null => {
  const edit = timeEdits.value.get(row.id)
  if (edit && edit.start) return edit.start
  if (row.start) {
    const d = new Date(row.start)
    if (!isNaN(d.getTime()) && d.getTime() > 0) return d
  }
  return defaultStart(row.checkpointId)
}

const effectiveEnd = (row: any): Date | null => {
  const edit = timeEdits.value.get(row.id)
  if (edit && edit.end) return edit.end
  if (row.end) {
    const d = new Date(row.end)
    if (!isNaN(d.getTime()) && d.getTime() > 0) return d
  }
  return defaultEnd(row.checkpointId)
}

// Build a flat list of rows for the DataTable
const personnel = computed(() => {
  const rows: any[] = []
  const assignedByCheckpoint = new Map<string, any[]>()

  for (const ap of assignedPersonnel.value) {
    const cpId = ap.checkpointId
    if (!assignedByCheckpoint.has(cpId)) {
      assignedByCheckpoint.set(cpId, [])
    }
    assignedByCheckpoint.get(cpId)!.push(ap)
  }

  for (const cp of checkpoints.value) {
    const assigned = assignedByCheckpoint.get(cp.id) || []
    const pendingList = newRows.value.get(cp.id) || []

    const visibleAssigned = assigned.filter((ap: any) => !markedForDeletion.value.has(ap.id))
    const deletedAssigned = assigned.filter((ap: any) => markedForDeletion.value.has(ap.id))

    if (visibleAssigned.length === 0 && pendingList.length === 0 && deletedAssigned.length === 0) {
      rows.push({
        id: 'empty-' + cp.id,
        checkpointId: cp.id,
        checkpointName: cp.name,
        name: '',
        start: null,
        end: null,
        isEmpty: true,
        isNew: false,
        isEditing: false,
        isEditingExisting: false,
        isDeleted: false,
        newRowKey: null
      })
    } else {
      for (const ap of visibleAssigned) {
        const isEditingThis = editingExisting.value.has(ap.id)
        rows.push({
          id: ap.id,
          checkpointId: cp.id,
          checkpointName: cp.name,
          userId: ap.userId,
          name: personName(ap.userId),
          start: ap.start,
          end: ap.end,
          isEmpty: false,
          isNew: false,
          isEditing: false,
          isEditingExisting: isEditingThis,
          isDeleted: false,
          newRowKey: null
        })
      }
      for (const ap of deletedAssigned) {
        rows.push({
          id: ap.id,
          checkpointId: cp.id,
          checkpointName: cp.name,
          userId: ap.userId,
          name: personName(ap.userId),
          start: ap.start,
          end: ap.end,
          isEmpty: false,
          isNew: false,
          isEditing: false,
          isEditingExisting: false,
          isDeleted: true,
          newRowKey: null
        })
      }
    }

    for (const pending of pendingList) {
      rows.push({
        id: 'new-' + pending.key,
        checkpointId: cp.id,
        checkpointName: cp.name,
        userId: pending.userId,
        name: pending.userId ? personName(pending.userId) : '',
        start: pending.start,
        end: pending.end,
        isEmpty: false,
        isNew: true,
        isEditing: pending.editing,
        isEditingExisting: false,
        isDeleted: false,
        newRowKey: pending.key
      })
    }
  }

  return rows
})

const load = async () => {
  if (!props.checkgroupId) return
  try {
    const rsp = await http.get('/checkgroup/' + props.checkgroupId, { withCredentials: true })
    if (rsp.status === 200) {
      checkgroup.value = rsp.data.checkgroup
      checkpoints.value = rsp.data.checkpoints || []
      availablePersonnel.value = rsp.data.availablePersonnel || []
      assignedPersonnel.value = rsp.data.assignedPersonnel || []
      year.value = rsp.data.year || {}
    }
  } catch (error: any) {
    console.log('error happened', error)
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke hente data',
      detail: error?.response?.data || error.message,
      life: 3000
    })
  }
}

const addPersonRow = (checkpointId: string) => {
  const list = newRows.value.get(checkpointId) || []
  list.push({
    key: ++newRowCounter,
    userId: null,
    editing: true,
    start: defaultStart(checkpointId),
    end: defaultEnd(checkpointId)
  })
  newRows.value.set(checkpointId, list)
  newRows.value = new Map(newRows.value)
}

const cancelNewRow = (checkpointId: string, key: number) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const updated = list.filter((r: any) => r.key !== key)
  if (updated.length === 0) {
    newRows.value.delete(checkpointId)
  } else {
    newRows.value.set(checkpointId, updated)
  }
  newRows.value = new Map(newRows.value)
}

const acceptNewRow = (checkpointId: string, key: number) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const pending = list.find((r: any) => r.key === key)
  if (!pending || !pending.userId) {
    toast.add({
      severity: 'warn',
      summary: 'Vælg en person',
      detail: 'Du skal vælge en person før du kan bekræfte',
      life: 3000
    })
    return
  }
  pending.editing = false
  newRows.value = new Map(newRows.value)
}

const editNewRow = (checkpointId: string, key: number) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const pending = list.find((r: any) => r.key === key)
  if (pending) {
    pending.editing = true
    newRows.value = new Map(newRows.value)
  }
}

const markForDeletion = (id: string) => {
  editingExisting.value.delete(id)
  editingExisting.value = new Set(editingExisting.value)
  markedForDeletion.value.add(id)
  markedForDeletion.value = new Set(markedForDeletion.value)
}

const unmarkForDeletion = (id: string) => {
  markedForDeletion.value.delete(id)
  markedForDeletion.value = new Set(markedForDeletion.value)
}

const startEditExisting = (row: any) => {
  // Initialize time edits with current values
  if (!timeEdits.value.has(row.id)) {
    const s = row.start ? new Date(row.start) : defaultStart(row.checkpointId)
    const e = row.end ? new Date(row.end) : defaultEnd(row.checkpointId)
    timeEdits.value.set(row.id, {
      start: s && !isNaN(s.getTime()) && s.getTime() > 0 ? s : null,
      end: e && !isNaN(e.getTime()) && e.getTime() > 0 ? e : null
    })
    timeEdits.value = new Map(timeEdits.value)
  }
  editingExisting.value.add(row.id)
  editingExisting.value = new Set(editingExisting.value)
}

const cancelEditExisting = (row: any) => {
  editingExisting.value.delete(row.id)
  timeEdits.value.delete(row.id)
  editingExisting.value = new Set(editingExisting.value)
  timeEdits.value = new Map(timeEdits.value)
}

const acceptEditExisting = (row: any) => {
  editingExisting.value.delete(row.id)
  editingExisting.value = new Set(editingExisting.value)
  // timeEdits stays so it's included in save
}

const setNewRowUser = (checkpointId: string, key: number, userId: string) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const pending = list.find((r: any) => r.key === key)
  if (pending) {
    pending.userId = userId
    newRows.value = new Map(newRows.value)
  }
}

const setNewRowStart = (checkpointId: string, key: number, val: Date) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const pending = list.find((r: any) => r.key === key)
  if (pending) {
    pending.start = val
    newRows.value = new Map(newRows.value)
  }
}

const setNewRowEnd = (checkpointId: string, key: number, val: Date) => {
  const list = newRows.value.get(checkpointId)
  if (!list) return
  const pending = list.find((r: any) => r.key === key)
  if (pending) {
    pending.end = val
    newRows.value = new Map(newRows.value)
  }
}

const setExistingStart = (id: string, val: Date) => {
  const edit = timeEdits.value.get(id)
  if (edit) {
    edit.start = val
    timeEdits.value = new Map(timeEdits.value)
  }
}

const setExistingEnd = (id: string, val: Date) => {
  const edit = timeEdits.value.get(id)
  if (edit) {
    edit.end = val
    timeEdits.value = new Map(timeEdits.value)
  }
}

// Save all pending changes (additions + deletions + time edits) to the API
const saveAll = async () => {
  for (const [, list] of newRows.value.entries()) {
    for (const pending of list) {
      if (pending.editing) {
        toast.add({
          severity: 'warn',
          summary: 'Ubekræftede rækker',
          detail: 'Bekræft eller annuller alle nye rækker før du gemmer',
          life: 3000
        })
        return
      }
      if (!pending.userId) {
        toast.add({
          severity: 'warn',
          summary: 'Manglende person',
          detail: 'Alle nye rækker skal have en person valgt',
          life: 3000
        })
        return
      }
    }
  }

  if (editingExisting.value.size > 0) {
    toast.add({
      severity: 'warn',
      summary: 'Ubekræftede rækker',
      detail: 'Bekræft eller annuller alle redigerede rækker før du gemmer',
      life: 3000
    })
    return
  }

  saving.value = true
  let hadError = false

  // Process deletions
  for (const id of markedForDeletion.value) {
    try {
      await http.delete('/checkpersonnel/' + id, { withCredentials: true })
    } catch (error: any) {
      hadError = true
      console.log('error deleting checkpersonnel', error)
      toast.add({
        severity: 'error',
        summary: 'Kunne ikke fjerne person',
        detail: error?.response?.data || error.message,
        life: 3000
      })
    }
  }

  // Process additions (with times)
  for (const [checkpointId, list] of newRows.value.entries()) {
    for (const pending of list) {
      try {
        const payload: any = { checkpointId, userId: pending.userId }
        if (pending.start) payload.start = pending.start
        if (pending.end) payload.end = pending.end
        await http.post('/checkpersonnel', payload, { withCredentials: true })
      } catch (error: any) {
        hadError = true
        console.log('error creating checkpersonnel', error)
        toast.add({
          severity: 'error',
          summary: 'Kunne ikke tilføje person',
          detail: error?.response?.data || error.message,
          life: 3000
        })
      }
    }
  }

  // Process time edits on existing rows
  for (const [id, edit] of timeEdits.value.entries()) {
    if (markedForDeletion.value.has(id)) continue
    try {
      const payload: any = {}
      if (edit.start) payload.start = edit.start
      if (edit.end) payload.end = edit.end
      await http.put('/checkpersonnel/' + id, payload, { withCredentials: true })
    } catch (error: any) {
      hadError = true
      console.log('error updating checkpersonnel time', error)
      toast.add({
        severity: 'error',
        summary: 'Kunne ikke opdatere tidspunkt',
        detail: error?.response?.data || error.message,
        life: 3000
      })
    }
  }

  saving.value = false

  if (!hadError) {
    toast.add({
      severity: 'success',
      summary: 'Ændringer gemt',
      detail: 'OK',
      life: 2000
    })
  }

  // Clear local state and reload
  newRows.value = new Map()
  markedForDeletion.value = new Set()
  timeEdits.value = new Map()
  editingExisting.value = new Set()
  await load()

  if (!hadError) {
    emit('saved', props.checkgroupId)
  }
}

const cancel = () => emit('canceled')
</script>

<template>
  <div>
    <h2 class="text-lg font-bold mb-3">{{ checkgroup.name }}</h2>

    <DataTable :value="personnel" rowGroupMode="subheader" groupRowsBy="checkpointName" sortMode="single" sortField="checkpointName" :sortOrder="1" scrollable scrollHeight="400px" tableStyle="min-width: 50rem">
      <Column field="checkpointName" header="Post"></Column>

      <Column field="name" header="Navn" style="min-width: 200px">
        <template #body="{ data }">
          <span v-if="data.isEmpty" class="italic text-surface-400">Intet postmandskab</span>
          <template v-else-if="data.isNew && data.isEditing">
            <Select :modelValue="newRows.get(data.checkpointId)?.find((r: any) => r.key === data.newRowKey)?.userId" @update:modelValue="(val: string) => setNewRowUser(data.checkpointId, data.newRowKey, val)" :options="unassignedPersonnel(data.newRowKey)" optionLabel="name" optionValue="id" placeholder="Vælg person" class="w-full" filter />
          </template>
          <span v-else-if="data.isDeleted" class="line-through text-surface-400">{{ data.name }}</span>
          <span v-else-if="data.isNew && !data.isEditing" class="text-green-600 font-medium">{{ data.name }}</span>
          <span v-else>{{ data.name }}</span>
        </template>
      </Column>

      <Column field="start" header="Start" style="min-width: 200px">
        <template #body="{ data }">
          <!-- New row editing -->
          <template v-if="data.isNew && data.isEditing">
            <DayTimePicker :value="newRows.get(data.checkpointId)?.find((r: any) => r.key === data.newRowKey)?.start" @update:value="(val: Date) => setNewRowStart(data.checkpointId, data.newRowKey, val)" :offset="year.dateStart" />
          </template>
          <!-- Existing row editing -->
          <template v-else-if="data.isEditingExisting">
            <DayTimePicker :value="effectiveStart(data)" @update:value="(val: Date) => setExistingStart(data.id, val)" :offset="year.dateStart" />
          </template>
          <!-- Deleted -->
          <span v-else-if="data.isDeleted && data.start" class="line-through text-surface-400">{{ dddhhmm(data.start) }}</span>
          <!-- New confirmed (read-only) -->
          <span v-else-if="data.isNew && !data.isEditing && data.start">{{ dddhhmm(data.start) }}</span>
          <!-- Normal display -->
          <span v-else-if="!data.isEmpty && data.start">{{ dddhhmm(data.start) }}</span>
        </template>
      </Column>

      <Column field="end" header="Slut" style="min-width: 200px">
        <template #body="{ data }">
          <!-- New row editing -->
          <template v-if="data.isNew && data.isEditing">
            <DayTimePicker :value="newRows.get(data.checkpointId)?.find((r: any) => r.key === data.newRowKey)?.end" @update:value="(val: Date) => setNewRowEnd(data.checkpointId, data.newRowKey, val)" :offset="year.dateStart" />
          </template>
          <!-- Existing row editing -->
          <template v-else-if="data.isEditingExisting">
            <DayTimePicker :value="effectiveEnd(data)" @update:value="(val: Date) => setExistingEnd(data.id, val)" :offset="year.dateStart" />
          </template>
          <!-- Deleted -->
          <span v-else-if="data.isDeleted && data.end" class="line-through text-surface-400">{{ hhmm(data.end) }}</span>
          <!-- New confirmed (read-only) -->
          <span v-else-if="data.isNew && !data.isEditing && data.end">{{ hhmm(data.end) }}</span>
          <!-- Normal display -->
          <span v-else-if="!data.isEmpty && data.end">{{ hhmm(data.end) }}</span>
        </template>
      </Column>

      <Column header="" style="width: 120px">
        <template #body="{ data }">
          <!-- New row in editing mode: accept / cancel -->
          <div class="flex gap-1 justify-end" v-if="data.isNew && data.isEditing">
            <Button icon="pi pi-check" severity="success" text rounded size="small" @click="acceptNewRow(data.checkpointId, data.newRowKey)" v-tooltip.top="'Bekræft'" />
            <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="cancelNewRow(data.checkpointId, data.newRowKey)" v-tooltip.top="'Annuller'" />
          </div>
          <!-- New row confirmed (not editing): edit again / remove -->
          <div class="flex gap-1 justify-end" v-else-if="data.isNew && !data.isEditing">
            <Button icon="pi pi-pencil" severity="secondary" text rounded size="small" @click="editNewRow(data.checkpointId, data.newRowKey)" v-tooltip.top="'Ret'" />
            <Button icon="pi pi-times" severity="danger" text rounded size="small" @click="cancelNewRow(data.checkpointId, data.newRowKey)" v-tooltip.top="'Fjern'" />
          </div>
          <!-- Existing row in edit mode: accept / cancel -->
          <div class="flex gap-1 justify-end" v-else-if="data.isEditingExisting">
            <Button icon="pi pi-check" severity="success" text rounded size="small" @click="acceptEditExisting(data)" v-tooltip.top="'Bekræft'" />
            <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="cancelEditExisting(data)" v-tooltip.top="'Annuller'" />
          </div>
          <!-- Existing row marked for deletion: undo -->
          <div class="flex justify-end" v-else-if="data.isDeleted">
            <Button icon="pi pi-undo" severity="secondary" text rounded size="small" @click="unmarkForDeletion(data.id)" v-tooltip.top="'Fortryd'" />
          </div>
          <!-- Existing row: edit times / mark for deletion -->
          <div class="flex gap-1 justify-end" v-else-if="!data.isEmpty">
            <Button icon="pi pi-pencil" severity="secondary" text rounded size="small" @click="startEditExisting(data)" v-tooltip.top="'Ret tider'" />
            <Button icon="pi pi-trash" severity="danger" text rounded size="small" @click="markForDeletion(data.id)" v-tooltip.top="'Fjern'" />
          </div>
        </template>
      </Column>

      <template #groupheader="{ data }">
        <div class="flex items-center justify-between w-full">
          <div class="flex items-center gap-2 font-bold">
            <i class="pi pi-map-marker"></i>
            <span>{{ data.checkpointName }}</span>
          </div>
          <Button icon="pi pi-user-plus" label="Tilføj person" severity="secondary" text size="small" @click="addPersonRow(data.checkpointId)" />
        </div>
      </template>
    </DataTable>

    <div class="flex justify-end gap-2 pt-4">
      <Button icon="pi pi-times" label="Luk" severity="secondary" size="small" @click="cancel" />
      <Button icon="pi pi-save" label="Gem" size="small" :disabled="!hasChanges || saving" :loading="saving" @click="saveAll" />
    </div>
  </div>
</template>
