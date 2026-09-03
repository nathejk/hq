<script setup lang="ts">
// Hønsegården — the shelter crew's screen (PRD 007).
//
// Read-only in this task: the actions (modtaget, handover, placering) arrive with tasks
// 093–095. What is here is the overview the crew has never had — who is on their way, who is
// here and where, who is still out — and it updates itself, because a screen a volunteer
// leaves open all night must not quietly go stale.

import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { FilterMatchMode } from '@primevue/core/api'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useConnectionState } from '@/composables/useConnectionState'
import MemberDetailDialog from '@/components/MemberDetailDialog.vue'
import PositionIndicator from '@/components/PositionIndicator.vue'
import {
  useNow,
  formatTimestamp,
  minutesSince,
  WAITING_ALARM_MINUTES,
} from '@/composables/shelter'

const router = useRouter()
const toast = useToast()

interface TeamRef {
  teamId: string
  teamNumber: string
  name: string
}

interface ShelterMember {
  memberId: string
  name: string
  status: string
  updatedAt: string
  team: TeamRef | null
  startTeam?: TeamRef | null
  placement: string
  placedAt: string | null
  sosId?: string
  teamDiscontinued: boolean
  noteCount: number
  latestNote: string
  latestNoteAt: string | null
  // Carried in the payload but no longer shown (removed 2026-08-23 at the crew's request — the
  // nødtelefon does the ringing). Kept on the type because the API still sends them, and the member
  // dialog shows both numbers where they are actually useful: in front of somebody making the call.
  phone: string
  phoneParent: string
}

interface ShelterSection {
  slug: string
  label: string
  members: ShelterMember[]
}

// Live, cached, and the whole page in one entry.
//
// dependsOn is entity **types**, never instances. A scout who drops out has a member id this
// client has never seen — that is the whole point of the screen — so an instance-keyed
// dependency could never make them appear. `spejder` is the entity in the lifecycle subjects
// (NATHEJK.{year}.spejder.{id}.*): *not* `spejderstatus` or `shelter`, which are only the
// projections' names, and the SPA warns in the dev console about a token nothing can emit.
// `patrulje` is here because the rows carry patrol names and numbers.
const { data, pending, error, refresh } = useLiveResource(
  'shelter',
  async () => {
    const response = await http.get('/shelter')
    return response.data as {
      sections: ShelterSection[]
      counts: Record<string, number>
      care: { total: number; byStatus: Record<string, number>; oldestWaitingAt: string | null }
      memberStatuses: { slug: string; label: string }[]
      placements: { placement: string; count: number }[]
    }
  },
  { dependsOn: ['spejder', 'patrulje'] },
)

// --- Deferring updates while somebody is typing ---
//
// This screen is read-only except for one field, and that field is enough to make it an
// editor. If a revalidation landed while a crew member was typing a tent name, PrimeVue would
// re-render the table under them: the row can move between sections, the input loses focus
// mid-word, and the half-typed value is gone. At 3am, with a queue at the door, that is how an
// operator stops trusting the screen.
//
// So the payload is mirrored into `applied` and everything renders from there. Incoming
// payloads are held back while an editor is open and applied when it closes — the same shape as
// `KlanListView`, with one deliberate difference: the pause lasts only as long as a field is
// actually open, not until somebody presses a save button. Nothing else on this page is
// editable, so there is nothing else to protect and no reason to keep the whole night's updates
// waiting.
type ShelterPayload = NonNullable<typeof data.value>

const applied = ref<ShelterPayload | null>(null)

/**
 * The row being edited, and the draft value. One at a time: you can only type in one field.
 *
 * Declared here, above the watch that reads it, and not with the rest of the editor further
 * down — the watch below runs synchronously at setup (`immediate: true`), so a later `const`
 * would still be in its temporal dead zone and the whole view would fail to mount.
 */
const editing = ref<{ memberId: string; value: string } | null>(null)

/**
 * The open member dialog, and whether its note form holds unsaved text.
 *
 * Same temporal-dead-zone reason for living up here: `paused` is read by the watch below.
 */
const openMember = ref<{ memberId: string; teamId: string; name: string; status: string } | null>(null)
const notesDirty = ref(false)

/**
 * Whether incoming payloads are being held back.
 *
 * Two kinds of unsaved state qualify: a placering being typed in a row, and a note being written in
 * the dialog. Both would be destroyed by a re-render — the placering field is inside a table row that
 * can move between sections, and the dialog's panel rebuilds around its textarea — so both pause the
 * list. Nothing else on this screen is editable, so nothing else pauses it.
 */
const paused = computed(() => editing.value !== null || notesDirty.value)

/** A payload arrived while something was being edited and has not been shown yet. */
const deferred = ref(false)

watch(
  data,
  (payload) => {
    if (!payload) return
    if (paused.value) {
      deferred.value = true
      return
    }
    applied.value = payload
    deferred.value = false
  },
  { immediate: true },
)

/** Show whatever arrived while an edit was in progress. */
const applyDeferred = () => {
  if (!deferred.value) return
  if (data.value) applied.value = data.value
  deferred.value = false
}

// Closing the note form is one of the two ways a pause ends, and the host cannot know it happened
// except by watching this — the dialog reports the state, not the transition.
watch(paused, (still) => {
  if (!still) applyDeferred()
})

const sections = computed<ShelterSection[]>(() => applied.value?.sections ?? [])
const counts = computed<Record<string, number>>(() => applied.value?.counts ?? {})

// Status labels come from the server (PRD 006 §6). A label map here would be the second copy
// of the same Danish copy, and the two drift until one of them says "waiting" to a volunteer
// at 3am.
const statusLabels = computed<Record<string, string>>(() => {
  const labels: Record<string, string> = {}
  for (const s of applied.value?.memberStatuses ?? []) labels[s.slug] = s.label
  return labels
})
const statusLabel = (slug: string) => statusLabels.value[slug] ?? slug

// The ticking clock behind every duration on the screen — see the composable for why the
// passage of time has to be reactive here.
const now = useNow()

// A wrong number is worse than no number, and this is the number the organisers go home on.
// Same argument, and the same mechanism, as the nødtelefon's in-care badge.
const { isDisconnected } = useConnectionState()
const careUnavailable = computed(() => isDisconnected.value || !!error.value || !applied.value)

// Highlighted when a scout has been waiting too long. The threshold is a placeholder and lives
// in one place (task 082 settles it); only `waiting` is measured, because a scout in a car or
// asleep in a tent is accounted for.
const isOverdue = (member: ShelterMember) =>
  member.status === 'waiting' && minutesSince(member.updatedAt, now.value) >= WAITING_ALARM_MINUTES

// Placering, or a visible reminder that there is not one yet. An accepted scout with nowhere
// recorded is the crew's next job, so it must not render as an empty cell that looks like
// nothing to do.
const placementText = (member: ShelterMember) => member.placement || 'ikke placeret'

// --- The placering editor (PRD 007 §6, task 095) ---
//
// The zones are not known until race start, so there is nothing to configure and no picker to
// build from a list: the suggestions are the placeringer the crew has already typed tonight,
// most-used first, and free text is always accepted. The first scout into a tent is typed, every
// one after that is picked — which is what stops "Telt 4", "telt4" and "t4" becoming three
// places without anybody setting anything up.
//
// `editing` itself is declared further up, next to the deferral watch that reads it.

/** Suggestions currently offered, filtered as the crew types. */
const suggestions = ref<string[]>([])

const zones = computed<string[]>(() => (applied.value?.placements ?? []).map((p) => p.placement))

const isEditing = (member: ShelterMember) => editing.value?.memberId === member.memberId

const startEditing = (member: ShelterMember) => {
  editing.value = { memberId: member.memberId, value: member.placement }
  suggestions.value = zones.value
}

const cancelEditing = () => {
  editing.value = null
  applyDeferred()
}

// Ordered by use, not alphabetically, and unfiltered on an empty query: the tent four scouts are
// already in is the likeliest answer for the fifth, and it should be the first thing offered.
const completePlacement = (event: { query: string }) => {
  const query = event.query.trim().toLowerCase()
  suggestions.value = query
    ? zones.value.filter((zone) => zone.toLowerCase().includes(query))
    : zones.value
}

const savePlacement = async (member: ShelterMember) => {
  const draft = (editing.value?.value ?? '').trim()
  if (!draft) {
    // The server refuses this too, but saying so here saves a round trip and phrases it as the
    // choice it is: clearing a placering is not offered, because "nowhere" is not a fact about a
    // child in our care. If they moved, the answer is where to.
    toast.add({
      severity: 'warn',
      life: 5000,
      summary: 'Placering mangler',
      detail: 'Skriv hvor spejderen er — en tom placering kan ikke gemmes',
    })
    return
  }
  // Unchanged: close the editor and publish nothing. The server would answer 200 for this too
  // (the command dirty-checks), but a request that cannot change anything is not worth making.
  if (draft === member.placement) {
    cancelEditing()
    return
  }
  await run(member, 'Kunne ikke gemme placeringen', () =>
    http.put(`/member/${member.memberId}/placement`, { placement: draft }),
  )
  editing.value = null
  applyDeferred()
}

const filters = ref({ global: { value: null, matchMode: FilterMatchMode.CONTAINS } })

const openPatrulje = (member: ShelterMember) => {
  if (!member.team?.teamId) return
  router.push({ name: 'patrulje', params: { teamId: member.team.teamId } })
}

// The scout's name opens the shared member dialog (task 103), which carries the guardian's phone,
// address, history and the note trail. The name is the affordance on every host — case card,
// Hønsegården, patrol page — so the crew learns one gesture rather than three.
//
// teamId is required by the endpoint: a scout the lifecycle never touched has no status row, so the
// server cannot infer which roster to find them on. The row's current team is the right answer.
const showMember = (member: ShelterMember) => {
  openMember.value = {
    memberId: member.memberId,
    teamId: member.team?.teamId ?? '',
    name: member.name,
    status: member.status,
  }
}

const closeMember = () => {
  openMember.value = null
  // The note form went with the dialog, so any pause it caused ends here rather than waiting for a
  // dirty event that will never arrive from an unmounted component.
  notesDirty.value = false
}

// The snippet is already truncated by the server (120 runes), so this only strips newlines: a note
// written across three lines would otherwise stretch the row and push every other scout down.
const noteSnippet = (member: ShelterMember) => member.latestNote.replace(/\s+/g, ' ').trim()

const openCase = (member: ShelterMember) => {
  if (!member.sosId) return
  router.push({ name: 'sos', params: { id: member.sosId } })
}

watch(error, (err) => {
  if (!err) return
  toast.add({ severity: 'error', summary: 'Kunne ikke hente Hønsegården', life: 5000 })
})

// --- Actions (PRD 007, task 093) ---
//
// Which actions need two clicks, and which do not, is a judgement about cost rather than a
// uniform policy:
//
//   Modtaget from *I bil* is one click. It is the most frequent action of the night, done with
//   a queue of children in the doorway, and its mistake is cheap — the scout was arriving
//   anyway, and accepting them a minute early is not wrong for long.
//
//   Modtaget from *Afventer afhentning* is two. It asserts an arrival the platform has no
//   pickup for, so it is either a genuine gap in the record or the wrong row.
//
//   Both handovers are two, and this is the one I would defend hardest: they record that a
//   child left our care, there is no undo, and a mis-click marks a scout released while they
//   are asleep in a tent. That is the single worst thing this screen can do.
//
// Two clicks are an inline arm/confirm on the button itself rather than a modal. No
// ConfirmationService is registered in this app, and a dialog is the wrong shape here anyway:
// it steals focus, needs dismissing, and a volunteer holding a phone in one hand should not
// have to chase it.

/** `${memberId}:${action}`, or null. One armed button at a time, app-wide. */
const armed = ref<string | null>(null)

/** Members with a request in flight, so their buttons cannot be pressed twice. */
const busy = ref<Set<string>>(new Set())

const armKey = (member: ShelterMember, action: string) => `${member.memberId}:${action}`
const isArmed = (member: ShelterMember, action: string) => armed.value === armKey(member, action)
const isBusy = (member: ShelterMember) => busy.value.has(member.memberId)

/** Arming one button disarms any other, so two half-pressed actions cannot sit on screen. */
const arm = (member: ShelterMember, action: string) => {
  armed.value = armKey(member, action)
}
const disarm = () => {
  armed.value = null
}

/**
 * The server's Danish message, or a fallback.
 *
 * The API answers validation failures as `{error: {field: "message"}}` and other failures as
 * `{error: "message"}`. Those messages are written for the crew — "modtag dem først" names the
 * button that fixes the problem — so showing them beats any generic text this view could
 * invent.
 */
const errorMessage = (err: unknown, fallback: string): string => {
  const payload = (err as { response?: { data?: { error?: unknown } } })?.response?.data?.error
  if (typeof payload === 'string') return payload
  if (payload && typeof payload === 'object') {
    const first = Object.values(payload as Record<string, unknown>)[0]
    if (typeof first === 'string') return first
  }
  return fallback
}

/**
 * Run a write, then refetch.
 *
 * No local mutation of the cached payload: the write path and the read path must not be able to
 * disagree about what happened, and the projection is the only thing that knows. The explicit
 * `refresh()` is not belt-and-braces — it is what keeps the screen correct when the live stream
 * is degraded to polling or down altogether, and `useLiveResource` collapses it into the
 * signal-triggered revalidation when both happen at once.
 */
const run = async (member: ShelterMember, fallback: string, request: () => Promise<unknown>) => {
  disarm()
  busy.value = new Set(busy.value).add(member.memberId)
  try {
    await request()
    await refresh()
  } catch (err) {
    toast.add({
      severity: 'error',
      closable: true,
      life: 8000,
      summary: `${member.name}: handlingen blev afvist`,
      detail: errorMessage(err, fallback),
    })
  } finally {
    const next = new Set(busy.value)
    next.delete(member.memberId)
    busy.value = next
  }
}

// A no-op answers 200, so pressing Modtaget on a scout another crew member has just accepted
// produces no toast and no error — the screen simply already says what this operator wanted.
const accept = (member: ShelterMember) =>
  run(member, 'Kunne ikke modtage spejderen', () =>
    http.put(`/member/${member.memberId}/shelter`, {}),
  )

const handover = (member: ShelterMember, to: 'released' | 'reunited') =>
  run(member, 'Kunne ikke registrere afleveringen', () =>
    http.put(`/member/${member.memberId}/handover`, { to }),
  )

// Disabled with an explanation rather than hidden: a missing button reads as a bug, and the
// reason is worth telling — the patrol has nobody left on the route, so it will not reach the
// finish for anybody to be reunited at. Those scouts end at `released`.
const reuniteTooltip = (member: ShelterMember) =>
  member.teamDiscontinued
    ? 'Patruljen er udgået — der er ingen tilbage i løbet at blive genforenet med'
    : 'Patruljen er nået i mål og har taget spejderen tilbage'
</script>

<template>
  <div class="card" id="hoensegaard">
    <div class="flex flex-wrap gap-2 items-center justify-between mb-4">
      <h1 class="font-nathejk text-2xl">
        <i class="fas fa-kiwi-bird mr-2" aria-hidden="true" />Hønsegården
      </h1>

      <!--
        I vores varetægt — the same figure the nødtelefon shows, from the same query, because
        two independent counts of the children we are responsible for is one more than the
        night can afford.
      -->
      <div
        class="flex items-center gap-2 rounded border px-3 py-1"
        :class="careUnavailable ? 'border-gray-300 bg-gray-50' : 'border-amber-300 bg-amber-50'"
      >
        <span class="text-sm text-gray-700">I vores varetægt</span>
        <span v-if="careUnavailable" class="text-sm italic text-gray-500">ingen forbindelse</span>
        <span v-else class="font-nathejk text-xl">{{ applied?.care?.total ?? 0 }}</span>
      </div>

      <IconField>
        <InputIcon><i class="pi pi-search" /></InputIcon>
        <InputText v-model="filters['global'].value" placeholder="Søg efter navn..." />
      </IconField>
    </div>

    <!--
      Said on screen, not just implied: while a placering is being typed the table is frozen, and
      an operator who knows a car just arrived deserves to know why the list has not changed yet.
      It disappears the moment the field closes.
    -->
    <div
      v-if="paused"
      class="mb-3 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800"
    >
      <i class="pi pi-pause mr-2" aria-hidden="true" />
      {{ editing ? 'Skriver placering' : 'Skriver note' }} — opdateringer fra andre er sat på pause
      indtil du gemmer eller fortryder.
      <span v-if="deferred" class="font-semibold">Der er nye ændringer, som vises når du er færdig.</span>
    </div>

    <!--
      One table per section, in the order the server sends them: the arrivals queue first,
      because it is the only section with somebody standing in front of it. The order and the
      Danish labels are the server's (PRD 007 §8), so this template does not hardcode either.
    -->
    <template v-for="section in sections" :key="section.slug">
      <h2 class="font-nathejk text-xl mt-6 mb-1">
        {{ section.label }} ({{ counts[section.slug] ?? 0 }})
      </h2>

      <!--
        An empty "På vej" is one quiet line, not an empty table: it is empty most of the night,
        and a table's worth of chrome would push the sections that matter off the screen.
      -->
      <p v-if="section.slug === 'onway' && !section.members.length" class="text-gray-500 italic">
        Ingen på vej
      </p>

      <DataTable
        v-else
        :value="section.members"
        :loading="pending"
        :filters="filters"
        :globalFilterFields="['name', 'team.name', 'team.teamNumber', 'placement']"
        :stripedRows="true"
        :rowClass="(row) => (isOverdue(row) ? '!bg-red-50' : '')"
        dataKey="memberId"
        size="small"
      >
        <template #empty>Ingen spejdere her</template>

        <Column field="name" header="Navn" sortable>
          <template #body="{ data: member }">
            <!--
              The name is the way in to everything about this scout: guardian's phone, address,
              history and the note trail. A button rather than a row click, so sorting a column or
              editing a placering cannot open a dialog by accident.
            -->
            <span class="inline-flex items-center gap-1">
              <Button
                :label="member.name"
                link
                class="!p-0 !font-medium"
                v-tooltip.top="'Vis oplysninger, historik og noter'"
                @click="showMember(member)"
              />
              <PositionIndicator :person-id="member.memberId" />
            </span>
          </template>
        </Column>

        <Column field="team.name" header="Patrulje" sortable>
          <template #body="{ data: member }">
            <Button
              v-if="member.team"
              :label="
                member.team.teamNumber
                  ? `${member.team.teamNumber} ${member.team.name}`
                  : member.team.name || member.team.teamId
              "
              link
              size="small"
              class="!p-0"
              @click="openPatrulje(member)"
            />
            <!--
              Only for a scout moved to another patrol before dropping out — the server omits it
              otherwise, so its presence is the fact itself. It sits under the patrol rather than
              under the name because it is a statement about which patrol, not about who.
            -->
            <span v-if="member.startTeam" class="block text-xs text-gray-500">
              startede i {{ member.startTeam.name || member.startTeam.teamId }}
            </span>
          </template>
        </Column>

        <!-- Dropped in *I Hønsegården*: every row there is `sheltered`, so the column would be
             one repeated word. -->
        <Column v-if="section.slug !== 'sheltered'" field="status" header="Status" sortable>
          <template #body="{ data: member }">{{ statusLabel(member.status) }}</template>
        </Column>

        <!--
          Both the clock time and the elapsed span: the first is what gets read out over the radio,
          the second is what triggers a decision. No "siden" in the cell — the header says it.
        -->
        <Column
          field="updatedAt"
          :header="section.slug === 'sheltered' ? 'Ankommet' : 'Siden'"
          sortable
        >
          <template #body="{ data: member }">
            <span :class="isOverdue(member) ? 'font-semibold text-red-700' : ''">
              {{ formatTimestamp(member.updatedAt, now) }}
            </span>
          </template>
        </Column>

        <Column v-if="section.slug === 'sheltered'" field="placement" header="Placering" sortable>
          <template #body="{ data: member }">
            <!--
              An editable combobox: suggestions are the placeringer already in use tonight, and
              anything typed is accepted. There is no zone list to pick from because the zones do
              not exist until race start — they define themselves as the night goes on.
            -->
            <div v-if="isEditing(member)" class="flex items-center gap-1">
              <AutoComplete
                v-model="editing!.value"
                :suggestions="suggestions"
                dropdown
                :maxlength="64"
                size="small"
                inputClass="!w-40"
                placeholder="fx Telt 4"
                autofocus
                @complete="completePlacement"
                @keyup.enter="savePlacement(member)"
                @keyup.escape="cancelEditing()"
              />
              <Button
                icon="pi pi-check"
                size="small"
                :loading="isBusy(member)"
                v-tooltip.top="'Gem placering (Enter)'"
                @click="savePlacement(member)"
              />
              <Button
                icon="pi pi-times"
                size="small"
                text
                severity="secondary"
                v-tooltip.top="'Fortryd (Esc)'"
                @click="cancelEditing()"
              />
            </div>

            <!--
              Not editing: the placering is the button. A scout with nowhere recorded shows in
              amber, because that is the crew's next job rather than an empty cell.
            -->
            <Button
              v-else
              :label="placementText(member)"
              :icon="member.placement ? 'pi pi-pencil' : 'pi pi-map-marker'"
              iconPos="right"
              link
              size="small"
              class="!p-0"
              :class="member.placement ? '' : '!text-amber-700 italic'"
              v-tooltip.top="'Sæt eller ret placering'"
              @click="startEditing(member)"
            />
          </template>
        </Column>

        <!--
          Noter: the count and the newest line, so the one scout with instructions is visible while
          scanning rather than found by opening forty dialogs. This is the deliberate answer to
          "expandable rows" as a host for the trail (PRD 008 §7) — the context without a second
          editing UI.
        -->
        <Column header="Noter">
          <template #body="{ data: member }">
            <Button
              v-if="member.noteCount"
              link
              class="!p-0 !text-left"
              v-tooltip.top="member.latestNote"
              @click="showMember(member)"
            >
              <span class="flex items-start gap-2">
                <Badge :value="member.noteCount" severity="info" />
                <span class="max-w-56 truncate text-xs text-gray-600">{{ noteSnippet(member) }}</span>
              </span>
            </Button>
            <!--
              No notes is not nothing to say: on a sheltered scout it means nobody has written down
              what was agreed, which is worth a nudge rather than an empty cell.
            -->
            <Button
              v-else
              label="Tilføj"
              icon="pi pi-plus"
              link
              size="small"
              class="!p-0 !text-xs"
              severity="secondary"
              v-tooltip.top="'Skriv hvad der er aftalt'"
              @click="showMember(member)"
            />
          </template>
        </Column>

        <!--
          The case link belongs to *På vej* only. A scout who is here, or has been handed on, is
          no longer something the nødtelefon's case is about — and the crew reaches the case from
          the patrol page if they need it.
        -->
        <Column v-if="section.slug === 'onway'" header="Sag">
          <template #body="{ data: member }">
            <Button
              v-if="member.sosId"
              icon="fas fa-headset"
              label="Sag"
              link
              size="small"
              class="!p-0"
              @click="openCase(member)"
            />
          </template>
        </Column>

        <!--
          Actions in the row, not behind a menu: a tired volunteer should not have to discover
          them. Keyed on the **member's status**, not the section — "På vej" holds both scouts in
          a car and scouts by a road, and the two are received differently even though the crew
          reads them as one list.
        -->
        <Column v-if="section.slug !== 'closed'" header="Handling">
          <template #body="{ data: member }">
            <div class="flex flex-wrap gap-2">
              <!-- Out of a car: one click. Frequent, and its mistake is cheap. -->
              <Button
                v-if="member.status === 'transit'"
                label="Modtaget"
                icon="pi pi-check"
                size="small"
                :loading="isBusy(member)"
                @click="accept(member)"
              />

              <!--
                Still on the trail as far as the platform knows: two clicks, because this
                asserts an arrival no pickup was ever recorded for — either a real gap in the
                record or the wrong row.
              -->
              <template v-else-if="member.status === 'waiting'">
                <Button
                  v-if="!isArmed(member, 'accept')"
                  label="Modtaget"
                  icon="pi pi-check"
                  size="small"
                  severity="secondary"
                  outlined
                  :loading="isBusy(member)"
                  v-tooltip.top="'Ingen bil er registreret for denne spejder — bekræft at de står i Hønsegården'"
                  @click="arm(member, 'accept')"
                />
                <template v-else>
                  <Button
                    label="Bekræft modtaget"
                    icon="pi pi-check"
                    size="small"
                    severity="warn"
                    :loading="isBusy(member)"
                    @click="accept(member)"
                  />
                  <Button label="Fortryd" size="small" text @click="disarm()" />
                </template>
              </template>

              <!--
                Both handovers are two clicks. They record that a child left our care, there is
                no undo, and a mis-click marks a scout released while they are asleep in a tent.
              -->
              <template v-else-if="member.status === 'sheltered'">
                <template v-if="isArmed(member, 'released')">
                  <Button
                    label="Bekræft: hentet af forældre"
                    icon="pi pi-check"
                    size="small"
                    severity="warn"
                    :loading="isBusy(member)"
                    @click="handover(member, 'released')"
                  />
                  <Button label="Fortryd" size="small" text @click="disarm()" />
                </template>
                <template v-else-if="isArmed(member, 'reunited')">
                  <Button
                    label="Bekræft: genforenet"
                    icon="pi pi-check"
                    size="small"
                    severity="warn"
                    :loading="isBusy(member)"
                    @click="handover(member, 'reunited')"
                  />
                  <Button label="Fortryd" size="small" text @click="disarm()" />
                </template>
                <template v-else>
                  <Button
                    label="Hentet af forældre"
                    icon="pi pi-home"
                    size="small"
                    outlined
                    :loading="isBusy(member)"
                    @click="arm(member, 'released')"
                  />
                  <Button
                    label="Genforenet"
                    icon="pi pi-users"
                    size="small"
                    outlined
                    severity="secondary"
                    :disabled="member.teamDiscontinued"
                    :loading="isBusy(member)"
                    v-tooltip.top="reuniteTooltip(member)"
                    @click="arm(member, 'reunited')"
                  />
                </template>
              </template>
            </div>
          </template>
        </Column>
      </DataTable>
    </template>

    <!--
      One scout in full, shared with the nødtelefon's case card and the patrol page (task 103). No
      actions slot: the shelter's own actions live in the row, where the crew is already looking, and
      duplicating them inside the dialog would give two places to press for one thing.

      `dirty` pauses this list while a note is being written — see `paused` above.
    -->
    <MemberDetailDialog
      v-if="openMember"
      :member-id="openMember.memberId"
      :team-id="openMember.teamId"
      :status="openMember.status"
      :name="openMember.name"
      @close="closeMember"
      @dirty="notesDirty = $event"
    />
  </div>
</template>

<style>
/* Roomier rows than the default: this is read at arm's length, at night, by somebody holding
   a phone in the other hand. */
#hoensegaard td {
  padding: 0.5rem 0.75rem;
}
</style>
