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
  handoutOptions,
  handoutToId,
  handoutToOption,
  isDegenerate,
  orderPicks,
  sameExtent,
  setExtents,
  splitCheckgroups,
  teamTypeLabel,
  teamTypeOptions,
  teamTypeToOption,
  teamTypeToValue,
  TEAM_TYPE_NONE_OPTION,
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
  /**
   * An existing area the operator has just dragged or resized on the map.
   *
   * By index, because it replaces a known rectangle rather than filling an armed slot the way `pick`
   * does. The map owns the gesture; this dialog stays the owner of the draft, so a drag lands in the
   * same unsaved buffer as typing does and is committed by the same "Gem kort".
   */
  extentEdit?: { index: number; extent: Extent; seq: number } | null
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
  /** A whole set's extents, drawn so an operator can judge coverage against the terrain. */
  (e: 'update:overlay', value: Extent[]): void
  /**
   * The areas of the one sheet under the pointer, drawn emphasised on top of the set's outlines.
   *
   * Separate from `overlay` rather than a flag on it: the set's coverage and "which of these is this
   * row?" are two questions asked at once, and the whole point is to see the one against the many.
   */
  (e: 'update:hoverExtents', value: Extent[]): void
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

// The edit buffer deliberately holds **no** `kortsaetId`. Moving a sheet between sets is
// a drag, not a dropdown: the list on the left already shows the sets in order with their sheets, so
// a select saying the same thing twice was both redundant and the easiest way to move a sheet by
// accident. Until the drag exists, a sheet's set is fixed at creation.
const draft = ref<{ name: string; format: string; note: string; handoutCheckgroupId: string }>({
  name: '',
  format: '',
  note: '',
  handoutCheckgroupId: '',
})
const saving = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const loadDraft = (sheet?: Kort) => {
  draft.value = {
    name: sheet?.name ?? '',
    format: sheet?.format ?? '',
    note: sheet?.note ?? '',
    // The dropdown's vocabulary, not the API's: `''` cannot be a selected value in PrimeVue's
    // Select, so "handed out at the QR scan" travels as its own option here (see handoutToOption).
    handoutCheckgroupId: handoutToOption(sheet?.handoutCheckgroupId),
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
    draft.value.handoutCheckgroupId !== handoutToOption(sheet.handoutCheckgroupId)
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
  // Caught before anything is sent, so a bad rectangle cannot let the name and the checkpoints
  // through and then fail: one button has to mean one outcome. The API refuses it too, but saying so
  // here means "vælg to forskellige hjørner" arrives while the operator still remembers clicking.
  const flat = draftExtents.value.findIndex(isDegenerate)
  if (flat !== -1) {
    fieldErrors.value = { extents: 'Området har ingen udstrækning — vælg to forskellige hjørner' }
    return
  }

  saving.value = true
  fieldErrors.value = {}
  try {
    // The description and the areas go in one PUT because they are one endpoint.
    //
    // Every field is sent, not only the ones the local dirty check thinks moved. That check exists to
    // enable the button and to protect unsaved work; making it also decide the *contents* of the
    // request gave it a second job it could fail silently at — a value the operator had chosen would
    // simply not be in the body, the save would report success, and the field would revert on the
    // next refresh. The server dirty-checks per field anyway, so restating an unchanged value costs
    // one comparison and publishes nothing.
    await http.put(
      `/kort/${sheet.id}`,
      {
        name: draft.value.name.trim(),
        // An empty format is omitted rather than sent: "" is not one of the four values and the API
        // would refuse it, and a sheet whose format is not yet decided is normal.
        ...(draft.value.format ? { format: draft.value.format } : {}),
        note: draft.value.note,
        // Back to the API's vocabulary: the QR option becomes `""`. Sent whatever its value, because
        // `""` is meaningful — omitting it when empty would make the QR rule unsaveable.
        handoutCheckgroupId: handoutToId(draft.value.handoutCheckgroupId),
        extents: draftExtents.value,
      },
      { withCredentials: true },
    )
    // The checkpoints are a second request because they are a second endpoint — a replace, not a
    // patch (see `PUT /kort/:id/checkpoints`). Sent after the description so that if it fails, the
    // sheet is at worst described but not re-pointed, which is the harmless direction.
    if (picksDirty.value) {
      await http.put(
        `/kort/${sheet.id}/checkpoints`,
        // In checkgroup order rather than tick order, so the stored list is stable and re-saving an
        // unchanged selection stays a no-op on the server.
        { checkpointIds: orderedPicks.value },
        { withCredentials: true },
      )
    }
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

/**
 * Discard every unsaved change to the sheet — fields, areas and checkpoints together.
 *
 * The counterpart of one save button: an "Annullér" that reverted only the text while leaving a
 * half-drawn rectangle behind would be the same confusion as three saves, one step later.
 */
const cancelEdit = () => {
  const sheet = selected.value
  if (sheet) loadBuffers(sheet)
  fieldErrors.value = {}
}

/**
 * The "Udleveret" choices, in course order with the QR rule last.
 *
 * Built from the checkgroups the view already passed in for the picker. A sheet pointing at a
 * checkgroup that has since been deleted reads back from the API as the QR rule, so there is no
 * dangling id to render an option for.
 */
const handoutChoices = computed(() => handoutOptions(props.checkgroups ?? []))

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

// --- the tree the picker renders ---
//
// One `TreeSelect` in checkbox mode instead of the flat list of every checkgroup and every
// checkpoint that used to be inlined here. The list was ~60 rows for a normal year, which pushed the
// extents and the buttons off screen and made the dialog scroll past the thing being edited. The
// tree collapses to one line and keeps the two gestures that matter: tick a whole checkgroup (its
// parent node), or tick single checkpoints (a skitse).
//
// The parent nodes are *not* selectable values — `selectable: false` — because a checkgroup is not
// something a sheet contains; ticking one is shorthand for its children, which is exactly what
// checkbox mode's propagation does.

/** A key prefix for group nodes, so a checkgroup id can never collide with a checkpoint id. */
const GROUP_KEY = 'cg:'

const checkpointTree = computed(() =>
  (props.checkgroups ?? []).map((group) => ({
    key: GROUP_KEY + group.id,
    label: group.name,
    selectable: false,
    children: group.checkpoints.map((cp) => ({
      key: cp.id,
      // The missing-position warning cannot ride along as an icon inside the overlay without
      // depending on TreeSelect's node slot, so it is summarised under the field instead — see
      // pickedWithoutPosition. A checkpoint with no pin is still perfectly assignable.
      label: cp.name,
      leaf: true,
    })),
  })),
)

/** Every checkpoint id the tree offers, for sorting out leaves from parents on the way back in. */
const leafIds = computed(() => new Set((props.checkgroups ?? []).flatMap((g) => g.checkpoints.map((cp) => cp.id))))

/**
 * `picked` in the shape TreeSelect wants, and back again.
 *
 * TreeSelect's checkbox model is a map of node key to `{ checked, partialChecked }`, and it needs the
 * *parent* keys in there too or a fully-ticked checkgroup renders unticked. `picked` stays the single
 * source of truth — it is what the save, the dirty check and the map highlighting read — so this is a
 * projection of it rather than a second copy that could drift.
 */
const pickedKeys = computed({
  get: () => {
    const keys: Record<string, { checked: boolean; partialChecked: boolean }> = {}
    for (const group of props.checkgroups ?? []) {
      let ticked = 0
      for (const cp of group.checkpoints) {
        if (picked.value.has(cp.id)) {
          keys[cp.id] = { checked: true, partialChecked: false }
          ticked++
        }
      }
      if (ticked === 0) continue
      const all = ticked === group.checkpoints.length
      keys[GROUP_KEY + group.id] = { checked: all, partialChecked: !all }
    }
    return keys
  },
  set: (keys: Record<string, { checked?: boolean; partialChecked?: boolean }> | null) => {
    // Only leaves become picks: parent keys are bookkeeping, and a partially-checked one would
    // otherwise be saved as a checkpoint id that does not exist.
    const next = new Set<string>()
    for (const [key, state] of Object.entries(keys ?? {})) {
      if (state?.checked && leafIds.value.has(key)) next.add(key)
    }
    picked.value = next
  },
})

/**
 * Picked checkpoints that have no position yet.
 *
 * Named under the field rather than flagged per row, now that the rows live inside an overlay. The
 * information is worth keeping — a checkpoint with no pin cannot be drawn on the printed sheet — and
 * it is more useful about the ones actually chosen than about all sixty.
 */
const pickedWithoutPosition = computed(() =>
  (props.checkgroups ?? [])
    .flatMap((group) => group.checkpoints)
    .filter((cp) => picked.value.has(cp.id) && !hasPosition(cp))
    .map((cp) => cp.name),
)

const orderedPicks = computed(() => orderPicks(props.checkgroups ?? [], picked.value))

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

// `teamType` here holds the *dropdown's* vocabulary, not the API's: `null` cannot be a selected value
// in PrimeVue's Select, so "no particular team type" travels as its own option (teamTypeToOption).
const setDraft = ref<{ name: string; teamType: string }>({
  name: '',
  teamType: TEAM_TYPE_NONE_OPTION,
})

const editSet = (set: Kortsaet) => {
  editingSetId.value = set.id
  setDraft.value = { name: set.name, teamType: teamTypeToOption(set.teamType) }
  fieldErrors.value = {}
}

const newSet = () => {
  editingSetId.value = 'new'
  // Not defaulted to `patrulje`: an unmarked set is the commonest one, and a default here would
  // quietly mark a crew set as the spejder set — which the hej-app would then serve to patrols.
  setDraft.value = { name: '', teamType: TEAM_TYPE_NONE_OPTION }
  fieldErrors.value = {}
}

const cancelSetEdit = () => {
  editingSetId.value = undefined
  fieldErrors.value = {}
}

/** Unsaved set edits, so they join the same dirty guard as everything else. */
const setDirty = computed(() => {
  if (editingSetId.value === 'new') {
    return setDraft.value.name.trim() !== '' || setDraft.value.teamType !== TEAM_TYPE_NONE_OPTION
  }
  const set = sets.value.find((candidate) => candidate.id === editingSetId.value)
  if (!set) return false
  // Trimmed, for the same reason as a sheet's name: the server trims, and an untrimmed comparison
  // would leave the set editor stuck open with nothing actually unsaved.
  return setDraft.value.name.trim() !== set.name || setDraft.value.teamType !== teamTypeToOption(set.teamType)
})

const saveSet = async () => {
  saving.value = true
  fieldErrors.value = {}
  try {
    // Back to the API's vocabulary: the "ingen" option becomes null, which is what clears the marking.
    const body = { name: setDraft.value.name.trim(), teamType: teamTypeToValue(setDraft.value.teamType) }
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

// An existing rectangle was dragged or resized on the map.
//
// Lands in the same draft as everything else, so the areas are saved by "Gem kort" along with the
// name and the checkpoints, and "Annullér" puts a mis-drag back. A drag is therefore also what pauses
// live updates, via `extentsDirty` — the same protection typing already had.
watch(
  () => props.extentEdit?.seq,
  () => {
    const edit = props.extentEdit
    if (!edit) return
    const next = [...draftExtents.value]
    // Only an index that exists: the map draws what this dialog gave it, so an out-of-range index
    // would mean the two had drifted apart, and appending a rectangle nobody drew would be a strange
    // way to find that out.
    if (edit.index < 0 || edit.index >= next.length) return
    next[edit.index] = edit.extent
    draftExtents.value = next
  },
)

/** A short, readable rendering of a rectangle's corners. */
const extentLabel = (extent: Extent) =>
  `${extent.northWest.latitude.toFixed(4)}, ${extent.northWest.longitude.toFixed(4)} → ` +
  `${extent.southEast.latitude.toFixed(4)}, ${extent.southEast.longitude.toFixed(4)}`

// --- a set's areas on the map (task 132, revised in 155) ---
//
// Shows every area in a set at once, so an operator can see how the sheets sit against the terrain
// and each other. It no longer computes "gaps": the set's bounding box is not something that has to
// be covered — the route is not a rectangle, so the corners between its legs are ground nobody walks
// — and flagging them made every healthy set look broken. Judging coverage is a job for eyes on the
// terrain; this just puts the areas in front of them.

/** Which set's areas are being shown, if any. */
const overlaySetId = ref<string | undefined>(undefined)

const overlaySet = computed(() => sets.value.find((set) => set.id === overlaySetId.value))

/** The areas on show — every extent of every sheet in the set. */
const overlayExtents = computed(() => (overlaySet.value ? setExtents(overlaySet.value) : []))

/**
 * Show or hide a set's areas.
 *
 * Bound to the set's **name**, not to an icon beside it: "how does this set cover the ground?" is a
 * question about the set, so the set itself is the control. The eye icon is left as a state
 * indicator only.
 */
const toggleOverlay = (set: Kortsaet) => {
  overlaySetId.value = overlaySetId.value === set.id ? undefined : set.id
}

/**
 * Emphasise one sheet's areas while the pointer is on its row.
 *
 * Only while a set's areas are on show: outside that, a hover highlight would be a rectangle
 * appearing from nowhere for a sheet the operator has not selected. With the areas up it is the
 * point — it says which of them is this row.
 *
 * The selected sheet is excluded, because its areas are already drawn as the editing preview and a
 * second, thicker outline on top of them reads as a second area.
 */
const hoverSheet = (sheet: Kort | null) => {
  if (!sheet || overlaySetId.value === undefined || sheet.id === props.selectedId) {
    emit('update:hoverExtents', [])
    return
  }
  emit('update:hoverExtents', sheet.extents)
}

watch(overlayExtents, (extents) => emit('update:overlay', extents), { deep: true })

// Turning the coverage off must take any lingering emphasis with it: the pointer may still be on a
// row, and `mouseleave` will not fire if the rows re-render underneath it.
watch(overlaySetId, (id) => {
  if (id === undefined) emit('update:hoverExtents', [])
})

/** How many sheets in the set have no area recorded — a skitse, or one nobody has drawn yet. */
const overlayWithoutExtent = computed(() => overlaySet.value?.kort.filter((sheet) => sheet.extents.length === 0).length ?? 0)

// --- the split-checkgroup warning (task 133) ---
//
// A warning, never a block. A half-entered set trips it constantly — the operator ticks one
// checkpoint of a group and it fires — and a save that refused to complete during data entry would
// be worse than the mistake it is guarding against.

const splitsBySet = computed(() =>
  sets.value.map((set) => ({ set, splits: splitCheckgroups(set, props.checkgroups ?? []) })),
)

const splitsFor = (set: Kortsaet) => splitsBySet.value.find((entry) => entry.set.id === set.id)?.splits ?? []

// --- unsaved state, all of it ---
//
// Declared here, after every source, because the emit below runs immediately: a computed that
// referenced one of these before its `const` was initialised would throw during setup.

/**
 * Unsaved work on the selected sheet: its description, its areas or its checkpoints.
 *
 * One flag because there is one save button. The three parts used to have a button each, which read
 * as three independent records and made "have I saved?" a question with three answers — and the
 * checkpoints in particular were easy to leave behind, since they are the part a consuming app
 * actually needs.
 */
const sheetDirty = computed(() => dirty.value || extentsDirty.value || picksDirty.value)

/** Anything unsaved: the sheet, a set being edited, or a drag not yet saved. */
const anyDirty = computed(() => sheetDirty.value || setDirty.value || reordering.value)

/** Which sheet the buffers currently hold. See `deferSameSheet` for why this has to be tracked. */
const loadedId = ref<string | undefined>(undefined)

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
  loadedId.value = sheet.id
}

/**
 * Defer only a *refresh of the sheet already loaded* — never the arrival of a different one.
 *
 * This distinction is the whole correctness of the guard, and getting it wrong deadlocked the
 * dialog: `anyDirty` compares the buffers against `selected`, so at the instant the selection
 * changes it is comparing the **previous** sheet's buffers against the **new** sheet and is
 * therefore true — an empty buffer against any sheet reads as "unsaved changes". Pausing on that
 * meant the buffers were never loaded, which kept them unequal, which kept the condition true. The
 * editor opened with blank fields and every other sheet was then refused with "gem eller annullér
 * først", with nothing to save.
 *
 * A dirty flag derived from the incoming value cannot gate that value's own first application. So
 * only a payload for the sheet the buffers already hold can be deferred; a change of sheet always
 * applies, which is safe because `select()` refuses to change the selection while dirty — so a
 * changed id is a clean moment by construction.
 */
const deferSameSheet = computed(() => anyDirty.value && selected.value?.id === loadedId.value)

/**
 * Follow the selected sheet — unless the operator has unsaved work on it.
 *
 * This is the bug that made task 131 worth doing, and it is not the one the task description
 * anticipated. The dialog reads the sheets straight from the live cache, so a payload arriving while
 * someone typed used to re-run the load and **wipe the field under the cursor**: another operator
 * renaming any sheet, or this operator's own save of a *different* sheet, was enough. Deferring only
 * the Leaflet markers would have left that untouched.
 *
 * The same composable as the map, for the same reason: watching the condition cannot miss an exit.
 */
useDeferredApply(selected, deferSameSheet, loadBuffers)

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
                <!-- The name is the coverage toggle: "is this set's ground fully covered?" is a
                     question about the set, so the set is the control. -->
                <button
                  class="flex min-w-0 items-center gap-1 truncate text-left font-medium hover:text-blue-700"
                  :title="overlaySetId === set.id ? 'Skjul sættets områder' : 'Vis alle sættets områder på kortet'"
                  @click="toggleOverlay(set)"
                >
                  <!-- The same eye either way, green when the set's areas are on the map. A crossed
                       eye reads as "hidden", which is the state you are *leaving*, not the one you
                       are in — so the icon changed meaning depending on how you read it. Colour
                       alone says "on" without that ambiguity. -->
                  <i class="pi pi-eye text-xs" :class="overlaySetId === set.id ? 'text-green-600' : 'text-gray-400'" />
                  <span class="truncate">{{ set.name }}</span>
                  <!-- The team-type marking, shown because it is what the hej-app matches on: an
                       operator needs to see which set is “the spejder set” without opening it. -->
                  <span v-if="set.teamType" class="text-xs font-normal text-gray-500">({{ teamTypeLabel(set.teamType) }})</span>
                </button>
              </div>
              <div class="flex items-center">
                <Button icon="pi pi-pencil" text size="small" :disabled="saving || anyDirty" @click="editSet(set)" />
                <Button icon="pi pi-plus" text size="small" :disabled="saving || anyDirty" @click="createSheet(set)" />
              </div>
            </div>

            <!-- What is on show, and how to read it. No coverage verdict: see the note in the
                 script for why the gap count was removed. -->
            <div v-if="overlaySetId === set.id" class="rounded bg-blue-50 px-2 py-1 text-xs text-gray-600">
              <div v-if="overlayExtents.length === 0">Ingen af sættets kort har et område endnu.</div>
              <div v-else>
                Viser {{ overlayExtents.length }} område{{ overlayExtents.length === 1 ? '' : 'r' }} på kortet — hold
                musen over et kort for at fremhæve det.
              </div>
              <div v-if="overlayWithoutExtent > 0" class="text-gray-500">
                {{ overlayWithoutExtent }} kort uden område (fx skitser).
              </div>
            </div>

            <draggable :list="set.kort" handle=".kort-handle" item-key="id" @start="reordering = true" @end="saveOrder(set)">
              <template #item="{ element: sheet }">
                <div
                  class="flex items-center gap-2 rounded px-2 py-1 text-sm"
                  :class="sheet.id === selectedId ? 'bg-blue-50 ring-1 ring-blue-300' : 'hover:bg-gray-50'"
                  @mouseenter="hoverSheet(sheet)"
                  @mouseleave="hoverSheet(null)"
                >
                  <i class="kort-handle pi pi-bars cursor-move text-xs text-gray-400" />
                  <button class="flex-1 truncate text-left" @click="select(sheet)">{{ sheet.name }}</button>
                  <span class="text-xs text-gray-400">{{ sheet.checkpointIds.length }}</span>
                </div>
              </template>
            </draggable>

            <div v-if="set.kort.length === 0" class="px-2 text-xs text-gray-400">Ingen kort i sættet</div>

            <!-- No single sheet covers this checkgroup, so a patrol shown the group would see posts
                 it holds no map for. A warning only: a half-entered set trips this constantly. -->
            <div v-for="split in splitsFor(set)" :key="split.checkgroupId" class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-800">
              <i class="pi pi-exclamation-triangle" />
              <strong>{{ split.checkgroupName }}</strong> er delt over
              {{ split.sheetNames.join(', ') }} — ingen af kortene viser hele postgruppen.
            </div>
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

      <!-- The editor for the selected sheet. Same blue box as the set editor above: both are "you
           are editing this one thing now", and one visual language for that beats two. -->
      <div v-if="selected" class="space-y-3 rounded border border-blue-200 bg-blue-50 p-2">
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

        <!-- Where the sheet is handed out, which is what decides when its checkpoints become
             visible to the scout in the hej app: at the named post, or at the scan of this
             sheet's own QR code. -->
        <div>
          <label class="block text-sm text-gray-700">Udleveret</label>
          <Select
            v-model="draft.handoutCheckgroupId"
            :options="handoutChoices"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
          <small class="text-xs text-gray-500">
            Afgør hvornår kortets poster bliver synlige for spejderen i hej-appen.
          </small>
          <small v-if="fieldErrors.handoutCheckgroupId" class="block text-red-600">
            {{ fieldErrors.handoutCheckgroupId }}
          </small>
        </div>

        <div>
          <label class="block text-sm text-gray-700">Note</label>
          <Textarea v-model="draft.note" rows="2" class="w-full" autoResize />
        </div>

        <!-- The ground this sheet shows. Zero areas is normal — a skitse has none worth recording;
             two means a double-sided sheet, and they are simply two areas with no front or back. -->
        <div class="space-y-1 border-t border-blue-200 pt-3">
          <div class="text-sm font-medium">Områder på kortet</div>

          <div v-if="draftExtents.length === 0 && pickingIndex === null" class="text-xs text-gray-400">
            Ingen områder — fx en skitse.
          </div>

          <!-- Said once, because the gesture is not visible: the rectangle looks the same whether or
               not it can be dragged. -->
          <div v-else-if="pickingIndex === null" class="text-xs text-gray-500">
            Træk området på kortet for at flytte det, eller træk et hjørne for at ændre størrelsen.
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

        <!-- The checkpoints drawn on this sheet. The part a consuming app cares about: this list
             is what may be revealed once the sheet is known to be in a team's hands. -->
        <div class="space-y-1 border-t border-blue-200 pt-3">
          <div class="text-sm font-medium">Poster på kortet</div>

          <div v-if="(checkgroups ?? []).length === 0" class="text-xs text-gray-400">Ingen poster fundet</div>

          <!-- Checkbox mode, so ticking a checkgroup ticks its checkpoints: a sheet almost always
               carries whole checkgroups, because a checkgroup is revealed as a whole. Partial
               selections stay possible — a skitse is exactly that. -->
          <TreeSelect
            v-else
            v-model="pickedKeys"
            :options="checkpointTree"
            selectionMode="checkbox"
            :metaKeySelection="false"
            filter
            filterPlaceholder="Søg efter post"
            placeholder="Vælg poster"
            class="w-full"
            :pt="{ tree: { root: { class: 'max-h-72 overflow-auto' } } }"
          >
            <!-- The chosen posts are summarised as a count. Listing sixty names in the closed field
                 is what made this picker unreadable in the first place. -->
            <template #value>
              <span v-if="picked.size === 0" class="text-gray-400">Vælg poster</span>
              <span v-else>{{ picked.size }} poster valgt</span>
            </template>
          </TreeSelect>

          <!-- A checkpoint with no position can be assigned; it just cannot be drawn on the printed
               sheet. Named rather than blocked, since the sheet may be drawn before the pin is
               placed. -->
          <small v-if="pickedWithoutPosition.length" class="block text-xs text-amber-700">
            Uden placering: {{ pickedWithoutPosition.join(', ') }}
          </small>
        </div>

        <!-- One save for the whole sheet: name, format, handout, note, areas and posts. Last in the
             editor rather than under the description, so it reads as belonging to everything above
             it. Sticky, because the editor is taller than the dialog and a save button that scrolls
             out of sight is how unsaved work gets abandoned. -->
        <div class="sticky bottom-0 -mx-2 -mb-2 flex items-center justify-between rounded-b border-t border-blue-200 bg-blue-50 px-2 py-2">
          <Button label="Slet" severity="danger" text size="small" :disabled="saving" @click="deleteSheet" />
          <div class="flex items-center gap-2">
            <span v-if="sheetDirty" class="text-xs text-amber-700">Ugemte ændringer</span>
            <Button label="Annullér" severity="secondary" text size="small" :disabled="saving || !sheetDirty" @click="cancelEdit" />
            <Button label="Gem kort" size="small" :loading="saving" :disabled="saving || !sheetDirty" @click="saveSheet" />
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
