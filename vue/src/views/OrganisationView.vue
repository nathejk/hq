<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useDeferredApply } from '@/composables/useDeferredApply'
import PositionIndicator from '@/components/PositionIndicator.vue'

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
  // The scouting identity, as the signup form collects it. Kept on the type
  // even though the tree shows only the name: the edit dialog is where they are
  // read and written, and leaving them off meant a save from that dialog had to
  // guess what it was preserving.
  medlemnr: string
  group: string
  /** Korps *slug* ("dds"), not a label — the labels come from `corpsOptions`. */
  corps: string
  diet: string
  /**
   * The remaining signup answers, as a JSON *string* (the projection stores the
   * document verbatim). Read-only here: the API hands it back to the event
   * untouched on save.
   */
  additionals: string
}

// A car present in the race area. Vehicles sit in the same tree as crew
// members because they hang off the same sections: a car is brought by a crew
// member (its custodian) and belongs to at most one section.
type Vehicle = {
  vehicleId: string
  yearSlug: string
  licensePlate: string
  custodianUserId: string
  driverUserId: string
  sectionSlug: string
  color: string
  brand: string
  model: string
  seatCount: number
  description: string
}

type OrganisationResponse = {
  year: string
  sections: Section[]
  crewMembers: CrewMember[]
  vehicles: Vehicle[]
  availableYearsForCopy: string[]
  // Which sections may be assigned an SOS case (PRD 001). A list of slugs beside
  // the sections rather than a field on each one: the section entity belongs to
  // shared-go and knows nothing about the nødtelefon.
  sosAssignableSections?: string[]
  // Which sections are dispatch units (PRD 009) — a subsection holding a vehicle,
  // a driver and possibly a co-driver, that a tour may be assigned to. Beside the
  // sections for the same reason: kørsel is not a property of a section.
  dispatchableSections?: string[]
  /**
   * Signup page per crew member, keyed by userId, for those who came through the
   * public form. Absent for anybody an HQ operator typed in, who has no signup
   * page — which is why this is a server-built map and not a URL this view
   * assembles from an id.
   */
  crewSignupUrls?: Record<string, string>
  /** The canonical korps list, slug + label, as the signup form offers it. */
  corpsOptions?: { slug: string; label: string }[]
}

type TreeNode = {
  key: string
  label: string
  icon?: string
  data: {
    type: 'section' | 'crewmember' | 'vehicle'
    slug?: string
    userId?: string
    vehicleId?: string
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
const vehicles = ref<Vehicle[]>([])
const crewSignupUrls = ref<Record<string, string>>({})
const corpsOptions = ref<{ slug: string; label: string }[]>([])
const availableYearsForCopy = ref<string[]>([])
const selectedCopyYear = ref<string | null>(null)

const busy = ref(false)

/**
 * A drag gesture is in progress.
 *
 * Declared here rather than with the drag handlers below because `paused` reads
 * it: a payload applied mid-drag rebuilds the very rows being dragged over.
 */
const dragActive = ref(false)

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

// -- Edit crew member dialog
const editCrewDialogOpen = ref(false)
const editingCrew = ref<CrewMember | null>(null)
const editCrewName = ref('')
const editCrewPhone = ref('')
const editCrewEmail = ref('')
const editCrewMedlemNr = ref('')
const editCrewGroup = ref('')
const editCrewCorps = ref<string | null>(null)
const editCrewDiet = ref('')

// -- Vehicle dialogs. One form serves both create and edit: the fields are the
// same, and `editingVehicle` is what tells them apart at submit time.
const vehicleDialogOpen = ref(false)
const editingVehicle = ref<Vehicle | null>(null)
const vehiclePlate = ref('')
const vehicleCustodian = ref<string>('')
const vehicleColor = ref('')
const vehicleBrand = ref('')
const vehicleModel = ref('')
const vehicleSeatCount = ref<number>(0)
const vehicleDescription = ref('')

// -- Edit section dialog
const editDialogOpen = ref(false)
const editingSlug = ref('')
const editingLabel = ref('')

// ----- Derived tree ----------------------------------------------------------

const unassignedCrew = computed(() => crewMembers.value.filter((m) => !m.sectionSlug))
const unassignedVehicles = computed(() => vehicles.value.filter((v) => !v.sectionSlug))

// A plate alone reads as noise in a tree of names, so the label carries what an
// operator recognises the car by. The plate stays first because that is what is
// visible from outside the car.
function vehicleLabel(v: Vehicle): string {
  const make = [v.brand, v.model].filter(Boolean).join(' ')
  const parts = [v.licensePlate || 'Ukendt nummerplade']
  if (make) parts.push(make)
  if (v.color) parts.push(v.color)
  return parts.join(' · ')
}

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
  const vehiclesBySection = new Map<string, Vehicle[]>()
  for (const v of vehicles.value) {
    if (!v.sectionSlug) continue
    const arr = vehiclesBySection.get(v.sectionSlug) ?? []
    arr.push(v)
    vehiclesBySection.set(v.sectionSlug, arr)
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
      const cars = (vehiclesBySection.get(s.slug) ?? []).map<TreeNode>((v) => ({
        key: `vehicle:${v.vehicleId}`,
        label: vehicleLabel(v),
        icon: 'pi pi-car',
        data: { type: 'vehicle', vehicleId: v.vehicleId, slug: s.slug },
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
        children: [...children, ...members, ...cars],
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

// Sections that may be assigned an SOS case (PRD 001 §6).
//
// Off by default and opted into per section, so the nødtelefon's assignee list
// starts empty rather than offering every section in the organisation.
const sosAssignable = ref<Set<string>>(new Set())

// Takes an optional slug because the tree's node data is loosely typed — cleaner
// than a non-null assertion at four call sites in the template.
const isSosAssignable = (slug?: string) => !!slug && sosAssignable.value.has(slug)

async function toggleSosAssignable(slug?: string, label?: string) {
  if (!slug) return
  const name = label ?? slug
  const next = !sosAssignable.value.has(slug)
  // Optimistic: the toggle is a switch, and a switch that waits for a round trip
  // before moving feels broken. Reverted below if the write fails.
  const snapshot = new Set(sosAssignable.value)
  const updated = new Set(sosAssignable.value)
  if (next) updated.add(slug)
  else updated.delete(slug)
  sosAssignable.value = updated

  try {
    await http.put(`/section/${slug}/sos-assignable`, { assignable: next })
    toast.add({
      severity: 'success',
      summary: next ? `${name} kan tildeles nødråb` : `${name} kan ikke længere tildeles nødråb`,
      life: 2500
    })
  } catch (err: any) {
    sosAssignable.value = snapshot
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke ændre nødråb-tildeling',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  }
}

// Sections that are dispatch units (PRD 009 §6).
//
// Off by default and opted into per section, exactly as the nødråb flag above, so a
// fresh year has no dispatch capacity until somebody says which subsections hold a
// car. The flag is a fact about kørsel, not about the section.
const dispatchable = ref<Set<string>>(new Set())

const isDispatchable = (slug?: string) => !!slug && dispatchable.value.has(slug)

async function toggleDispatchable(slug?: string, label?: string) {
  if (!slug) return
  const name = label ?? slug
  const next = !dispatchable.value.has(slug)
  // Optimistic, as the nødråb toggle is — and here it is also necessary: the command
  // dirty-checks before publishing, so a toggle that changes nothing emits no event
  // and therefore no live signal to confirm it by.
  const snapshot = new Set(dispatchable.value)
  const updated = new Set(dispatchable.value)
  if (next) updated.add(slug)
  else updated.delete(slug)
  dispatchable.value = updated

  try {
    await http.put(`/section/${slug}/dispatchable`, { dispatchable: next })
    toast.add({
      severity: 'success',
      summary: next ? `${name} kan køre ture` : `${name} kan ikke længere køre ture`,
      life: 2500
    })
  } catch (err: any) {
    dispatchable.value = snapshot
    toast.add({
      severity: 'error',
      summary: 'Kunne ikke ændre kørsels-enhed',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  }
}

// The whole screen is one live resource (PRD 004).
//
// One key, not one per collection: sections, crew members and vehicles arrive in
// a single payload because the tree needs all three to render a row, and splitting
// them would let the parts revalidate independently and be drawn half-updated.
//
// The tokens are the event *subjects'* entities, taken from the projections that
// own them:
//
//   section   NATHEJK.*.section.*.added|moved|deleted
//   sections  NATHEJK.*.sections.sorted        — collection-level, no id
//   crewmember NATHEJK.*.crewmember.*.registered|updated|deleted|section.assigned
//   crew      NATHEJK.*.crew.*.signedup       — a signup mints a crew member, and
//             is also what gives it a signup URL in this payload
//   vehicle   NATHEJK.*.vehicle.*.…
//   sos:section  NATHEJK:*.sos.section.*.assignable — the nødråb toggle. Instance
//             rather than the bare `sos` type on purpose: every comment on every
//             case is an `sos` signal, and none of them changes this screen.
//
// All type-level (except that last one) because this is a list: a section or crew
// member created by somebody else has an id this client has never seen, so an
// instance-keyed dependency could not make it appear.
const {
  data: organisation,
  pending,
  error: organisationError,
  refresh
} = useLiveResource(
  'organisation',
  async () => (await http.get<OrganisationResponse>('/organisation')).data,
  { dependsOn: ['section', 'sections', 'crewmember', 'crew', 'vehicle', 'sos:section'] }
)

watch(organisationError, (err: any) => {
  if (!err) return
  toast.add({
    severity: 'error',
    summary: 'Kunne ikke hente organisation',
    detail: err?.response?.data?.error ?? String(err),
    life: 5000
  })
})

/**
 * Copy a payload into the refs the tree and the side panel are built from.
 *
 * The view keeps its own refs rather than rendering the cached payload directly
 * because both the tree and the assignment handlers mutate them: `rebuildTree`
 * needs a plain array PrimeVue may reorder, and the optimistic assignment writes
 * `sectionSlug` before the server has confirmed it. Applying is therefore an
 * explicit step — which is also what makes it deferrable.
 */
function applyPayload(res: OrganisationResponse) {
  year.value = res.year
  sections.value = res.sections ?? []
  crewMembers.value = res.crewMembers ?? []
  vehicles.value = res.vehicles ?? []
  crewSignupUrls.value = res.crewSignupUrls ?? {}
  corpsOptions.value = res.corpsOptions ?? []
  availableYearsForCopy.value = res.availableYearsForCopy ?? []
  sosAssignable.value = new Set(res.sosAssignableSections ?? [])
  dispatchable.value = new Set(res.dispatchableSections ?? [])
  if (!selectedCopyYear.value && availableYearsForCopy.value.length > 0) {
    selectedCopyYear.value = availableYearsForCopy.value[0]
  }
  rebuildTree()
}

/**
 * True while the screen must not be redrawn underneath the operator.
 *
 * Three kinds of unsaved state live on this page, and a payload applied during
 * any of them destroys work or aim:
 *
 *  - **an open dialog** holds a half-typed form, and `editingCrew` /
 *    `editingVehicle` point into the very arrays a payload replaces;
 *  - **a drag in progress** would have its rows renumbered mid-gesture, so the
 *    drop would land somewhere the operator was not pointing;
 *  - **a write in flight** — a reparent followed by a sort is two requests, and
 *    the signal from the first would rebuild the tree while the second is still
 *    on its way, showing an order that never existed.
 */
const paused = computed(
  () =>
    busy.value ||
    dragActive.value ||
    addDialogOpen.value ||
    editDialogOpen.value ||
    newCrewDialogOpen.value ||
    editCrewDialogOpen.value ||
    vehicleDialogOpen.value
)

/**
 * A payload arrived while paused and still needs applying.
 *
 * Reported on screen: an operator who has been told the page is live deserves to
 * know when it is deliberately not.
 */
const { updatesWaiting } = useDeferredApply(organisation, paused, applyPayload)

/**
 * Force a revalidation.
 *
 * Kept under the old name, and still awaited by every write, because a write's
 * own signal is not a substitute: a command that changes nothing publishes no
 * event (the dirty-check in the commands), so a no-op save would otherwise leave
 * the operator looking at whatever they had typed with nothing to correct it.
 */
async function load() {
  await refresh()
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

// ----- Crew member editing ---------------------------------------------------

/** The signup page of a crew member who came through the public form, if any. */
const signupUrl = (userId?: string) => (userId ? crewSignupUrls.value[userId] : undefined)

/**
 * The korps list, plus the member's own value when the canonical list has no
 * such slug.
 *
 * Without this, a row carrying a korps from an earlier signup form would have no
 * matching option: the Select would fall back to its placeholder, the operator
 * would see an empty field, and saving anything else on the dialog would write
 * that emptiness back. Silent data loss on a form the operator never touched.
 */
const corpsSelectOptions = computed(() => {
  const current = editCrewCorps.value
  if (!current || corpsOptions.value.some((c) => c.slug === current)) return corpsOptions.value
  return [...corpsOptions.value, { slug: current, label: current }]
})

/**
 * The signup answers that have no column of their own, as label/value pairs.
 *
 * Shown read-only beside Kost because of a trap in the data: the signup form
 * files its dietary answer in `additionals` (`{"diet":"Nej"}`), while the `diet`
 * column is the free-text note HQ maintains. Every crew member who signed up
 * therefore has an empty `diet` and an answer nobody could see — so an operator
 * looking at a blank Kost field could not tell "no dietary needs" from "never
 * asked".
 */
const signupAnswers = computed<{ label: string; value: string }[]>(() => {
  const raw = editingCrew.value?.additionals
  if (!raw) return []
  let parsed: Record<string, unknown>
  try {
    parsed = JSON.parse(raw)
  } catch {
    // A malformed document is the server's problem, not something to throw a
    // render over: the rest of the dialog is still worth showing.
    return []
  }
  const labels: Record<string, string> = { diet: 'Kost oplyst', tshirtSize: 'T-shirt' }
  return Object.entries(parsed)
    .filter(([, v]) => v !== '' && v !== null && v !== undefined)
    .map(([k, v]) => ({ label: labels[k] ?? k, value: String(v) }))
})

function openEditCrew(userId?: string) {
  const m = crewMembers.value.find((x) => x.userId === userId)
  if (!m) return
  editingCrew.value = m
  editCrewName.value = m.name
  editCrewPhone.value = m.phone
  editCrewEmail.value = m.email
  editCrewMedlemNr.value = m.medlemnr ?? ''
  editCrewGroup.value = m.group ?? ''
  // null rather than '' so the Select shows its placeholder for an unset korps
  // instead of an empty selected option.
  editCrewCorps.value = m.corps || null
  editCrewDiet.value = m.diet ?? ''
  editCrewDialogOpen.value = true
}

async function submitEditCrew() {
  const m = editingCrew.value
  if (!m) return
  if (!editCrewName.value.trim()) {
    toast.add({ severity: 'warn', summary: 'Navn mangler', life: 2500 })
    return
  }
  busy.value = true
  try {
    await http.patch(`/crewmember/${encodeURIComponent(m.userId)}`, {
      name: editCrewName.value.trim(),
      phone: editCrewPhone.value.trim(),
      email: editCrewEmail.value.trim(),
      // Sent on every save, including when blank: the endpoint treats an omitted
      // field as "keep" and a present one as "replace", so an empty string is how
      // a value gets cleared.
      medlemnr: editCrewMedlemNr.value.trim(),
      group: editCrewGroup.value.trim(),
      corps: editCrewCorps.value ?? '',
      diet: editCrewDiet.value.trim()
    })
    toast.add({ severity: 'success', summary: 'Crew-medlem opdateret', life: 2500 })
    editCrewDialogOpen.value = false
    editingCrew.value = null
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

async function deleteCrew() {
  const m = editingCrew.value
  if (!m) return
  if (!window.confirm(`Slet crew-medlemmet "${m.name || m.userId}"?`)) return
  busy.value = true
  try {
    await http.delete(`/crewmember/${encodeURIComponent(m.userId)}`)
    toast.add({ severity: 'success', summary: 'Crew-medlem slettet', life: 2500 })
    editCrewDialogOpen.value = false
    editingCrew.value = null
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

// ----- Vehicles --------------------------------------------------------------

// Crew members offered as custodian. A vehicle must have one — somebody has to
// answer for the car — so there is no empty option.
const custodianOptions = computed(() =>
  crewMembers.value.map((m) => ({
    label: m.name || m.email || m.userId,
    value: m.userId
  }))
)

function resetVehicleForm() {
  vehiclePlate.value = ''
  vehicleCustodian.value = ''
  vehicleColor.value = ''
  vehicleBrand.value = ''
  vehicleModel.value = ''
  vehicleSeatCount.value = 0
  vehicleDescription.value = ''
}

function openNewVehicle() {
  editingVehicle.value = null
  resetVehicleForm()
  vehicleDialogOpen.value = true
}

function openEditVehicle(vehicleId?: string) {
  const v = vehicles.value.find((x) => x.vehicleId === vehicleId)
  if (!v) return
  editingVehicle.value = v
  vehiclePlate.value = v.licensePlate
  vehicleCustodian.value = v.custodianUserId
  vehicleColor.value = v.color
  vehicleBrand.value = v.brand
  vehicleModel.value = v.model
  vehicleSeatCount.value = v.seatCount ?? 0
  vehicleDescription.value = v.description
  vehicleDialogOpen.value = true
}

// One submit for both create and edit. The edit path PATCHes the whole form:
// the API compares it against what is recorded and publishes only the fields
// that actually change, so re-saving an untouched form writes nothing.
async function submitVehicle() {
  if (!vehiclePlate.value.trim()) {
    toast.add({ severity: 'warn', summary: 'Nummerplade mangler', life: 2500 })
    return
  }
  if (!vehicleCustodian.value) {
    toast.add({ severity: 'warn', summary: 'Ansvarlig mangler', life: 2500 })
    return
  }
  const payload = {
    licensePlate: vehiclePlate.value.trim(),
    custodianUserId: vehicleCustodian.value,
    color: vehicleColor.value.trim(),
    brand: vehicleBrand.value.trim(),
    model: vehicleModel.value.trim(),
    seatCount: vehicleSeatCount.value ?? 0,
    description: vehicleDescription.value.trim()
  }
  const existing = editingVehicle.value
  busy.value = true
  try {
    if (existing) {
      await http.patch(`/vehicle/${encodeURIComponent(existing.vehicleId)}`, payload)
      toast.add({ severity: 'success', summary: 'Køretøj opdateret', life: 2500 })
    } else {
      await http.post('/vehicle', payload)
      toast.add({ severity: 'success', summary: 'Køretøj oprettet', life: 2500 })
    }
    vehicleDialogOpen.value = false
    editingVehicle.value = null
    resetVehicleForm()
    await load()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: existing ? 'Opdatering fejlede' : 'Kunne ikke oprette køretøj',
      detail: err?.response?.data?.error ?? String(err),
      life: 5000
    })
  } finally {
    busy.value = false
  }
}

async function deleteVehicle() {
  const v = editingVehicle.value
  if (!v) return
  if (!window.confirm(`Slet køretøjet "${vehicleLabel(v)}"?`)) return
  busy.value = true
  try {
    await http.delete(`/vehicle/${encodeURIComponent(v.vehicleId)}`)
    toast.add({ severity: 'success', summary: 'Køretøj slettet', life: 2500 })
    vehicleDialogOpen.value = false
    editingVehicle.value = null
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

async function assignVehicle(vehicleId: string, sectionSlug: string) {
  busy.value = true
  try {
    await http.put(`/vehicle/${encodeURIComponent(vehicleId)}/section`, { sectionSlug })
    // Optimistic local update; reload will reconcile with the read model.
    const v = vehicles.value.find((x) => x.vehicleId === vehicleId)
    if (v) v.sectionSlug = sectionSlug
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

// A click on a leaf row opens its editor. Sections are excluded: clicking one
// expands it, and they have their own pencil button in the row.
//
// The parameter is typed structurally rather than as our TreeNode because the
// slot hands us PrimeVue's own (looser) node type.
function onRowClick(node: { data?: any }) {
  if (node.data?.type === 'crewmember') openEditCrew(node.data.userId)
  else if (node.data?.type === 'vehicle') openEditVehicle(node.data.vehicleId)
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
  dragActive.value = true
  if (node.data.type === 'section') {
    ev.dataTransfer.setData('text/x-section-slug', node.data.slug ?? '')
  } else if (node.data.type === 'crewmember') {
    ev.dataTransfer.setData('text/x-crewmember-id', node.data.userId ?? '')
  } else if (node.data.type === 'vehicle') {
    ev.dataTransfer.setData('text/x-vehicle-id', node.data.vehicleId ?? '')
  }
  ev.dataTransfer.effectAllowed = 'move'
}

function onRowDragOver(ev: DragEvent, node: TreeNode) {
  const dt = ev.dataTransfer
  if (!dt) return
  const isSectionDrag = dt.types.includes('text/x-section-slug')
  const isLeafDrag = dt.types.includes('text/x-crewmember-id') || dt.types.includes('text/x-vehicle-id')
  if (!isSectionDrag && !isLeafDrag) return

  // Crew and vehicles can only drop ON sections (or on a leaf row, treated as
  // its section). Sections can only drop ON/BEFORE/AFTER other sections.
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
  dragActive.value = false
  dragHover.value.targetKey = null
  dragHover.value.position = null
}

async function onRowDrop(ev: DragEvent, node: TreeNode) {
  ev.preventDefault()
  ev.stopPropagation()

  const dt = ev.dataTransfer
  const crewId = dt?.getData('text/x-crewmember-id') ?? ''
  const vehicleId = dt?.getData('text/x-vehicle-id') ?? ''
  const draggedSectionSlug = dt?.getData('text/x-section-slug') ?? ''
  const position = dragHover.value.position ?? 'on'
  onRowDragEnd()

  // Crew drop -> assign to whatever section this row is (or belongs to).
  if (crewId) {
    const targetSection = node.data.type === 'section' ? node.data.slug : /* leaf row */ node.data.slug
    if (!targetSection) return
    await assignCrew(crewId, targetSection)
    return
  }

  // Vehicle drop -> same rule as crew: the row's section takes the car.
  if (vehicleId) {
    const targetSection = node.data.slug
    if (!targetSection) return
    await assignVehicle(vehicleId, targetSection)
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
  dragActive.value = false
  const userId = ev.dataTransfer?.getData('text/x-crewmember-id')
  if (userId) {
    assignCrew(userId, '')
    return
  }
  const vehicleId = ev.dataTransfer?.getData('text/x-vehicle-id')
  if (vehicleId) assignVehicle(vehicleId, '')
}
function onDragStartCrew(ev: DragEvent, userId: string) {
  dragActive.value = true
  ev.dataTransfer?.setData('text/x-crewmember-id', userId)
  ev.dataTransfer!.effectAllowed = 'move'
}
function onDragStartVehicle(ev: DragEvent, vehicleId: string) {
  dragActive.value = true
  ev.dataTransfer?.setData('text/x-vehicle-id', vehicleId)
  ev.dataTransfer!.effectAllowed = 'move'
}
function allowDrop(ev: DragEvent) {
  const types = ev.dataTransfer?.types
  if (types?.includes('text/x-crewmember-id') || types?.includes('text/x-vehicle-id')) {
    ev.preventDefault()
  }
}

// Quick-assign via dropdown for accessibility / when drag-and-drop isn't
// convenient.
function quickAssignFromDropdown(userId: string, slug: string) {
  assignCrew(userId, slug)
}

function quickAssignVehicleFromDropdown(vehicleId: string, slug: string) {
  assignVehicle(vehicleId, slug)
}
</script>

<template>
  <div class="py-2">
    <div class="flex justify-between items-center pb-4">
      <h1 class="font-nathejk text-2xl">Organisation</h1>
      <div class="flex items-center gap-2">
        <!--
          Said out loud, because otherwise the screen is quietly lying: this page
          updates itself, so an operator who has learned that has to be told the one
          time it is not. Shown only once something is actually waiting — "paused"
          with nothing pending would be noise on every dialog open — and not while a
          write is in flight, which pauses too but already shows its own spinner and
          resolves in a moment.
        -->
        <span v-if="updatesWaiting && !busy" class="text-sm text-gray-500 mr-2" v-tooltip.bottom="'Ændringer fra andre anvendes, når du er færdig'">
          <i class="pi pi-pause-circle" /> Opdateringer sat på pause
        </span>
        <Button icon="pi pi-user-plus" label="Nyt crew-medlem" size="small" severity="secondary" @click="newCrewDialogOpen = true" />
        <Button icon="pi pi-car" label="Nyt køretøj" size="small" severity="secondary" :disabled="crewMembers.length === 0" v-tooltip.bottom="crewMembers.length === 0 ? 'Opret først et crew-medlem, der kan være ansvarlig' : undefined" @click="openNewVehicle" />
        <Button icon="pi pi-plus" label="Ny sektion" size="small" :disabled="sections.length === 0 && availableYearsForCopy.length > 0" @click="addDialogOpen = true" />
      </div>
    </div>

    <!-- Empty state with copy-from-year option -->
    <div v-if="!pending && sections.length === 0" class="border border-dashed border-gray-300 rounded p-6 mb-4 text-center">
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

    <!--
      Only when there is nothing cached to show: `pending` stays false during a
      background revalidation, so a live update must not replace the tree with a
      loading line, and a revisit must not flash one.
    -->
    <div v-if="pending" class="p-4 text-center text-gray-500">Indlæser…</div>

    <div v-else-if="sections.length > 0" class="grid gap-4 md:grid-cols-3">
      <div class="md:col-span-2">
        <Tree :value="tree" v-model:expandedKeys="expandedKeys" selectionMode="single" class="border rounded">
          <template #default="{ node }">
            <div class="node-row" :class="[rowDropClasses(node), node.data.type === 'section' ? '' : 'cursor-pointer']" :draggable="true" @click="onRowClick(node)" @dragstart.stop="onRowDragStart($event, node)" @dragover.stop="onRowDragOver($event, node)" @dragleave.stop="onRowDragLeave(node)" @drop.stop="onRowDrop($event, node)" @dragend.stop="onRowDragEnd">
              <span class="flex-1">{{ node.label }}</span>
              <template v-if="node.data.type === 'section'">
                <Badge v-if="node.data.memberCount" :value="node.data.memberCount" severity="secondary" />
                <!--
                  Nødråb-tildeling (PRD 001). Shown as a persistent icon when enabled
                  and only on hover when not: which sections take nødråb is worth
                  seeing at a glance, but an off state on every row would be clutter.
                -->
                <i v-if="isSosAssignable(node.data.slug)" class="pi pi-phone text-primary-500"
                   v-tooltip.top="'Kan tildeles nødråb'" />
                <Button class="row-action" :icon="isSosAssignable(node.data.slug) ? 'pi pi-phone-slash' : 'pi pi-phone'"
                        size="small" severity="secondary" text rounded :disabled="busy"
                        v-tooltip.top="isSosAssignable(node.data.slug) ? 'Kan ikke tildeles nødråb' : 'Kan tildeles nødråb'"
                        @click.stop="toggleSosAssignable(node.data.slug, node.label)" />
                <!--
                  Kørsels-enhed (PRD 009). Same treatment as the nødråb flag: a
                  persistent icon when the subsection is a dispatch unit, and a hover
                  action when it is not.
                -->
                <i v-if="isDispatchable(node.data.slug)" class="pi pi-truck text-primary-500"
                   v-tooltip.top="'Kørsels-enhed'" />
                <Button class="row-action" icon="pi pi-truck"
                        size="small" :severity="isDispatchable(node.data.slug) ? 'danger' : 'secondary'" text rounded :disabled="busy"
                        v-tooltip.top="isDispatchable(node.data.slug) ? 'Er ikke kørsels-enhed' : 'Er kørsels-enhed'"
                        @click.stop="toggleDispatchable(node.data.slug, node.label)" />
                <Button class="row-action" icon="pi pi-pencil" size="small" severity="secondary" text rounded :disabled="busy" @click.stop="openEditDialog(node.data.slug, node.label)" />
                <Button class="row-action" icon="pi pi-trash" size="small" severity="danger" text rounded :disabled="busy" @click.stop="deleteSection(node.data.slug, node.label)" />
              </template>
            </div>
          </template>
        </Tree>
      </div>

      <aside>
        <h2 class="font-semibold pb-2">Ikke tildelt ({{ unassignedCrew.length + unassignedVehicles.length }})</h2>
        <ul class="border rounded p-2 min-h-32 space-y-1 bg-gray-50" @dragover="allowDrop" @drop="onDropUnassigned">
          <li v-if="unassignedCrew.length === 0 && unassignedVehicles.length === 0" class="text-sm text-gray-500 italic">Ingen frie crew-medlemmer eller køretøjer</li>
          <li v-for="m in unassignedCrew" :key="m.userId" :draggable="true" class="p-2 bg-white border rounded flex items-center gap-2 cursor-move" @dragstart="onDragStartCrew($event, m.userId)" @dragend="onRowDragEnd">
            <i class="pi pi-user text-gray-500" />
            <span class="flex-1 truncate cursor-pointer" @click="openEditCrew(m.userId)">{{ m.name || m.email || m.userId }}</span>
            <PositionIndicator :person-id="m.userId" />
            <Select :modelValue="''" :options="sectionOptions.filter((o) => o.value !== '')" optionLabel="label" optionValue="value" placeholder="Tildel…" size="small" class="w-32" @update:modelValue="(v: string) => v && quickAssignFromDropdown(m.userId, v)" />
          </li>
          <li v-for="v in unassignedVehicles" :key="v.vehicleId" :draggable="true" class="p-2 bg-white border rounded flex items-center gap-2 cursor-move" @dragstart="onDragStartVehicle($event, v.vehicleId)" @dragend="onRowDragEnd">
            <i class="pi pi-car text-gray-500" />
            <span class="flex-1 truncate cursor-pointer" @click="openEditVehicle(v.vehicleId)">{{ vehicleLabel(v) }}</span>
            <Select :modelValue="''" :options="sectionOptions.filter((o) => o.value !== '')" optionLabel="label" optionValue="value" placeholder="Tildel…" size="small" class="w-32" @update:modelValue="(s: string) => s && quickAssignVehicleFromDropdown(v.vehicleId, s)" />
          </li>
        </ul>

        <p class="mt-3 text-xs text-gray-500">Træk en sektion for at ændre rækkefølge eller flytte den til en ny forælder. Træk et crew-navn eller et køretøj ind i en sektion for at tildele, eller tilbage hertil for at fjerne. Klik på et navn eller en nummerplade for at rette det.</p>
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

    <!-- Edit crew member dialog -->
    <Dialog v-model:visible="editCrewDialogOpen" modal :style="{ width: '34rem' }" header="Rediger crew-medlem">
      <div class="grid gap-3 pt-2">
        <FloatLabel variant="on">
          <InputText id="edit-crew-name" v-model="editCrewName" class="w-full" autocomplete="off" @keydown.enter="submitEditCrew" />
          <label for="edit-crew-name">Navn</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="edit-crew-phone" v-model="editCrewPhone" class="w-full" autocomplete="off" />
          <label for="edit-crew-phone">Telefon</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="edit-crew-email" v-model="editCrewEmail" class="w-full" autocomplete="off" />
          <label for="edit-crew-email">Email</label>
        </FloatLabel>

        <!--
          The scouting identity, below the contact details: reaching somebody is
          what this dialog is opened for in a hurry, and gruppe/korps/medlemsnummer
          are what it is opened for at a desk. Gruppe and korps share a row because
          they are read as one fact ("12. Århus, DDS").
        -->
        <div class="grid grid-cols-2 gap-3">
          <FloatLabel variant="on">
            <InputText id="edit-crew-group" v-model="editCrewGroup" class="w-full" autocomplete="off" />
            <label for="edit-crew-group">Gruppe / Division</label>
          </FloatLabel>
          <FloatLabel variant="on">
            <!--
              A Select, not a text field: the API casts whatever it receives to a
              CorpsSlug, so free text would mint korps that no filter or export
              knows about. `showClear` because "not stated" is a legitimate answer
              and there has to be a way back to it.
            -->
            <Select
              id="edit-crew-corps"
              v-model="editCrewCorps"
              :options="corpsSelectOptions"
              optionLabel="label"
              optionValue="slug"
              showClear
              class="w-full"
            />
            <label for="edit-crew-corps">Korps</label>
          </FloatLabel>
        </div>
        <FloatLabel variant="on">
          <InputText id="edit-crew-medlemnr" v-model="editCrewMedlemNr" class="w-full" autocomplete="off" />
          <label for="edit-crew-medlemnr">Medlemsnummer</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <InputText id="edit-crew-diet" v-model="editCrewDiet" class="w-full" autocomplete="off" />
          <label for="edit-crew-diet">Kost / allergi</label>
        </FloatLabel>

        <!--
          What the person answered on the signup form. Read-only, and shown because
          the free-text Kost field above is HQ's own note: without this, an empty
          Kost field looks like "no dietary needs" when it may only mean nobody has
          written anything down yet.
        -->
        <div v-if="signupAnswers.length" class="text-sm text-gray-500">
          Fra tilmeldingen:
          <span v-for="(a, i) in signupAnswers" :key="a.label">
            <span v-if="i > 0"> · </span>{{ a.label }}: <span class="text-gray-700">{{ a.value }}</span>
          </span>
        </div>
      </div>
      <template #footer>
        <!--
          The signup page, left of the destructive action so it is nowhere near
          Slet. Only for members who came through the public form — one typed in
          here by an operator has no such page, and a dead link on a screen like
          this is worse than no link.
        -->
        <a
          v-if="signupUrl(editingCrew?.userId)"
          :href="signupUrl(editingCrew?.userId)"
          target="_blank"
          rel="noopener"
          class="mr-auto"
        >
          <Button label="Tilmelding" icon="pi pi-external-link" iconPos="right" text />
        </a>
        <Button label="Slet" icon="pi pi-trash" severity="danger" text :loading="busy" @click="deleteCrew" />
        <Button label="Annuller" text @click="editCrewDialogOpen = false" />
        <Button label="Gem" icon="pi pi-check" :loading="busy" @click="submitEditCrew" />
      </template>
    </Dialog>

    <!-- Create / edit vehicle dialog. One form for both: the fields are the
         same, and only the submit differs. -->
    <Dialog v-model:visible="vehicleDialogOpen" modal :style="{ width: '32rem' }" :header="editingVehicle ? 'Rediger køretøj' : 'Nyt køretøj'">
      <div class="grid gap-3 pt-2">
        <FloatLabel variant="on">
          <InputText id="vehicle-plate" v-model="vehiclePlate" class="w-full" autocomplete="off" placeholder="DK+AB12345" />
          <label for="vehicle-plate">Nummerplade</label>
        </FloatLabel>
        <FloatLabel variant="on">
          <Select id="vehicle-custodian" v-model="vehicleCustodian" :options="custodianOptions" optionLabel="label" optionValue="value" filter class="w-full" />
          <label for="vehicle-custodian">Ansvarlig</label>
        </FloatLabel>
        <div class="grid grid-cols-2 gap-3">
          <FloatLabel variant="on">
            <InputText id="vehicle-brand" v-model="vehicleBrand" class="w-full" autocomplete="off" />
            <label for="vehicle-brand">Mærke</label>
          </FloatLabel>
          <FloatLabel variant="on">
            <InputText id="vehicle-model" v-model="vehicleModel" class="w-full" autocomplete="off" />
            <label for="vehicle-model">Model</label>
          </FloatLabel>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <FloatLabel variant="on">
            <InputText id="vehicle-color" v-model="vehicleColor" class="w-full" autocomplete="off" />
            <label for="vehicle-color">Farve</label>
          </FloatLabel>
          <FloatLabel variant="on">
            <InputNumber id="vehicle-seats" v-model="vehicleSeatCount" :min="0" :max="99" showButtons class="w-full" />
            <label for="vehicle-seats">Passagerpladser</label>
          </FloatLabel>
        </div>
        <FloatLabel variant="on">
          <Textarea id="vehicle-description" v-model="vehicleDescription" class="w-full" rows="3" autoResize />
          <label for="vehicle-description">Bemærkninger</label>
        </FloatLabel>
        <p class="text-xs text-gray-500">Passagerpladser er ud over føreren. 0 betyder, at bilen kun kører ejeren selv.</p>
      </div>
      <template #footer>
        <Button v-if="editingVehicle" label="Slet" icon="pi pi-trash" severity="danger" text :loading="busy" @click="deleteVehicle" />
        <Button label="Annuller" text @click="vehicleDialogOpen = false" />
        <Button :label="editingVehicle ? 'Gem' : 'Opret'" icon="pi pi-check" :loading="busy" @click="submitVehicle" />
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
