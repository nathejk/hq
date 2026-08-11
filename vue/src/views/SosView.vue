<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource, seedLiveResource } from '@/composables/useLiveResource'
import { optimisticWrite } from '@/composables/optimisticWrite'
import SosActivityLine from '@/components/SosActivityLine.vue'
import SosTeamCard from '@/components/SosTeamCard.vue'
import {
  severityOptions,
  severityLabel,
  severityTagSeverity,
  formatDateTime,
} from '@/composables/sos'

const props = defineProps<{ id?: string }>()
const route = useRoute()
const router = useRouter()
const toast = useToast()

const isNew = computed(() => route.name === 'sos-new')
const caseId = computed(() => props.id ?? (route.params.id as string) ?? '')

interface Activity {
  seq: number
  type: string
  actorUserId: string
  activityId?: string
  refActivityId?: string
  value?: string
  createdAt: string
}

interface SosCase {
  id: string
  headline: string
  description: string
  status: 'open' | 'closed'
  severity: string
  assigneeSectionSlug: string
  createdAt: string
  lastActivityAt: string
  timeline?: Activity[]
}

interface SosTeam {
  teamId: string
  teamNumber: string
  name: string
  group: string
  korps: string
  contactName: string
  contactPhone: string
}

// One cache entry per case, keyed by id.
//
// dependsOn names the instance *and* the type: the instance so this case's own
// changes arrive, the type because the timeline and team associations are
// published on the same `sos` token and a signal for a sibling projection would
// otherwise be missed. `immediate: false` while creating — there is nothing to
// fetch yet.
const resource = useLiveResource<{ case: SosCase; teams: SosTeam[] } | null>(
  `sos:${caseId.value || 'new'}`,
  async () => {
    if (!caseId.value) return null
    const response = await http.get(`/sos/${caseId.value}`)
    return response.data
  },
  { dependsOn: caseId.value ? [`sos:${caseId.value}`, 'sos'] : ['sos'], immediate: !isNew.value },
)

const { data, pending, error, refresh } = resource

const sosCase = computed<SosCase | null>(() => data.value?.case ?? null)
const teams = computed<SosTeam[]>(() => data.value?.teams ?? [])
const timeline = computed<Activity[]>(() => sosCase.value?.timeline ?? [])

// A deleted case stops resolving, so a 404 here is the expected way to learn it is
// gone rather than an error to hide.
const isGone = computed(() => {
  const err = error.value as { response?: { status?: number } } | undefined
  return !!err && err.response?.status === 404
})

// --- assignable sections -------------------------------------------------

const { data: orgData } = useLiveResource(
  'sos:sections',
  async () => {
    const response = await http.get('/organisation')
    return response.data as {
      sections: { slug: string; label: string }[]
      sosAssignableSections: string[]
    }
  },
  { dependsOn: ['section', 'sections'] },
)

// Only sections flagged assignable are offered. The list is empty until somebody
// opts sections in on the Organisation page, which is deliberate — better an empty
// select than a case routed to a section that does not handle nødråb.
const assigneeOptions = computed(() => {
  const assignable = new Set(orgData.value?.sosAssignableSections ?? [])
  const options = (orgData.value?.sections ?? [])
    .filter((s) => assignable.has(s.slug))
    .map((s) => ({ value: s.slug, label: s.label }))

  // A case assigned to a section that has since been deleted or un-flagged keeps
  // its assignment; it must therefore remain selectable, or saving anything else
  // would silently clear it.
  const current = sosCase.value?.assigneeSectionSlug
  if (current && !options.some((o) => o.value === current)) {
    const known = (orgData.value?.sections ?? []).find((s) => s.slug === current)
    options.unshift({ value: current, label: known ? known.label : `${current} (slettet sektion)` })
  }
  return options
})

// --- dirty guard ---------------------------------------------------------
//
// This page holds unsaved state: an open headline or description editor, and a
// half-typed comment. An incoming revalidation must not wipe what the operator is
// typing while on the phone — see KlanListView for the same guard on the LOK
// editor. Only the *editors* are guarded; the timeline keeps updating, because
// that is the part another operator's work shows up in.

const editingHeadline = ref(false)
const headlineDraft = ref('')
const editingDescription = ref(false)
const descriptionDraft = ref('')
const commentDraft = ref('')

const dirty = computed(
  () => editingHeadline.value || editingDescription.value || commentDraft.value.trim() !== '',
)

const startHeadlineEdit = () => {
  headlineDraft.value = sosCase.value?.headline ?? ''
  editingHeadline.value = true
}
const startDescriptionEdit = () => {
  descriptionDraft.value = sosCase.value?.description ?? ''
  editingDescription.value = true
}

// --- new case ------------------------------------------------------------

const newHeadline = ref('')
const newDescription = ref('')
const saving = ref(false)

const createCase = async () => {
  if (!newHeadline.value.trim() || !newDescription.value.trim()) {
    toast.add({
      severity: 'warn',
      summary: 'Overskrift og beskrivelse skal udfyldes',
      life: 4000,
    })
    return
  }
  saving.value = true
  try {
    const response = await http.post('/sos', {
      headline: newHeadline.value,
      description: newDescription.value,
    })
    const created = response.data.case as SosCase
    // Seed the new case's cache entry from the response before navigating. The
    // projection is asynchronous, so a fetch on arrival would 404 for a moment and
    // the operator would be told the case they just described does not exist.
    // Seeding means they see it at once, and the live signal replaces it with the
    // projected row a moment later.
    seedLiveResource(`sos:${created.id}`, { case: created, teams: [] })
    await router.push({ name: 'sos', params: { id: created.id } })
  } catch {
    // The axios plugin already surfaces failures as a toast.
  } finally {
    saving.value = false
  }
}

// --- writes --------------------------------------------------------------

const patch = async (body: Record<string, unknown>, optimistic: Partial<SosCase>) => {
  const current = data.value
  if (!current) return
  await optimisticWrite(
    resource,
    { ...current, case: { ...current.case, ...optimistic } },
    () => http.patch(`/sos/${caseId.value}`, body),
  )
}

const saveHeadline = async () => {
  const headline = headlineDraft.value.trim()
  editingHeadline.value = false
  if (!headline || headline === sosCase.value?.headline) return
  await patch({ headline }, { headline })
}

const saveDescription = async () => {
  const description = descriptionDraft.value.trim()
  editingDescription.value = false
  if (!description || description === sosCase.value?.description) return
  await patch({ description }, { description })
}

const setSeverity = (severity: string) => patch({ severity }, { severity })
const setAssignee = (slug: string) =>
  patch({ assigneeSectionSlug: slug ?? '' }, { assigneeSectionSlug: slug ?? '' })

const toggleStatus = async () => {
  const status = sosCase.value?.status === 'closed' ? 'open' : 'closed'
  await patch({ status }, { status })
}

const addComment = async () => {
  const comment = commentDraft.value.trim()
  if (!comment) return
  commentDraft.value = ''
  try {
    await http.post(`/sos/${caseId.value}/comment`, { comment })
    void refresh()
  } catch {
    // Put the text back rather than losing it: it was typed while on the phone.
    commentDraft.value = comment
  }
}

const editingComment = ref<string | null>(null)
const commentEditDraft = ref('')

const startCommentEdit = (activity: Activity) => {
  editingComment.value = activity.activityId ?? null
  commentEditDraft.value = currentCommentText(activity)
}

const saveCommentEdit = async (activity: Activity) => {
  const comment = commentEditDraft.value.trim()
  const id = activity.activityId
  editingComment.value = null
  if (!id || !comment || comment === currentCommentText(activity)) return
  try {
    await http.patch(`/sos/${caseId.value}/comment/${id}`, { comment })
    void refresh()
  } catch {
    // surfaced by the axios plugin
  }
}

// A comment's current text is its latest amendment, if any. The original entry is
// never rewritten — the timeline is append-only — so the view resolves it.
const currentCommentText = (activity: Activity) => {
  const edits = timeline.value.filter(
    (a) => a.type === 'comment.updated' && a.refActivityId === activity.activityId,
  )
  if (edits.length === 0) return activity.value ?? ''
  return edits[edits.length - 1].value ?? ''
}

const isEdited = (activity: Activity) =>
  timeline.value.some(
    (a) => a.type === 'comment.updated' && a.refActivityId === activity.activityId,
  )

// The rendered timeline hides the amendment entries themselves and shows the
// amended comment with a "redigeret" marker instead — the record is intact, the
// screen is readable.
const visibleTimeline = computed(() => timeline.value.filter((a) => a.type !== 'comment.updated'))

const deleteCase = async () => {
  if (!confirm('Slet sagen? Den skjules fra listerne.')) return
  try {
    await http.delete(`/sos/${caseId.value}`)
    await router.push({ name: 'sos-list' })
  } catch {
    // surfaced by the axios plugin
  }
}

watch(error, (err) => {
  if (!err || isGone.value) return
  toast.add({ severity: 'error', summary: 'Kunne ikke hente sagen', life: 5000 })
})
</script>

<template>
  <!-- New case: two fields and a save, because this is typed while the phone is ringing. -->
  <div v-if="isNew" class="card" id="sos-new">
    <h1 class="font-nathejk text-2xl mb-4">Ny sag</h1>
    <div class="flex flex-col gap-3 max-w-2xl">
      <div>
        <label class="block text-sm mb-1">Overskrift</label>
        <InputText v-model="newHeadline" class="w-full" placeholder="Fx: Forstuvet ankel ved post 4" autofocus />
      </div>
      <div>
        <label class="block text-sm mb-1">Beskrivelse</label>
        <Textarea v-model="newDescription" class="w-full" rows="4" placeholder="Hvad ringer de om?" />
      </div>
      <div class="flex gap-2">
        <Button label="Opret sag" icon="pi pi-check" :loading="saving" @click="createCase" />
        <Button label="Annuller" severity="secondary" text @click="router.push({ name: 'sos-list' })" />
      </div>
    </div>
  </div>

  <div v-else-if="isGone" class="card">
    <h1 class="font-nathejk text-2xl mb-2">Sagen er slettet</h1>
    <p class="mb-4">Sagen findes ikke længere.</p>
    <Button label="Tilbage til nødtelefonen" @click="router.push({ name: 'sos-list' })" />
  </div>

  <div v-else-if="sosCase" class="grid grid-cols-1 lg:grid-cols-3 gap-4" id="sos-detail">
    <!-- Left: the case and its history -->
    <div class="lg:col-span-2 card">
      <div class="flex items-start justify-between gap-2">
        <div class="flex-1">
          <div v-if="editingHeadline" class="flex gap-2">
            <InputText v-model="headlineDraft" class="flex-1" @keyup.enter="saveHeadline" />
            <Button icon="pi pi-check" @click="saveHeadline" />
            <Button icon="pi pi-times" severity="secondary" text @click="editingHeadline = false" />
          </div>
          <h1 v-else class="font-nathejk text-2xl">
            {{ sosCase.headline }}
            <Button icon="pi pi-pencil" text rounded size="small" @click="startHeadlineEdit" />
          </h1>
        </div>
        <Tag :value="sosCase.status === 'closed' ? 'Afsluttet' : 'Åben'"
             :severity="sosCase.status === 'closed' ? 'secondary' : 'info'" />
      </div>

      <div class="text-sm text-surface-500 mt-1">
        Oprettet {{ formatDateTime(sosCase.createdAt) }}
      </div>

      <div class="mt-4">
        <div v-if="editingDescription" class="flex flex-col gap-2">
          <Textarea v-model="descriptionDraft" rows="4" class="w-full" />
          <div class="flex gap-2">
            <Button label="Gem" icon="pi pi-check" size="small" @click="saveDescription" />
            <Button label="Annuller" severity="secondary" text size="small" @click="editingDescription = false" />
          </div>
        </div>
        <p v-else class="whitespace-pre-wrap">
          {{ sosCase.description }}
          <Button icon="pi pi-pencil" text rounded size="small" @click="startDescriptionEdit" />
        </p>
      </div>

      <div class="flex flex-wrap gap-2 mt-4">
        <Button
          :label="sosCase.status === 'closed' ? 'Genåbn sag' : 'Luk sag'"
          :icon="sosCase.status === 'closed' ? 'pi pi-replay' : 'pi pi-check-circle'"
          @click="toggleStatus"
        />
        <Button label="Slet sag" icon="pi pi-trash" severity="danger" text @click="deleteCase" />
      </div>

      <!-- Updates paused, said on screen: an operator must know why the page is
           not moving while they type. -->
      <div v-if="dirty" class="mt-4 text-sm italic text-amber-700">
        Opdateringer er sat på pause, mens du skriver.
      </div>

      <h2 class="font-nathejk text-xl mt-6 mb-2">Hændelser</h2>
      <div class="flex flex-col gap-1">
        <SosActivityLine
          v-for="activity in visibleTimeline"
          :key="activity.seq"
          :activity="activity"
          :comment-text="activity.type === 'commented' ? currentCommentText(activity) : activity.value"
          :edited="activity.type === 'commented' && isEdited(activity)"
          :editing="editingComment === activity.activityId"
          v-model:draft="commentEditDraft"
          @edit="startCommentEdit(activity)"
          @save="saveCommentEdit(activity)"
          @cancel="editingComment = null"
        />
        <div v-if="visibleTimeline.length === 0 && pending" class="text-sm text-surface-500">
          Henter hændelser...
        </div>
      </div>

      <div class="mt-4">
        <Textarea v-model="commentDraft" rows="2" class="w-full" placeholder="Tilføj kommentar..." />
        <Button label="Tilføj kommentar" icon="pi pi-comment" size="small" class="mt-2"
                :disabled="!commentDraft.trim()" @click="addComment" />
      </div>
    </div>

    <!-- Right: handling -->
    <div class="flex flex-col gap-4">
      <div class="card">
        <h2 class="font-nathejk text-xl mb-3">Prioritet</h2>
        <div class="flex gap-2">
          <Button
            v-for="option in severityOptions"
            :key="option.value"
            :label="option.label"
            :severity="severityTagSeverity(option.value)"
            :outlined="sosCase.severity !== option.value"
            size="small"
            @click="setSeverity(option.value)"
          />
        </div>

        <h2 class="font-nathejk text-xl mt-4 mb-2">Tildelt</h2>
        <Select
          :modelValue="sosCase.assigneeSectionSlug"
          :options="assigneeOptions"
          optionLabel="label"
          optionValue="value"
          placeholder="Ingen"
          showClear
          class="w-full"
          @update:modelValue="setAssignee"
        />
        <small v-if="assigneeOptions.length === 0" class="block mt-2 text-surface-500">
          Ingen sektioner kan tildeles nødråb endnu. Slå det til på Organisation-siden.
        </small>
      </div>

      <SosTeamCard :sos-id="caseId" :teams="teams" @changed="refresh" />
    </div>
  </div>

  <div v-else class="card">
    <ProgressSpinner v-if="pending" />
  </div>
</template>

<style>
#sos-detail .card {
  padding: 1rem;
}
</style>
