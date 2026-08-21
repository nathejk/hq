<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { hhmm, parseApiDate } from '@/composables/datefilters'
import { memberEventPhrase, formatDateTime, memberStatusBadge } from '@/composables/sos'

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
      status: string
      updatedAt: string | null
      movedAway?: boolean
      movedIn?: boolean
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

// Label, colour and glyph all come from the one shared badge vocabulary, so a member row's
// icon and the modal's Tag cannot disagree about the same status.
//
// A member row shows the status as an icon alone. Twelve rows of coloured word-tags read as
// a wall of text when what an operator scans for is "is anybody not racing?", and the
// colour already answers that; the glyph is what disambiguates two states sharing a colour
// (Udgår and Transit are both amber). The label appears in the modal, on the one member
// being looked at.
const statusLabel = (slug: string) => memberStatusBadge(slug).label
const statusSeverity = (slug: string) => memberStatusBadge(slug).severity
const statusIcon = (slug: string) => memberStatusBadge(slug).icon

const statusColour = (slug: string) => {
  switch (statusSeverity(slug)) {
    case 'success':
      return 'text-green-600'
    case 'danger':
      return 'text-red-600'
    case 'warn':
      return 'text-amber-600'
    case 'info':
      return 'text-blue-500'
    case 'secondary':
      return 'text-gray-500'
    default:
      return 'text-gray-400'
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

// --- member detail modal ---
//
// Opened by clicking a row. Its data comes from its own endpoint rather than the case
// payload: a case with three patrols has eighteen members, and carrying each one's address,
// birthday and full lifecycle would make the screen an operator stares at all night pay for
// detail they open one member at a time.
//
// It is still **live**, without a second live resource: the open member's row comes from
// `props.teams`, which the case resource keeps current, so watching that row's status is
// enough to know when the history has changed and reload it. That reuses the live channel
// already in place instead of introducing a resource whose key would have to vary.
// A patrol as the member modal names it: enough to identify and link to it. Deliberately
// not the card's team row, which also carries strength and contact details — the member's
// current patrol may not be on this case at all, so those are not available for it.
interface TeamRef {
  teamId: string
  teamNumber: string
  name: string
}

const detail = ref<{ memberId: string; team: TeamRow } | null>(null)
const detailData = ref<{
  member: {
    name: string
    phone: string
    phoneParent: string
    address: string
    postalCode: string
    city: string
    birthday: string | null
    status: string
  }
  startTeam: TeamRef | null
  currentTeam: TeamRef | null
  history: { seq: number; status: string; event: string; createdAt: string }[]
} | null>(null)
const detailPending = ref(false)

// The open member as the *card* sees them — live, because props.teams is. This is what the
// modal's status badge and action buttons read, so they cannot disagree with the row behind
// the modal.
const detailMember = computed<MemberRow | null>(() => {
  const d = detail.value
  if (!d) return null
  const team = props.teams.find((t) => t.teamId === d.team.teamId)
  return team?.members.find((m) => m.memberId === d.memberId) ?? null
})

const loadDetail = async () => {
  const d = detail.value
  if (!d) return
  detailPending.value = true
  try {
    const response = await http.get(`/member/${d.memberId}`, {
      // The member's team, so a member with no status row yet can still be found on a
      // roster — the endpoint cannot infer it for somebody the lifecycle has not touched.
      params: { teamId: d.team.teamId },
    })
    detailData.value = response.data
  } catch {
    detailData.value = null
    // surfaced by the axios plugin
  } finally {
    detailPending.value = false
  }
}

const openMember = (member: MemberRow, team: TeamRow) => {
  detail.value = { memberId: member.memberId, team }
  detailData.value = null
  void loadDetail()
}

const closeMember = () => {
  detail.value = null
  detailData.value = null
}

// A status change — whether this operator's or a colleague's — adds a line to the history,
// so reload when the live row moves.
watch(
  () => detailMember.value?.status,
  (next, previous) => {
    if (detail.value && next !== undefined && previous !== undefined && next !== previous) {
      void loadDetail()
    }
  },
)

// Birthday is a date, not a moment. The API serves it as an instant, so formatting it as a
// datetime would show the day before for anybody born in the evening — a scout born on the
// 5th at 23:00 UTC is a 5th-of-December birthday in Copenhagen, not a 4th.
const birthday = (value: string | null) => {
  const date = parseApiDate(value)
  return date ? date.toLocaleDateString('da-DK', { day: 'numeric', month: 'long', year: 'numeric' }) : ''
}

// --- the 3-member requirement (PRD 006, task 077) ---
//
// Nothing here enforces anything. The requirement is displayed, an operator applies it
// or judges an exception warranted, and the tool records what happened — there is
// deliberately no exception object, no approval and no "handled" flag to clear
// (PRD 006 §11). That is why the warning has no dismiss button: it states a fact for as
// long as the fact holds.

type TeamRow = (typeof props.teams)[number]
type MemberRow = TeamRow['members'][number]

// The members a below-strength action operates on: those actually racing **for this team**.
// A member who moved away is racing for somebody else, so collecting or moving "the rest of
// this patrol" must not reach them — and the server would refuse anyway, since the origin it
// derives comes from the member's own row.
const racingMembers = (team: TeamRow) =>
  team.members.filter((m) => m.status === 'racing' && !m.movedAway)

// "Active" has one meaning throughout this domain: racing. It is what `activeMemberCount`
// counts and what team strength is measured in, so a member who is waiting, in a car, at HQ,
// reunited, released — or who never started — is not active.
//
// A member who moved to another patrol is not active *here* either, even though their status
// is `racing`: they are racing for somebody else, and this team's strength does not count
// them. Dimming them keeps the list consistent with the number beside the team name.
//
// Kept separate from canWithdraw even though the conditions overlap, because they are
// different questions: one is a domain fact about the member, the other is a permission this
// screen grants. Collapsing them would tie the appearance of the list to what the buttons
// happen to allow.
const isActive = (member: MemberRow) => member.status === 'racing' && !member.movedAway

// Dimmed rather than hidden: a member who has left the race is still the patrol's member and
// still the person an operator may be asked about. Grey says "not counting towards strength"
// while the status icon keeps its own colour, which says something different — a waiting
// member is out of the race *and* urgent, and those two facts should not fight over one
// visual channel.
//
// gray-500 rather than a lighter grey deliberately: these are names read off a screen at
// three in the morning, and gray-400 on white falls below the contrast a small font needs.
const nameClass = (member: MemberRow) => (isActive(member) ? '' : 'text-gray-500')

// Pre-commit warning. Confirming a withdrawal that takes the team below the minimum
// should change the conversation the operator is having on the phone *before* it is
// recorded — but it must never block: the member is leaving whether or not the patrol
// is compliant.
const confirm = ref<{ member: MemberRow; team: TeamRow; resulting: number } | null>(null)

const askWithdraw = (member: MemberRow, team: TeamRow) => {
  const resulting = team.activeMemberCount - 1
  if (team.started && resulting < team.minMemberCount) {
    confirm.value = { member, team, resulting }
    return
  }
  act(member.memberId, 'waiting')
}

const proceedWithdraw = () => {
  const c = confirm.value
  confirm.value = null
  if (c) act(c.member.memberId, 'waiting')
}

// Collecting the whole patrol is one request, not one per member: three separate calls
// could half-succeed and leave the team split across two states with nobody noticing.
const collecting = ref<Record<string, boolean>>({})

const collect = async (team: TeamRow) => {
  confirm.value = null
  collecting.value = { ...collecting.value, [team.teamId]: true }
  try {
    await http.post(`/sos/${props.sosId}/team/${team.teamId}/collect`)
    emit('changed')
  } catch {
    // surfaced by the axios plugin
  } finally {
    collecting.value = { ...collecting.value, [team.teamId]: false }
  }
}

// --- moving the remaining members ---
//
// The dialog snapshots the members it offers when it opens. That is the dirty guard: the
// operator is choosing from a list, and having rows appear or vanish underneath them
// mid-choice is how the wrong scout gets moved. Incoming updates still arrive for the
// rest of the card — only this working set is frozen, which is the part that would be
// lost.
const moveDlg = ref<{
  team: TeamRow
  offered: MemberRow[]
  selected: Set<string>
  target: TeamRef | null
  query: string
  submitting: boolean
} | null>(null)

const openMove = (team: TeamRow) => {
  confirm.value = null
  const offered = racingMembers(team)
  moveDlg.value = {
    team,
    offered,
    // All of them by default: survivors usually stay together. Deselecting is how the
    // rarer split is expressed, rather than making the common case laborious.
    selected: new Set(offered.map((m) => m.memberId)),
    target: null,
    query: '',
    submitting: false,
  }
}

const toggleMember = (memberId: string) => {
  const d = moveDlg.value
  if (!d) return
  const next = new Set(d.selected)
  if (next.has(memberId)) next.delete(memberId)
  else next.add(memberId)
  moveDlg.value = { ...d, selected: next }
}

// Any patrol in the same year that is still racing is a valid destination — no
// proximity, liga or size rule, because crew in the field agree the destination and the
// operator is recording it, not choosing it (PRD 006 §11). So there is no candidate
// endpoint: this filters the patrol list the SPA already holds live, exactly as the
// association picker above does.
//
// Shared by the bulk move and the single member's "Skift", so the two cannot come to offer
// different destinations for the same decision.
const destinations = (query: string, excludeTeamId: string) => {
  const q = query.trim().toLowerCase()
  if (q.length < 2) return []
  return (patruljeData.value ?? [])
    .filter((p: { teamId: string; signupStatus?: string; activeMemberCount?: number }) =>
      // Still racing: a team that never started has nobody on the route to join, and one
      // that has been emptied is itself discontinued. Moving somebody into either would
      // be recording a fiction.
      p.teamId !== excludeTeamId && p.signupStatus === 'STARTED' && (p.activeMemberCount ?? 0) > 0,
    )
    .filter((p: { teamNumber?: string; name?: string; group?: string }) =>
      [p.teamNumber, p.name, p.group].some((f) => (f ?? '').toLowerCase().includes(q)),
    )
    .slice(0, 10)
}

const moveTargets = computed(() =>
  moveDlg.value ? destinations(moveDlg.value.query, moveDlg.value.team.teamId) : [],
)

const submitMove = async () => {
  const d = moveDlg.value
  if (!d || !d.target || d.selected.size === 0) return
  moveDlg.value = { ...d, submitting: true }
  try {
    // One request for the whole operation, not one per member. N calls from here could
    // half-succeed, leaving some survivors moved and some not, with the operator told only
    // that something failed — from a dialog whose selection had already gone. It also makes
    // this one timeline entry, which is what it is: one decision about one patrol's
    // remnants.
    //
    // The per-member endpoint is still what the single row action uses; this is the bulk
    // case (task 085).
    await http.post(`/sos/${props.sosId}/team/${d.team.teamId}/move`, {
      memberIds: [...d.selected],
      toTeamId: d.target.teamId,
    })
    moveDlg.value = null
    emit('changed')
  } catch {
    moveDlg.value = moveDlg.value ? { ...moveDlg.value, submitting: false } : null
    // surfaced by the axios plugin; the dialog stays open so the operator can see what
    // was selected rather than having to reconstruct it
  }
}

// --- switching one member's patrol, from the member modal ---
//
// The per-member endpoint, not the bulk one the below-strength panel posts to: that route is
// scoped to a team on the case and derives the origin from it, while this member's current
// patrol may be one they were moved *into* and which the case has never heard of.
//
// A separate dialog from the bulk move for the same reason it is a separate request: there is
// no selection to make. Offering one checkbox, pre-ticked, would be a question with one
// answer.
const switchDlg = ref<{
  memberId: string
  name: string
  fromTeamId: string
  target: TeamRef | null
  query: string
  submitting: boolean
} | null>(null)

const switchTargets = computed(() =>
  switchDlg.value ? destinations(switchDlg.value.query, switchDlg.value.fromTeamId) : [],
)

const openSwitch = () => {
  const member = detailMember.value
  if (!member) return
  switchDlg.value = {
    memberId: member.memberId,
    name: detailData.value?.member.name || member.name,
    // The patrol they are actually with, which is what they are being moved *from* and what
    // must be excluded from the destinations. For a member who moved in, that is not the
    // team whose card was clicked.
    fromTeamId: detailData.value?.currentTeam?.teamId || detail.value?.team.teamId || '',
    target: null,
    query: '',
    submitting: false,
  }
}

const submitSwitch = async () => {
  const d = switchDlg.value
  if (!d || !d.target) return
  switchDlg.value = { ...d, submitting: true }
  try {
    await http.put(`/member/${d.memberId}/team`, { sosId: props.sosId, teamId: d.target.teamId })
    switchDlg.value = null
    // The modal stays open on purpose: the operator is still on the phone about this member,
    // and the two patrol lines they just changed are the confirmation.
    await loadDetail()
    emit('changed')
  } catch {
    switchDlg.value = switchDlg.value ? { ...switchDlg.value, submitting: false } : null
    // surfaced by the axios plugin; the chosen destination stays on screen
  }
}

// The team a withdrawal is measured against is the member's *current* patrol, not the card
// the modal was opened from: those differ for somebody who was moved, and warning about the
// wrong patrol's strength would send a car to the wrong place. Falls back to the opened card
// when the current patrol is not on this case, which is the only strength figure available.
const withdrawTeam = computed<TeamRow | null>(() => {
  const currentId = detailData.value?.currentTeam?.teamId
  const onCase = currentId ? props.teams.find((t) => t.teamId === currentId) : undefined
  return onCase ?? detail.value?.team ?? null
})
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
      <!--
        A member row: status icon and name, nothing else. The row is the *index* — what an
        operator scans down looking for somebody not racing — so it carries the colour that
        answers that and the name that identifies who. Phone numbers, address, birthday and
        the lifecycle belong to one member at a time, which is what the modal is for.
      -->
      <div v-for="member in team.members" :key="member.memberId"
           class="flex cursor-pointer items-center gap-2 rounded py-1 pl-3 pr-2 text-sm hover:bg-gray-100"
           role="button" tabindex="0"
           @click="openMember(member, team)"
           @keydown.enter="openMember(member, team)"
           @keydown.space.prevent="openMember(member, team)">
        <i :class="[statusIcon(member.status), statusColour(member.status)]"
           class="text-base" :aria-label="statusLabel(member.status)" />
        <span class="font-medium" :class="nameClass(member)">{{ member.name || member.memberId }}</span>
        <!--
          Why a member with an active status is not counted in this team's strength, said on
          the row rather than left for the operator to work out. Both directions are worth
          marking: one explains a racing member who does not count, the other explains a name
          that is not on the patrol's own roster.
        -->
        <span v-if="member.movedAway" class="text-xs italic text-gray-500">
          → flyttet til anden patrulje
        </span>
        <span v-else-if="member.movedIn" class="text-xs italic text-gray-500">
          ← flyttet hertil
        </span>
      </div>
      <div v-if="team.members.length === 0" class="pl-3 py-1 text-sm text-gray-500">
        Ingen deltagere registreret.
      </div>

      <!--
        Below-strength panel. Persists for as long as the patrol is short-handed and has
        no dismiss: there is nothing to grant and nothing to settle, so it states a fact
        rather than raising an open item. The timeline below the card says how the patrol
        got here.
      -->
      <div v-if="belowStrength(team)"
           class="ml-3 mt-2 rounded border border-red-300 bg-red-50 p-2 text-sm">
        <div class="font-semibold text-red-800">
          Patruljen har kun {{ team.activeMemberCount }} tilbage i løbet — krævet er {{ team.minMemberCount }}.
        </div>
        <div class="mt-1 text-red-900">
          Aftal med patruljen og feltet hvad der skal ske, og registrer det her.
        </div>
        <div class="mt-2 flex flex-wrap gap-2">
          <Button label="Hent hele patruljen" size="small" severity="danger"
                  :loading="collecting[team.teamId]" @click="collect(team)" />
          <Button label="Flyt de resterende" size="small" severity="secondary" outlined
                  @click="openMove(team)" />
        </div>
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

    <!--
      Pre-commit warning. It names the resulting strength because that is the fact that
      changes the conversation, and it offers the two actions an operator actually takes
      — plus proceeding, which is always allowed: the member is leaving regardless.
    -->
    <Dialog v-if="confirm" :visible="true" modal header="Patruljen kommer under styrke"
            :style="{ width: '30rem' }" @update:visible="confirm = null">
      <p class="mb-3">
        Hvis <strong>{{ confirm.member.name || confirm.member.memberId }}</strong> udgår, har
        <strong>{{ confirm.team.name }}</strong> kun <strong>{{ confirm.resulting }}</strong>
        tilbage i løbet. Krævet er {{ confirm.team.minMemberCount }}.
      </p>
      <p class="mb-3 text-sm text-gray-600">
        Registreringen sker uanset hvad — aftal med feltet om resten af patruljen skal
        hentes, flyttes, eller fortsætte alligevel.
      </p>
      <template #footer>
        <div class="flex flex-wrap justify-end gap-2">
          <Button label="Annuller" severity="secondary" text @click="confirm = null" />
          <Button label="Flyt de resterende" severity="secondary" outlined
                  @click="openMove(confirm.team)" />
          <Button label="Hent hele patruljen" severity="danger" outlined
                  @click="collect(confirm.team)" />
          <Button label="Kun denne udgår" severity="danger" @click="proceedWithdraw" />
        </div>
      </template>
    </Dialog>

    <!--
      Move dialog. The member list is a snapshot taken when the dialog opened — see the
      script — so rows cannot appear or vanish under the operator mid-choice.
    -->
    <Dialog v-if="moveDlg" :visible="true" modal header="Flyt deltagere til anden patrulje"
            :style="{ width: '34rem' }" @update:visible="moveDlg = null">
      <p class="mb-2 text-sm italic text-amber-700">
        Listen er låst mens du vælger — opdateringer fra andre er sat på pause.
      </p>

      <div class="mb-3">
        <div v-for="m in moveDlg.offered" :key="m.memberId" class="flex items-center gap-2 py-1">
          <Checkbox :model-value="moveDlg.selected.has(m.memberId)" binary
                    @update:model-value="toggleMember(m.memberId)" />
          <span>{{ m.name || m.memberId }}</span>
        </div>
        <small v-if="moveDlg.offered.length === 0" class="text-gray-500">
          Ingen af patruljens deltagere er i løbet.
        </small>
      </div>

      <div v-if="moveDlg.target" class="mb-2">
        Flyttes til:
        <strong>
          <span v-if="moveDlg.target.teamNumber">{{ moveDlg.target.teamNumber }} — </span>
          {{ moveDlg.target.name }}
        </strong>
        <Button label="Skift" text size="small" @click="moveDlg.target = null" />
      </div>
      <div v-else>
        <InputText v-model="moveDlg.query" class="w-full"
                   placeholder="Søg modtagende patrulje (nummer, navn, gruppe)" />
        <div v-for="c in moveTargets" :key="c.teamId"
             class="flex items-center justify-between gap-2 py-1">
          <span class="text-sm">
            <span v-if="c.teamNumber">{{ c.teamNumber }} — </span>{{ c.name }}
            <span class="text-gray-500">{{ c.group }} · {{ c.activeMemberCount }} i løbet</span>
          </span>
          <Button label="Vælg" size="small" @click="moveDlg.target = c" />
        </div>
        <small v-if="moveDlg.query.trim().length === 1" class="text-gray-500">Skriv mindst to tegn...</small>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <Button label="Annuller" severity="secondary" text @click="moveDlg = null" />
          <Button label="Flyt" :disabled="!moveDlg.target || moveDlg.selected.size === 0"
                  :loading="moveDlg.submitting" @click="submitMove" />
        </div>
      </template>
    </Dialog>
    <!--
      Member detail. Everything known about one participant, plus the actions that belong to
      them — which is where they live now that the row is an index rather than a control
      surface.
    -->
    <Dialog v-if="detail" :visible="true" modal
            :style="{ width: '48rem' }" :breakpoints="{ '768px': '95vw' }"
            @update:visible="closeMember">
      <!--
        A titled header in the house style, like the page headings: the person is the subject
        of this dialog, so their name is set as a heading rather than as dialog chrome. The
        status rides in the header as a Tag because it is the one fact that changes while the
        dialog is open, and it belongs next to the name it qualifies.
      -->
      <template #header>
        <div class="inline-flex items-center gap-2 text-2xl">
          <i class="fas fa-fw fa-user"></i>
          <h1 class="font-nathejk">
            {{ detailData?.member.name || detailMember?.name || 'Deltager' }}
          </h1>
          <Tag v-if="detailMember" :icon="statusIcon(detailMember.status)"
               :value="statusLabel(detailMember.status)"
               :severity="statusSeverity(detailMember.status)" class="text-base" />
        </div>
      </template>

      <div v-if="detailMember" class="mb-3 flex items-center gap-2 text-sm text-gray-500">
        <span v-if="detailMember.updatedAt && detailMember.status">
          siden {{ since(detailMember.updatedAt) }}
        </span>
      </div>

      <!--
        The actions that are not about a patrol. `waiting` offers carrying on — an ordinary
        outcome that saves a car being sent, so it is a primary button rather than something
        tucked away. Leaving the race lives on the "Nuværende patrulje" line below, with the
        patrol it takes them out of. From `transit` onwards there is deliberately nothing to
        press: the car and shelter interfaces record what happens next, and saying so is
        better than an empty space that looks unfinished.
      -->
      <div v-if="detailMember" class="mb-4">
        <Button v-if="canResume(detailMember.status)" label="Fortsætter selv" severity="success"
                :loading="pending[detailMember.memberId]"
                @click="act(detailMember.memberId, 'racing')" />
        <div v-else-if="detailMember.status && !canWithdraw(detailMember.status)"
             class="text-sm italic text-gray-500">
          Ingen handlinger herfra — bil og HQ registrerer selv de næste skridt.
        </div>
      </div>

      <div v-if="detailPending && !detailData" class="text-sm text-gray-500">Henter…</div>

      <!--
        The profile in a fieldset, so the modal reads as "who this is" separated from the
        status and history around it. The two patrol lines come first because they are what an
        operator on the phone needs before an address: which patrol the scout set out with, and
        which one they are with now. They are always both shown, identical name and all — the
        ordinary case is a fact worth stating, not a value to hide.
      -->
      <template v-if="detailData">
        <Fieldset legend="Oplysninger" class="mb-4">
          <dl class="grid grid-cols-[8rem_1fr] items-baseline gap-x-3 gap-y-1 text-sm">
            <dt class="text-gray-500">Startpatrulje</dt>
            <dd>
              <router-link v-if="detailData.startTeam" class="underline"
                           :to="{ name: 'patrulje', params: { teamId: detailData.startTeam.teamId } }">
                <span v-if="detailData.startTeam.teamNumber">{{ detailData.startTeam.teamNumber }} · </span>
                {{ detailData.startTeam.name || 'Patrulje' }}
              </router-link>
              <span v-else class="text-gray-400">—</span>
            </dd>

            <dt class="text-gray-500">Nuværende patrulje</dt>
            <dd class="flex flex-wrap items-center gap-2">
              <router-link v-if="detailData.currentTeam" class="underline"
                           :to="{ name: 'patrulje', params: { teamId: detailData.currentTeam.teamId } }">
                <span v-if="detailData.currentTeam.teamNumber">{{ detailData.currentTeam.teamNumber }} · </span>
                {{ detailData.currentTeam.name || 'Patrulje' }}
              </router-link>
              <span v-else class="text-gray-400">—</span>

              <!--
                The two things that change a scout's patrol, on the line that states it. Both
                are offered only while the member is on their own legs: once a car has them,
                what happens next is recorded by the car and shelter interfaces, and a button
                here would be this screen pretending to own a transition it does not.
              -->
              <span v-if="detailMember && canWithdraw(detailMember.status)" class="flex gap-1">
                <Button label="Skift" size="small" severity="secondary" outlined
                        @click="openSwitch" />
                <Button label="Udgår" size="small" severity="danger" outlined
                        :loading="pending[detailMember.memberId]"
                        @click="withdrawTeam && askWithdraw(detailMember, withdrawTeam)" />
              </span>
            </dd>

            <dt class="text-gray-500">Telefon</dt>
            <dd>
              <!-- tel: so an operator on a phone taps to call rather than copying digits -->
              <a v-if="detailData.member.phone" :href="`tel:${detailData.member.phone}`" class="underline">
                {{ detailData.member.phone }}
              </a>
              <span v-else class="text-gray-400">—</span>
            </dd>

            <dt class="text-gray-500">Kontaktperson</dt>
            <dd>
              <a v-if="detailData.member.phoneParent" :href="`tel:${detailData.member.phoneParent}`" class="underline">
                {{ detailData.member.phoneParent }}
              </a>
              <span v-else class="text-gray-400">—</span>
            </dd>

            <dt class="text-gray-500">Adresse</dt>
            <dd>
              <span v-if="detailData.member.address">
                {{ detailData.member.address }}<br>
                {{ detailData.member.postalCode }} {{ detailData.member.city }}
              </span>
              <span v-else class="text-gray-400">—</span>
            </dd>

            <dt class="text-gray-500">Fødselsdag</dt>
            <dd>
              <span v-if="detailData.member.birthday">{{ birthday(detailData.member.birthday) }}</span>
              <span v-else class="text-gray-400">—</span>
            </dd>
          </dl>
        </Fieldset>

        <!--
          The lifecycle, oldest first, because it reads as a story: started, asked to leave,
          carried on. Both the status and the event are shown — they answer different
          questions, and "racing" reached by carrying on is a different fact from "racing"
          reached by being moved to another patrol.
        -->
        <h3 class="font-nathejk text-lg">Statushistorik</h3>
        <ol v-if="detailData.history.length" class="mt-1 text-sm">
          <li v-for="entry in detailData.history" :key="entry.seq"
              class="flex items-baseline gap-2 border-b border-gray-100 py-1 last:border-0">
            <i :class="[statusIcon(entry.status), statusColour(entry.status)]" class="text-xs" />
            <span class="w-28 shrink-0 text-gray-500">{{ formatDateTime(entry.createdAt) }}</span>
            <span>{{ memberEventPhrase(entry.event) }}</span>
            <span class="text-gray-500">→ {{ statusLabel(entry.status) }}</span>
          </li>
        </ol>
        <p v-else class="mt-1 text-sm text-gray-500">
          Ingen statusskift registreret — deltageren er ikke startet.
        </p>
      </template>
    </Dialog>

    <!--
      Switching one member's patrol. Opened from the member modal and left on top of it, so the
      operator keeps the person they are talking about in view while naming the destination.
    -->
    <Dialog v-if="switchDlg" :visible="true" modal header="Skift patrulje"
            :style="{ width: '26rem' }" @update:visible="switchDlg = null">
      <p class="mb-3 text-sm">
        <strong>{{ switchDlg.name }}</strong> flyttes til en anden patrulje og bliver ved med at
        være i løbet. Startpatruljen ændres ikke.
      </p>

      <div v-if="switchDlg.target" class="mb-2 flex items-center gap-2">
        <strong>
          <span v-if="switchDlg.target.teamNumber">{{ switchDlg.target.teamNumber }} · </span>
          {{ switchDlg.target.name }}
        </strong>
        <Button label="Skift" text size="small" @click="switchDlg.target = null" />
      </div>
      <div v-else>
        <InputText v-model="switchDlg.query" class="w-full"
                   placeholder="Søg patrulje (nummer, navn, gruppe)" />
        <div v-for="c in switchTargets" :key="c.teamId"
             class="flex items-center justify-between gap-2 border-b border-gray-100 py-1 last:border-0">
          <span class="text-sm">
            <span v-if="c.teamNumber">{{ c.teamNumber }} · </span>
            {{ c.name }}
            <span class="text-gray-500">{{ c.group }}</span>
          </span>
          <Button label="Vælg" size="small" @click="switchDlg.target = c" />
        </div>
        <small v-if="switchDlg.query.trim().length === 1" class="text-gray-500">Skriv mindst to tegn</small>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <Button label="Annuller" severity="secondary" text @click="switchDlg = null" />
          <Button label="Flyt" :disabled="!switchDlg.target" :loading="switchDlg.submitting"
                  @click="submitSwitch" />
        </div>
      </template>
    </Dialog>
  </div>
</template>
