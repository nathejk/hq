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
import {
  useNow,
  formatSince,
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
  phone: string
  phoneParent: string
  team: TeamRef | null
  startTeam?: TeamRef | null
  placement: string
  placedAt: string | null
  sosId?: string
  teamDiscontinued: boolean
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

const sections = computed<ShelterSection[]>(() => data.value?.sections ?? [])
const counts = computed<Record<string, number>>(() => data.value?.counts ?? {})

// Status labels come from the server (PRD 006 §6). A label map here would be the second copy
// of the same Danish copy, and the two drift until one of them says "waiting" to a volunteer
// at 3am.
const statusLabels = computed<Record<string, string>>(() => {
  const labels: Record<string, string> = {}
  for (const s of data.value?.memberStatuses ?? []) labels[s.slug] = s.label
  return labels
})
const statusLabel = (slug: string) => statusLabels.value[slug] ?? slug

// The ticking clock behind every duration on the screen — see the composable for why the
// passage of time has to be reactive here.
const now = useNow()

// A wrong number is worse than no number, and this is the number the organisers go home on.
// Same argument, and the same mechanism, as the nødtelefon's in-care badge.
const { isDisconnected } = useConnectionState()
const careUnavailable = computed(() => isDisconnected.value || !!error.value || !data.value)

// Highlighted when a scout has been waiting too long. The threshold is a placeholder and lives
// in one place (task 082 settles it); only `waiting` is measured, because a scout in a car or
// asleep in a tent is accounted for.
const isOverdue = (member: ShelterMember) =>
  member.status === 'waiting' && minutesSince(member.updatedAt, now.value) >= WAITING_ALARM_MINUTES

// Placering, or a visible reminder that there is not one yet. An accepted scout with nowhere
// recorded is the crew's next job, so it must not render as an empty cell that looks like
// nothing to do.
const placementText = (member: ShelterMember) => member.placement || 'ikke placeret'

const filters = ref({ global: { value: null, matchMode: FilterMatchMode.CONTAINS } })

const openPatrulje = (member: ShelterMember) => {
  if (!member.team?.teamId) return
  router.push({ name: 'patrulje', params: { teamId: member.team.teamId } })
}

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
        <span v-else class="font-nathejk text-xl">{{ data?.care?.total ?? 0 }}</span>
      </div>

      <IconField>
        <InputIcon><i class="pi pi-search" /></InputIcon>
        <InputText v-model="filters['global'].value" placeholder="Søg efter navn..." />
      </IconField>
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
        :stripedRows="true"
        :rowClass="(row) => (isOverdue(row) ? '!bg-red-50' : '')"
        dataKey="memberId"
        size="small"
      >
        <template #empty>Ingen spejdere her</template>

        <Column field="name" header="Navn" sortable>
          <template #body="{ data: member }">
            <span class="font-medium">{{ member.name }}</span>
            <!--
              Only for a scout moved to another patrol before dropping out — the server omits
              it otherwise, so its presence is the fact itself.
            -->
            <span v-if="member.startTeam" class="block text-xs text-gray-500">
              startede i {{ member.startTeam.name || member.startTeam.teamId }}
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
          </template>
        </Column>

        <Column field="status" header="Status" sortable>
          <template #body="{ data: member }">{{ statusLabel(member.status) }}</template>
        </Column>

        <!--
          Both the clock time and the elapsed span: the first is what gets written on paper,
          the second is what triggers a decision.
        -->
        <Column field="updatedAt" header="Siden" sortable>
          <template #body="{ data: member }">
            <span :class="isOverdue(member) ? 'font-semibold text-red-700' : ''">
              {{ formatSince(member.updatedAt, now) }}
            </span>
          </template>
        </Column>

        <Column v-if="section.slug === 'sheltered'" field="placement" header="Placering" sortable>
          <template #body="{ data: member }">
            <span :class="member.placement ? '' : 'italic text-amber-700'">
              {{ placementText(member) }}
            </span>
          </template>
        </Column>

        <!-- What the crew actually dials. The rest of the scout's details are one click away. -->
        <Column header="Telefon">
          <template #body="{ data: member }">
            <a v-if="member.phone" :href="`tel:${member.phone}`" class="block">{{ member.phone }}</a>
            <a
              v-if="member.phoneParent"
              :href="`tel:${member.phoneParent}`"
              class="block text-sm text-gray-600"
              >{{ member.phoneParent }} (forælder)</a
            >
          </template>
        </Column>

        <Column header="Sag">
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
  </div>
</template>

<style>
/* Roomier rows than the default: this is read at arm's length, at night, by somebody holding
   a phone in the other hand. */
#hoensegaard td {
  padding: 0.5rem 0.75rem;
}
</style>
