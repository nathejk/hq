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
  formatTime,
  activityIcon,
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

// The rendered timeline is what *happened to the case*, not what the case is.
//
// Creation and headline/description edits are hidden: the card above states the
// current title and description, so an entry saying they were set is noise on the
// one surface an operator reads during a handover. They are still recorded as
// events and rows — the audit trail is intact, only the display is curated.
//
// Comment amendments are hidden for a different reason: the comment they amend is
// shown with its latest text and a "(redigeret)" marker, so a separate entry would
// say the same thing twice.
const HIDDEN_TIMELINE_TYPES = new Set([
  'created',
  'headline.updated',
  'description.updated',
  'comment.updated',
])

const visibleTimeline = computed(() =>
  timeline.value.filter((a) => !HIDDEN_TIMELINE_TYPES.has(a.type)),
)

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

  <div v-else-if="sosCase" class="grid grid-cols-1 lg:grid-cols-4 gap-4" id="sos-detail">
    <!--
      Left, and given the room: what the case *is*, then what has happened to it.
      The handling controls sit in a deliberately quieter column — they are set once
      or twice per case, while the headline, the description and the timeline are
      read on every glance and during every handover.
    -->
    <!--
      One panel: the case, then everything that happened to it, then the box for
      adding to it. Same outer surface for all three, because they are read as one
      story — and the case itself is a note on that surface exactly like a comment,
      only full width rather than sitting on the timeline rail.
    -->
    <div class="lg:col-span-3 card">
      <Card class="sos-note">
        <template #title>
          <div class="flex items-start justify-between gap-3">
            <div v-if="editingHeadline" class="flex gap-2 flex-1">
              <InputText v-model="headlineDraft" class="flex-1" autofocus @keyup.enter="saveHeadline" />
              <Button icon="pi pi-check" @click="saveHeadline" />
              <Button icon="pi pi-times" severity="secondary" text @click="editingHeadline = false" />
            </div>
            <h1 v-else class="font-nathejk text-3xl leading-tight flex-1">
              {{ sosCase.headline }}
              <Button class="align-middle" icon="pi pi-pencil" text rounded size="small"
                      @click="startHeadlineEdit" />
            </h1>
            <Tag :value="sosCase.status === 'closed' ? 'Afsluttet' : 'Åben'"
                 :severity="sosCase.status === 'closed' ? 'secondary' : 'info'" />
          </div>
        </template>

        <template #subtitle>
          <span class="text-sm">Oprettet {{ formatDateTime(sosCase.createdAt) }}</span>
          <Tag v-if="sosCase.severity" class="ml-2" :value="severityLabel(sosCase.severity)"
               :severity="severityTagSeverity(sosCase.severity)" />
        </template>

        <template #content>
          <div v-if="editingDescription" class="flex flex-col gap-2">
            <Textarea v-model="descriptionDraft" rows="5" class="w-full" autofocus />
            <div class="flex gap-2">
              <Button label="Gem" icon="pi pi-check" size="small" @click="saveDescription" />
              <Button label="Annuller" severity="secondary" text size="small"
                      @click="editingDescription = false" />
            </div>
          </div>
          <p v-else class="whitespace-pre-wrap text-base">
            {{ sosCase.description }}
            <Button class="align-middle" icon="pi pi-pencil" text rounded size="small"
                    @click="startDescriptionEdit" />
          </p>

          <!-- Updates paused, said on screen: an operator must know why the page is
               not moving while they type. -->
          <div v-if="dirty" class="mt-3 text-sm italic text-amber-700">
            Opdateringer er sat på pause, mens du skriver.
          </div>
        </template>

        <template #footer>
          <div class="flex flex-wrap gap-2">
            <Button
              :label="sosCase.status === 'closed' ? 'Genåbn sag' : 'Luk sag'"
              :icon="sosCase.status === 'closed' ? 'pi pi-replay' : 'pi pi-check-circle'"
              @click="toggleStatus"
            />
            <Button label="Slet sag" icon="pi pi-trash" severity="danger" text @click="deleteCase" />
          </div>
        </template>
      </Card>

      <h2 class="text-xs uppercase tracking-wide text-gray-500 mt-5 mb-2">Hændelser</h2>

      <Timeline v-if="visibleTimeline.length" :value="visibleTimeline" class="sos-timeline">
        <template #opposite="{ item }">
          <span class="text-xs text-gray-500">{{ formatTime(item.createdAt) }}</span>
        </template>
        <template #marker="{ item }">
          <span class="flex w-7 h-7 items-center justify-center rounded-full bg-gray-100 text-gray-600">
            <i :class="activityIcon(item.type)" />
          </span>
        </template>
        <template #content="{ item }">
          <SosActivityLine
            :activity="item"
            :comment-text="item.type === 'commented' ? currentCommentText(item) : item.value"
            :edited="item.type === 'commented' && isEdited(item)"
            :editing="editingComment === item.activityId"
            v-model:draft="commentEditDraft"
            @edit="startCommentEdit(item)"
            @save="saveCommentEdit(item)"
            @cancel="editingComment = null"
          />
        </template>
      </Timeline>

      <div v-else class="text-sm text-gray-500">
        {{ pending ? 'Henter hændelser...' : 'Endnu ingen hændelser på sagen.' }}
      </div>

      <div class="mt-4">
        <Textarea v-model="commentDraft" rows="2" class="w-full" placeholder="Tilføj kommentar..." />
        <Button label="Tilføj kommentar" icon="pi pi-comment" size="small" class="mt-2"
                :disabled="!commentDraft.trim()" @click="addComment" />
      </div>
    </div>

    <!--
      Handling: priority, assignee, patrols. Dimmed until hovered and kept narrow,
      because these are occasional actions sitting beside something read constantly.
      Dimmed rather than hidden — an operator taking over a shift still needs to see
      at a glance who the case is assigned to.
    -->
    <aside class="sos-aside flex flex-col gap-3 text-sm">
      <div class="card !p-3">
        <h2 class="text-xs uppercase tracking-wide text-gray-500 mb-2">Prioritet</h2>
        <div class="flex gap-1">
          <Button
            v-for="option in severityOptions"
            :key="option.value"
            :label="option.label"
            :severity="severityTagSeverity(option.value)"
            :outlined="sosCase.severity !== option.value"
            size="small"
            class="flex-1 !py-1"
            @click="setSeverity(option.value)"
          />
        </div>

        <h2 class="text-xs uppercase tracking-wide text-gray-500 mt-4 mb-2">Tildelt</h2>
        <Select
          :modelValue="sosCase.assigneeSectionSlug"
          :options="assigneeOptions"
          optionLabel="label"
          optionValue="value"
          placeholder="Ingen"
          showClear
          size="small"
          class="w-full"
          @update:modelValue="setAssignee"
        />
        <small v-if="assigneeOptions.length === 0" class="block mt-2 text-gray-500">
          Ingen sektioner kan tildeles nødråb endnu. Slå det til på Organisation-siden.
        </small>
      </div>

      <SosTeamCard :sos-id="caseId" :teams="teams" @changed="refresh" />
    </aside>
  </div>

  <div v-else class="card">
    <ProgressSpinner v-if="pending" />
  </div>
</template>

<style>
#sos-detail .card {
  padding: 1rem;
}

/*
  The "note" surface: a flat, tinted, bordered card used for the case itself and for
  every comment. Defined here — in the view's unscoped block — rather than inside
  SosActivityLine, because the requirement is that the two look *the same*, and two
  copies of these four declarations would drift the first time one was adjusted.

  The page's own panels are white, so the tint is what stops a note dissolving into
  the panel behind it; flat rather than shadowed, because a run of raised cards down a
  timeline reads as floating boxes. Specificity beats PrimeVue's own .p-card rules
  without !important thanks to the id.
*/
#sos-detail .sos-note {
  background: #f4f6f8;
  border: 1px solid #e5e7eb;
  box-shadow: none;
}
#sos-detail .sos-note .p-card-body {
  /* Sized for both uses: roomy enough under the case's 3xl headline, still tight
     enough that a one-line comment does not float in space. */
  padding: 0.75rem 1rem;
}
#sos-detail .sos-note .p-card-content {
  padding: 0;
}

/* Quieter than the case itself, but readable without interaction — full opacity on
   hover or while a control inside has focus. */
#sos-detail .sos-aside {
  opacity: 0.75;
  transition: opacity 120ms ease;
}
#sos-detail .sos-aside:hover,
#sos-detail .sos-aside:focus-within {
  opacity: 1;
}

/* The time column only needs room for HH:MM; the default split wastes half the
   width on it. */
#sos-detail .sos-timeline .p-timeline-event-opposite {
  flex: 0 0 3.5rem;
  padding-top: 0.35rem;
  text-align: right;
}

/* Aura sizes every timeline event at `timeline.event.minHeight: 5rem`, which is the
   real source of the vertical spacing — the content padding is a rounding error next
   to it. Halved via the design token rather than by fighting it with padding, so the
   connector, marker and content all shorten together and stay aligned. */
#sos-detail .sos-timeline {
  --p-timeline-event-min-height: 2.5rem;
}
#sos-detail .sos-timeline .p-timeline-event-content {
  padding-bottom: 0.375rem;
}
</style>
