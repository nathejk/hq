<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'

// ----- Types -----------------------------------------------------------------

type Section = {
  slug: string
  yearSlug: string
  parentSlug?: string
  label: string
}

type CrewMember = {
  userId: string
  yearSlug: string
  name: string
  phone: string
  email: string
  sectionSlug: string
}

type OrganisationResponse = {
  year: string
  sections: Section[]
  crewMembers: CrewMember[]
  availableYearsForCopy: string[]
}

type TreeNode = {
  key: string
  label: string
  icon?: string
  data: {
    type: 'section' | 'crewmember'
    slug?: string
    userId?: string
    parentSlug?: string
    memberCount?: number
  }
  children?: TreeNode[]
  droppable?: boolean
  draggable?: boolean
  selectable?: boolean
}

// ----- State -----------------------------------------------------------------

const toast = useToast()

const year = ref('')
const sections = ref<Section[]>([])
const crewMembers = ref<CrewMember[]>([])
const availableYearsForCopy = ref<string[]>([])
const selectedCopyYear = ref<string | null>(null)

const loading = ref(false)
const busy = ref(false)

// -- New section dialog
const addDialogOpen = ref(false)
const newSectionLabel = ref('')
const newSectionSlug = ref('')
const newSectionParent = ref<string>('')

// -- New crew member dialog
const newCrewDialogOpen = ref(false)
const newCrewName = ref('')
const newCrewPhone = ref('')
const newCrewEmail = ref('')

// -- Edit section dialog
const editDialogOpen = ref(false)
const editingSlug = ref('')
const editingLabel = ref('')

// ----- Derived tree ----------------------------------------------------------

const unassignedCrew = computed(() => crewMembers.value.filter((m) => !m.sectionSlug))

// `tree` must be a ref (not a computed) because PrimeVue Tree mutates the
// bound array on drag-drop. We rebuild it explicitly from `sections` and
// `crewMembers` after every load(), and after any local mutation.
const tree = ref<TreeNode[]>([])

// PrimeVue Tree's internal expansion state resets whenever the bound array
// is replaced. Hoist it into our own ref via `v-model:expandedKeys` so the
// tree stays expanded across load()/rebuildTree() cycles.
const expandedKeys = ref<Record<string, boolean>>({})

// Section keys we've ever rendered. PrimeVue *deletes* a key from
// expandedKeys when the user collapses a section, so "absence" alone can't
// distinguish "never seen" from "explicitly collapsed by the user". This
// set gives us that distinction: on first sight of a section we auto-expand
// it, on subsequent rebuilds we leave it alone.
const seenSectionKeys = new Set<string>()

function rebuildTree() {
  const membersBySection = new Map<string, CrewMember[]>()
  for (const m of crewMembers.value) {
    if (!m.sectionSlug) continue
    const arr = membersBySection.get(m.sectionSlug) ?? []
    arr.push(m)
    membersBySection.set(m.sectionSlug, arr)
  }

  const build = (parentSlug: string): TreeNode[] => {
    const siblings = sections.value.filter((s) => (s.parentSlug ?? '') === parentSlug)
    return siblings.map<TreeNode>((s) => {
      const children = build(s.slug)
      const members = (membersBySection.get(s.slug) ?? []).map<TreeNode>((m) => ({
        key: `crew:${m.userId}`,
        label: m.name || m.email || m.userId,
        icon: 'pi pi-user',
        data: { type: 'crewmember', userId: m.userId, slug: s.slug },
        draggable: true,
        droppable: false,
        selectable: true
      }))
      return {
        key: `section:${s.slug}`,
        label: s.label,
        icon: 'pi pi-folder',
        data: {
          type: 'section',
          slug: s.slug,
          parentSlug,
          memberCount: members.length
        },
        children: [...children, ...members],
        // Sections are draggable for sibling reordering. Reparent attempts
        // (drop under a different parent) are detected in onNodeDrop and
        // rejected because shared-go has no SectionMoved event.
        draggable: true,
        droppable: true,
        selectable: true
      }
    })
  }
  tree.value = build('')

  // Auto-expand any section keys we've never rendered before, while
  // preserving the user's explicit collapses. Using `seenSectionKeys`
  // (rather than the presence-in-expandedKeys check) is deliberate:
  // PrimeVue *deletes* a collapsed key from expandedKeys, so we can't
  // tell "never seen" from "collapsed" without our own set.
  const walk = (nodes: TreeNode[]) => {
    for (const n of nodes) {
      if (n.data.type === 'section' && !seenSectionKeys.has(n.key)) {
        expandedKeys.value[n.key] = true
        seenSectionKeys.add(n.key)
      }
      if (n.children) walk(n.children)
    }
  }
  walk(tree.value)
}

// Flat list of sections for parent-picker dropdowns. The root option is
// labelled after the current year (e.g. "Nathejk 2026") rather than a generic
// "root" placeholder.
const sectionOptions = computed(() => [
  { label: year.value ? `Nathejk ${year.value}` : 'Nathejk', value: '' },
  ...sections.value.map((s) => ({
    label: sectionPath(s),
    value: s.slug
  }))
])

function sectionPath(s: Section): string {
  const parts: string[] = [s.label]
  let cur = s
  const guard = new Set<string>()
  while (cur.parentSlug && !guard.has(cur.parentSlug)) {
    guard.add(cur.parentSlug)
    const parent = sections.value.find((x) => x.slug === cur.parentSlug)
    if (!parent) break
    parts.unshift(parent.label)
    cur = parent
  }
  return parts.join(' › ')
}

// ----- API ------------------------------------------------------------------

async function load() {
  loading.value = true
  try {
    const res = await http.get<OrganisationResponse>('/organisation')
    year.value = res.data.year
    sections.value = res.data.sections ?? []
    crewMembers.value = res.data.crewMembers ?? []
    availableYearsForCopy.value = res.data.availableYearsForCopy ?? []
    if (!selectedCopyYear.value && availableYearsForCopy.value.length > 0) {
      selectedCopyYear.value = availableYearsForCopy.value[0]
    }
    rebuildTree()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke hente organisation',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    loading.value = false
  }
}

async function submitNewSection() {
  if (!newSectionLabel.value.trim()) {
    toast.add({ severity: 'warn', summary: 'Navn mangler', life: 2500 })
    return
  }
  busy.value = true
  try {
    await http.post('/section', {
      label: newSectionLabel.value.trim(),
      slug: newSectionSlug.value.trim() || undefined,
      parentSectionSlug: newSectionParent.value || undefined
    })
    toast.add({
      severity: 'success',
      summary: 'Sektion oprettet',
      detail: newSectionLabel.value,
      life: 2500
    })
    addDialogOpen.value = false
    newSectionLabel.value = ''
    newSectionSlug.value = ''
    newSectionParent.value = ''
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Oprettelse fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

// moveSection reparents a section server-side. Cycle detection lives on the
// server; a rejected move surfaces as a toast and the tree resyncs on reload.
async function moveSection(slug: string, newParentSlug: string): Promise<boolean> {
  try {
    await http.put(`/section/${encodeURIComponent(slug)}/parent`, {
      parentSectionSlug: newParentSlug
    })
    return true
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Flytning fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
    return false
  }
}

// sortSiblings posts a new sibling order for a given parent to the API.
// Returns true on success. The caller is responsible for reloading after
// (possibly batched with other calls to avoid a reload storm).
async function sortSiblings(parentSlug: string, orderedSlugs: string[]): Promise<boolean> {
  try {
    await http.put('/sections/sorted', {
      parentSectionSlug: parentSlug,
      sortedSlugs: orderedSlugs
    })
    return true
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Sortering fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
    return false
  }
}

function openEditDialog(slug: string, label: string) {
  editingSlug.value = slug
  editingLabel.value = label
  editDialogOpen.value = true
}

async function submitEditSection() {
  const label = editingLabel.value.trim()
  if (!label) {
    toast.add({ severity: 'warn', summary: 'Navn mangler', life: 2500 })
    return
  }
  busy.value = true
  try {
    await http.patch(`/section/${encodeURIComponent(editingSlug.value)}`, { label })
    toast.add({ severity: 'success', summary: 'Sektion opdateret', detail: label, life: 2500 })
    editDialogOpen.value = false
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Opdatering fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

async function deleteSection(slug: string, label: string) {
  if (!window.confirm(`Slet sektionen "${label}"?`)) return
  busy.value = true
  try {
    await http.delete(`/section/${encodeURIComponent(slug)}`)
    toast.add({ severity: 'success', summary: 'Sektion slettet', detail: label, life: 2500 })
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Sletning fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

async function assignCrew(userId: string, sectionSlug: string) {
  busy.value = true
  try {
    await http.put(`/crewmember/${encodeURIComponent(userId)}/section`, {
      sectionSlug
    })
    // Optimistic local update; reload will reconcile with the read model.
    const m = crewMembers.value.find((x) => x.userId === userId)
    if (m) m.sectionSlug = sectionSlug
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Tildeling fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

async function submitNewCrew() {
  if (!newCrewName.value.trim()) {
    toast.add({ severity: 'warn', summary: 'Navn mangler', life: 2500 })
    return
  }
  busy.value = true
  try {
    await http.post('/crewmember', {
      name: newCrewName.value.trim(),
      phone: newCrewPhone.value.trim(),
      email: newCrewEmail.value.trim()
    })
    toast.add({ severity: 'success', summary: 'Crew tilføjet', life: 2500 })
    newCrewDialogOpen.value = false
    newCrewName.value = ''
    newCrewPhone.value = ''
    newCrewEmail.value = ''
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke tilføje crew',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

async function copyFromYear() {
  if (!selectedCopyYear.value) return
  const source = selectedCopyYear.value
  if (!window.confirm(`Kopier alle sektioner fra ${source} til dette år?`)) return
  busy.value = true
  try {
    const res = await http.post(`/organisation/copy-from/${encodeURIComponent(source)}`)
    toast.add({
      severity: 'success',
      summary: 'Sektioner kopieret',
      detail: `${res.data.copied} sektion(er) fra ${source}`,
      life: 3000
    })
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Kopiering fejlede',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

// ----- Drag & drop -----------------------------------------------------------
// PrimeVue Tree in v4 has no drag-drop support (only TreeTable does). We
// implement it natively via HTML5 DnD attached to each tree row inside the
// #default slot. dataTransfer carries the drag payload so drops originating
// from the sidebar (unassigned crew list) work interchangeably with drops
// originating from within the tree.

type DropPosition = 'before' | 'on' | 'after'

// Reactive drag state, used purely for hover visual feedback (the actual
// drop payload comes from ev.dataTransfer, which survives cross-list drags).
const dragHover = ref<{ targetKey: string | null; position: DropPosition | null }>({
  targetKey: null,
  position: null
})

function onRowDragStart(ev: DragEvent, node: TreeNode) {
  if (!ev.dataTransfer) return
  if (node.data.type === 'section') {
    ev.dataTransfer.setData('text/x-section-slug', node.data.slug ?? '')
  } else if (node.data.type === 'crewmember') {
    ev.dataTransfer.setData('text/x-crewmember-id', node.data.userId ?? '')
  }
  ev.dataTransfer.effectAllowed = 'move'
}

function onRowDragOver(ev: DragEvent, node: TreeNode) {
  const dt = ev.dataTransfer
  if (!dt) return
  const isSectionDrag = dt.types.includes('text/x-section-slug')
  const isCrewDrag = dt.types.includes('text/x-crewmember-id')
  if (!isSectionDrag && !isCrewDrag) return

  // Crew can only drop ON sections (or on crew rows, treated as their section).
  // Sections can only drop ON/BEFORE/AFTER other sections.
  if (isSectionDrag && node.data.type !== 'section') return

  ev.preventDefault()

  let position: DropPosition = 'on'
  if (isSectionDrag && node.data.type === 'section') {
    const rect = (ev.currentTarget as HTMLElement).getBoundingClientRect()
    const y = ev.clientY - rect.top
    if (y < rect.height * 0.25) position = 'before'
    else if (y > rect.height * 0.75) position = 'after'
  }
  dragHover.value.targetKey = node.key
  dragHover.value.position = position
}

function onRowDragLeave(node: TreeNode) {
  if (dragHover.value.targetKey === node.key) {
    dragHover.value.targetKey = null
    dragHover.value.position = null
  }
}

function onRowDragEnd() {
  dragHover.value.targetKey = null
  dragHover.value.position = null
}

async function onRowDrop(ev: DragEvent, node: TreeNode) {
  ev.preventDefault()
  ev.stopPropagation()

  const dt = ev.dataTransfer
  const crewId = dt?.getData('text/x-crewmember-id') ?? ''
  const draggedSectionSlug = dt?.getData('text/x-section-slug') ?? ''
  const position = dragHover.value.position ?? 'on'
  onRowDragEnd()

  // Crew drop -> assign to whatever section this row is (or belongs to).
  if (crewId) {
    const targetSection = node.data.type === 'section' ? node.data.slug : /* crew row */ node.data.slug
    if (!targetSection) return
    await assignCrew(crewId, targetSection)
    return
  }

  if (!draggedSectionSlug) return
  if (node.data.type !== 'section') return
  if (draggedSectionSlug === node.data.slug) return // self-drop

  busy.value = true
  try {
    if (position === 'on') {
      // Dropped ONTO section -> become its child (at end).
      const moved = await moveSection(draggedSectionSlug, node.data.slug!)
      if (!moved) {
        await load()
        return
      }
    } else {
      // Dropped BEFORE / AFTER section -> sibling of that section.
      const newParent = node.data.parentSlug ?? ''
      const src = sections.value.find((s) => s.slug === draggedSectionSlug)

      // Build target sibling order excluding the dragged section, then
      // splice it in at the requested position.
      const siblingSlugs = sections.value.filter((s) => (s.parentSlug ?? '') === newParent && s.slug !== draggedSectionSlug).map((s) => s.slug)
      const targetIdx = siblingSlugs.indexOf(node.data.slug!)
      if (targetIdx < 0) return
      const insertAt = position === 'before' ? targetIdx : targetIdx + 1
      const newOrder = [...siblingSlugs.slice(0, insertAt), draggedSectionSlug, ...siblingSlugs.slice(insertAt)]

      // Reparent first if needed.
      if (src && (src.parentSlug ?? '') !== newParent) {
        const moved = await moveSection(draggedSectionSlug, newParent)
        if (!moved) {
          await load()
          return
        }
      }
      await sortSiblings(newParent, newOrder)
    }
    await load()
  } finally {
    busy.value = false
  }
}

function rowDropClasses(node: TreeNode): Record<string, boolean> {
  const active = dragHover.value.targetKey === node.key
  return {
    'drop-on': active && dragHover.value.position === 'on',
    'drop-before': active && dragHover.value.position === 'before',
    'drop-after': active && dragHover.value.position === 'after'
  }
}

// Drop onto the "unassigned" list = unassignment.
function onDropUnassigned(ev: DragEvent) {
  ev.preventDefault()
  const userId = ev.dataTransfer?.getData('text/x-crewmember-id')
  if (!userId) return
  assignCrew(userId, '')
}
function onDragStartCrew(ev: DragEvent, userId: string) {
  ev.dataTransfer?.setData('text/x-crewmember-id', userId)
  ev.dataTransfer!.effectAllowed = 'move'
}
function allowDrop(ev: DragEvent) {
  if (ev.dataTransfer?.types.includes('text/x-crewmember-id')) {
    ev.preventDefault()
  }
}

// Quick-assign via dropdown for accessibility / when drag-and-drop isn't
// convenient.
function quickAssignFromDropdown(userId: string, slug: string) {
  assignCrew(userId, slug)
}

onMounted(load)
</script>

<template>
  <div class="py-2">
    <div class="flex justify-between items-center pb-4">
      <h1 class="font-nathejk text-2xl">Organisation</h1>
      <div class="flex gap-2">
        <Button icon="pi pi-user-plus" label="Nyt crew-medlem" size="small" severity="secondary" @click="newCrewDialogOpen = true" />
        <Button icon="pi pi-plus" label="Ny sektion" size="small" :disabled="sections.length === 0 && availableYearsForCopy.length > 0" @click="addDialogOpen = true" />
      </div>
    </div>

    <!-- Empty state with copy-from-year option -->
    <div v-if="!loading && sections.length === 0" class="border border-dashed border-gray-300 rounded p-6 mb-4 text-center">
      <p class="mb-3">Der er endnu ingen sektioner for dette år.</p>
      <div v-if="availableYearsForCopy.length > 0" class="flex justify-center items-center gap-2">
        <Select v-model="selectedCopyYear" :options="availableYearsForCopy" placeholder="Vælg år" class="w-40" />
        <Button icon="pi pi-copy" label="Kopier sektioner" size="small" :disabled="!selectedCopyYear || busy" @click="copyFromYear" />
        <Button icon="pi pi-plus" label="Start fra bunden" size="small" severity="secondary" @click="addDialogOpen = true" />
      </div>
      <div v-else>
        <Button icon="pi pi-plus" label="Opret første sektion" size="small" @click="addDialogOpen = true" />
      </div>
    </div>

    <div v-if="loading" class="p-4 text-center text-gray-500">Indlæser…</div>

    <div v-else-if="sections.length > 0" class="grid gap-4 md:grid-cols-3">
      <div class="md:col-span-2">
        <Tree :value="tree" v-model:expandedKeys="expandedKeys" selectionMode="single" class="border rounded">
          <template #default="{ node }">
            <div class="node-row" :class="rowDropClasses(node)" :draggable="true" @dragstart.stop="onRowDragStart($event, node)" @dragover.stop="onRowDragOver($event, node)" @dragleave.stop="onRowDragLeave(node)" @drop.stop="onRowDrop($event, node)" @dragend.stop="onRowDragEnd">
              <span class="flex-1">{{ node.label }}</span>
              <template v-if="node.data.type === 'section'">
                <Badge v-if="node.data.memberCount" :value="node.data.memberCount" severity="secondary" />
                <Button class="row-action" icon="pi pi-pencil" size="small" severity="secondary" text rounded :disabled="busy" @click.stop="openEditDialog(node.data.slug, node.label)" />
                <Button class="row-action" icon="pi pi-trash" size="small" severity="danger" text rounded :disabled="busy" @click.stop="deleteSection(node.data.slug, node.label)" />
              </template>
            </div>
          </template>
        </Tree>
      </div>

      <aside>
        <h2 class="font-semibold pb-2">Ikke tildelt ({{ unassignedCrew.length }})</h2>
        <ul class="border rounded p-2 min-h-32 space-y-1 bg-gray-50" @dragover="allowDrop" @drop="onDropUnassigned">
          <li v-if="unassignedCrew.length === 0" class="text-sm text-gray-500 italic">Ingen frie crew-medlemmer</li>
          <li v-for="m in unassignedCrew" :key="m.userId" :draggable="true" class="p-2 bg-white border rounded flex items-center gap-2 cursor-move" @dragstart="onDragStartCrew($event, m.userId)">
            <i class="pi pi-user text-gray-500" />
            <span class="flex-1 truncate">{{ m.name || m.email || m.userId }}</span>
            <Select :modelValue="''" :options="sectionOptions.filter((o) => o.value !== '')" optionLabel="label" optionValue="value" placeholder="Tildel…" size="small" class="w-32" @update:modelValue="(v: string) => v && quickAssignFromDropdown(m.userId, v)" />
          </li>
        </ul>

        <p class="mt-3 text-xs text-gray-500">Træk en sektion for at ændre rækkefølge eller flytte den til en ny forælder. Træk et crew-navn ind i en sektion for at tildele, eller tilbage hertil for at fjerne.</p>
      </aside>
    </div>

    <!-- Create section dialog -->
    <Dialog v-model:visible="addDialogOpen" modal :style="{ width: '30rem' }" header="Ny sektion">
      <div class="grid gap-3 pt-2">
        <FloatLabel variant="on">
          <InputText id="section-label" v-model="newSectionLabel" class="w-full" autocomplete="off" />
          <label for="section-label">Navn</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="section-slug" v-model="newSectionSlug" class="w-full" autocomplete="off" placeholder="auto" />
          <label for="section-slug">Slug (valgfri)</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <Select id="section-parent" v-model="newSectionParent" :options="sectionOptions" optionLabel="label" optionValue="value" class="w-full" />
          <label for="section-parent">Forældresektion</label>
        </FloatLabel>
      </div>
      <template #footer>
        <Button label="Annuller" text @click="addDialogOpen = false" />
        <Button label="Opret" icon="pi pi-check" :loading="busy" @click="submitNewSection" />
      </template>
    </Dialog>

    <!-- Edit section dialog -->
    <Dialog v-model:visible="editDialogOpen" modal :style="{ width: '30rem' }" header="Rediger sektion">
      <div class="grid gap-3 pt-2">
        <FloatLabel variant="on">
          <InputText id="edit-section-label" v-model="editingLabel" class="w-full" autocomplete="off" @keydown.enter="submitEditSection" />
          <label for="edit-section-label">Navn</label>
        </FloatLabel>
        <p class="text-xs text-gray-500">Slug ({{ editingSlug }}) og placering i træet kan endnu ikke ændres.</p>
      </div>
      <template #footer>
        <Button label="Annuller" text @click="editDialogOpen = false" />
        <Button label="Gem" icon="pi pi-check" :loading="busy" @click="submitEditSection" />
      </template>
    </Dialog>

    <!-- Create crew member dialog -->
    <Dialog v-model:visible="newCrewDialogOpen" modal :style="{ width: '30rem' }" header="Nyt crew-medlem">
      <div class="grid gap-3 pt-2">
        <FloatLabel variant="on">
          <InputText id="crew-name" v-model="newCrewName" class="w-full" autocomplete="off" />
          <label for="crew-name">Navn</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="crew-phone" v-model="newCrewPhone" class="w-full" autocomplete="off" />
          <label for="crew-phone">Telefon</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="crew-email" v-model="newCrewEmail" class="w-full" autocomplete="off" />
          <label for="crew-email">Email</label>
        </FloatLabel>
      </div>
      <template #footer>
        <Button label="Annuller" text @click="newCrewDialogOpen = false" />
        <Button label="Opret" icon="pi pi-check" :loading="busy" @click="submitNewCrew" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.node-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 2px 4px;
  border-radius: 3px;
  position: relative;
  cursor: grab;
}
.node-row:active {
  cursor: grabbing;
}
.node-row.drop-on {
  background: rgba(59, 130, 246, 0.15);
}
.node-row.drop-before::before,
.node-row.drop-after::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  height: 2px;
  background: rgb(59, 130, 246);
  pointer-events: none;
}
.node-row.drop-before::before {
  top: -1px;
}
.node-row.drop-after::after {
  bottom: -1px;
}

/* Row actions (edit / delete) reveal on hover. focus-within keeps them
   visible when a button has keyboard focus. */
.row-action {
  opacity: 0;
  transition: opacity 120ms ease;
}
.node-row:hover .row-action,
.node-row:focus-within .row-action {
  opacity: 1;
}
</style>
