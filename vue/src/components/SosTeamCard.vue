<script setup lang="ts">
import { computed, ref } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { hhmm } from '@/composables/datefilters'

// The patrols associated with a case, and their members.
//
// PRD 001 shipped this card with identity and contact only, deliberately leaving room
// here: a list of names with nothing next to them reads as a broken feature. PRD 006
// fills that room with the member rows *and* the actions that give them a purpose.
//
// What is deliberately **not** here: the status override. That is a correction, not part
// of the call an operator is on, and it lives on the patrol page (task 084). Being a
// different screen is a stronger separation than a differently-styled button would be.
const props = defineProps<{
  sosId: string
  statuses: { slug: string; label: string }[]
  teams: {
    teamId: string
    teamNumber: string
    name: string
    group: string
    korps: string
    contactName: string
    contactPhone: string
    activeMemberCount: number
    minMemberCount: number
    started: boolean
    members: {
      memberId: string
      name: string
      phone: string
      phoneParent: string
      status: string
      updatedAt: string | null
    }[]
  }[]
}>()

const emit = defineEmits<{ changed: [] }>()

// The picker filters the year's patrol list that the SPA already holds live for
// PatruljeListView — same cache key, so opening a case costs no extra request and
// the list cannot be stale in one place and fresh in the other. No search endpoint.
const { data: patruljeData } = useLiveResource(
  'patrulje:list',
  async () => {
    const response = await http.get('/patrulje')
    return response.data.teams.filter((p: { name: string }) => p.name != '')
  },
  { dependsOn: ['patrulje'] },
)

const query = ref('')
const associating = ref(false)

const associatedIds = computed(() => new Set(props.teams.map((t) => t.teamId)))

// Number, name and group, because a caller reads out their number and an operator
// hears a group name. Capped at ten: this is a mid-call lookup, not a browse.
const candidates = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (q.length < 2) return []
  return (patruljeData.value ?? [])
    .filter((p: { teamId: string }) => !associatedIds.value.has(p.teamId))
    .filter((p: { teamNumber?: string; name?: string; group?: string }) =>
      [p.teamNumber, p.name, p.group].some((field) => (field ?? '').toLowerCase().includes(q)),
    )
    .slice(0, 10)
})

const associate = async (teamId: string) => {
  associating.value = true
  try {
    await http.put(`/sos/${props.sosId}/team/${teamId}`)
    query.value = ''
    emit('changed')
  } catch {
    // surfaced by the axios plugin
  } finally {
    associating.value = false
  }
}

const disassociate = async (teamId: string) => {
  try {
    await http.delete(`/sos/${props.sosId}/team/${teamId}`)
    emit('changed')
  } catch {
    // surfaced by the axios plugin
  }
}

// --- member rows (PRD 006) ---

// Labels come from the backend, so this view holds no map of its own. Two screens show
// these strings and they are persisted values; a copy in each is how they drift until
// one says "waiting" to an operator at 3am.
const statusLabel = (slug: string) =>
  props.statuses.find((s) => s.slug === slug)?.label ?? (slug || 'Ikke startet')

const statusSeverity = (slug: string) => {
  switch (slug) {
    case 'racing':
      return 'success'
    case 'waiting':
      return 'danger' // the only state that blocks a whole patrol
    case 'transit':
    case 'sheltered':
      return 'warn'
    case 'reunited':
    case 'released':
      return 'secondary'
    default:
      return 'contrast'
  }
}

// A member is self-carrying up to and including `waiting`: they have covered every
// metre on their own legs. Those are the transitions this screen owns. From `transit`
// onwards the row is read-only, because the car and shelter interfaces record what
// happens next and this card must not pretend to.
const canWithdraw = (status: string) => status === 'racing'
const canResume = (status: string) => status === 'waiting'

// Pending per member rather than one flag for the card: an operator mid-call may act on
// two members in quick succession, and a single spinner would lock the second row while
// the first was in flight.
const pending = ref<Record<string, boolean>>({})

const act = async (memberId: string, path: 'waiting' | 'racing') => {
  pending.value = { ...pending.value, [memberId]: true }
  try {
    await http.put(`/member/${memberId}/${path}`, { sosId: props.sosId })
    emit('changed')
  } catch {
    // surfaced by the axios plugin; the row stays as it was, which is the honest
    // outcome for the resume the server may legitimately reject ("allerede hentet")
  } finally {
    pending.value = { ...pending.value, [memberId]: false }
  }
}

// Strength and discontinuation are the same number, read two ways (PRD 006 §11).
const belowStrength = (team: { started: boolean; activeMemberCount: number; minMemberCount: number }) =>
  team.started && team.activeMemberCount < team.minMemberCount && team.activeMemberCount > 0

// **The "started" half is not optional.** A team that never started also has zero racing
// members, so the count alone conflates *left the route* with *never on it* — without
// this, every patrol of a year that has not raced yet would be badged Udgået, which on
// the dev data is all 310 of them.
const discontinued = (team: { started: boolean; activeMemberCount: number }) =>
  team.started && team.activeMemberCount === 0

const since = (value: string | null) => (value ? hhmm(value) : '')
</script>

<template>
  <div class="card">
    <h2 class="font-nathejk text-xl mb-3">Tilknyttede patruljer</h2>

    <div v-for="team in teams" :key="team.teamId" class="border-b border-gray-200 py-2 last:border-0">
      <div class="flex items-start justify-between gap-2">
        <div>
          <router-link :to="{ name: 'patrulje', params: { teamId: team.teamId } }" class="font-semibold">
            <span v-if="team.teamNumber">{{ team.teamNumber }} — </span>{{ team.name || team.teamId }}
          </router-link>
          <!--
            Strength beside the name, because it is the number that decides whether the
            conversation the operator is having changes. Only shown once the patrol has
            started: before that it is 0 and means nothing.
          -->
          <span v-if="team.started" class="ml-2 text-sm text-gray-600">
            {{ team.activeMemberCount }}/{{ team.minMemberCount }} i løbet
          </span>
          <Tag v-if="discontinued(team)" value="Udgået" severity="contrast" class="ml-2" />
          <Tag v-else-if="belowStrength(team)" value="Under styrke" severity="danger" class="ml-2" />
          <div class="text-sm text-gray-500">
            {{ team.group }}<span v-if="team.korps"> · {{ team.korps }}</span>
          </div>
          <div v-if="team.contactPhone" class="text-sm">
            {{ team.contactName }}
            <!-- tel: so an operator on a phone taps to call rather than copying digits -->
            <a :href="`tel:${team.contactPhone}`" class="underline">{{ team.contactPhone }}</a>
          </div>
        </div>
        <Button icon="pi pi-times" text rounded size="small" severity="secondary"
                @click="disassociate(team.teamId)" />
      </div>

      <!--
        Member rows. Each shows where the member is and offers only the transitions this
        screen owns — from `transit` onwards the row is deliberately read-only, because
        the car and shelter interfaces record those and this card must not pretend to.
      -->
      <div v-for="member in team.members" :key="member.memberId"
           class="flex items-center justify-between gap-2 py-1 pl-3 text-sm">
        <div class="min-w-0">
          <span class="font-medium">{{ member.name || member.memberId }}</span>
          <a v-if="member.phone" :href="`tel:${member.phone}`" class="ml-2 underline">{{ member.phone }}</a>
          <a v-if="member.phoneParent" :href="`tel:${member.phoneParent}`" class="ml-2 underline text-gray-500">
            {{ member.phoneParent }}
          </a>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <Tag :value="statusLabel(member.status)" :severity="statusSeverity(member.status)" />
          <span v-if="member.updatedAt && member.status" class="text-gray-500">
            {{ since(member.updatedAt) }}
          </span>
          <Button v-if="canWithdraw(member.status)" label="Ønsker at udgå" size="small" severity="danger"
                  outlined :loading="pending[member.memberId]"
                  @click="act(member.memberId, 'waiting')" />
          <!--
            Prominent rather than tucked into a menu: a scout getting their breath back
            is an ordinary outcome and saves a car being sent. Not optimistic — the
            server may legitimately reject it if a car has already collected them, so it
            shows as pending and lets the server answer.
          -->
          <Button v-if="canResume(member.status)" label="Fortsætter selv" size="small" severity="success"
                  :loading="pending[member.memberId]"
                  @click="act(member.memberId, 'racing')" />
        </div>
      </div>
      <div v-if="team.members.length === 0" class="pl-3 py-1 text-sm text-gray-500">
        Ingen deltagere registreret.
      </div>
    </div>

    <div v-if="teams.length === 0" class="text-sm text-gray-500 mb-2">
      Ingen patruljer tilknyttet.
    </div>

    <div class="mt-3">
      <InputText v-model="query" class="w-full" placeholder="Søg patrulje (nummer, navn, gruppe)" />
      <div v-for="candidate in candidates" :key="candidate.teamId"
           class="flex items-center justify-between gap-2 py-1">
        <span class="text-sm">
          <span v-if="candidate.teamNumber">{{ candidate.teamNumber }} — </span>{{ candidate.name }}
          <span class="text-gray-500">{{ candidate.group }}</span>
        </span>
        <Button label="Tilknyt" size="small" :loading="associating"
                @click="associate(candidate.teamId)" />
      </div>
      <small v-if="query.trim().length === 1" class="text-gray-500">Skriv mindst to tegn...</small>
    </div>
  </div>
</template>
