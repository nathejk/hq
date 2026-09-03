<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { memberStatusBadge, memberStatusColour } from '@/composables/sos'
import MemberDetailDialog from '@/components/MemberDetailDialog.vue'
import PositionIndicator from '@/components/PositionIndicator.vue'

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
    maxMemberCount: number
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

const emit = defineEmits<{
  changed: []
  /**
   * The operator wants a car for these members (PRD 009 §6).
   *
   * Emitted rather than handled here: the case owns the dispatch dialog, because the task belongs
   * to the *case* and not to this card — and because one dialog on the view is one place where the
   * case id, the patrol and the members are assembled.
   */
  'request-transport': [{ teamId: string; memberIds: string[]; label: string }]
}>()

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
const statusIcon = (slug: string) => memberStatusBadge(slug).icon

// Moved to composables/sos.ts as `memberStatusColour` (task 103): the member dialog's history
// timeline needs the same mapping, and a copy in each component is how the two would drift.
const statusColour = memberStatusColour

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

// The strength badge's colour, which is the whole point of showing the number as a badge: an
// operator reads "is this patrol all right?" off it without doing arithmetic against the
// minimum. Green inside the expected band, amber outside it in either direction — too many is
// also worth a glance, since it means members have been moved in and the patrol is carrying
// somebody else's people.
//
// Never red: below strength is not an error, it is a situation the operator is already
// handling, and the Under styrke tag beside it says so in words.
const strengthSeverity = (team: { activeMemberCount: number; minMemberCount: number; maxMemberCount: number }) =>
  team.activeMemberCount >= team.minMemberCount && team.activeMemberCount <= team.maxMemberCount
    ? 'success'
    : 'warn'

// What the badge counts, since a bare number does not say. The expected band is deliberately
// not repeated here — the colour carries whether the count is a problem, and the Under styrke
// tag beside it says so in words when it is.
const strengthTooltip = (team: { activeMemberCount: number }) =>
  team.activeMemberCount === 1
    ? '1 spejder aktiv i løbet'
    : `${team.activeMemberCount} spejdere aktive i løbet`

// **The "started" half is not optional.** A team that never started also has zero racing
// members, so the count alone conflates *left the route* with *never on it* — without
// this, every patrol of a year that has not raced yet would be badged Udgået, which on
// the dev data is all 310 of them.
const discontinued = (team: { started: boolean; activeMemberCount: number }) =>
  team.started && team.activeMemberCount === 0

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

// The payload the dialog loaded, mirrored here because two things outside the dialog need facts from
// it: the switch dialog's "from" patrol, and the strength the withdrawal warning is measured against.
// Fed by the dialog's `loaded` event rather than fetched again — a second request for the same member
// would give two copies that disagree the moment one revalidates.
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

// The open member as the *card* sees them — live, because props.teams is. This is what the
// modal's status badge and action buttons read, so they cannot disagree with the row behind
// the modal.
const detailMember = computed<MemberRow | null>(() => {
  const d = detail.value
  if (!d) return null
  const team = props.teams.find((t) => t.teamId === d.team.teamId)
  return team?.members.find((m) => m.memberId === d.memberId) ?? null
})

const openMember = (member: MemberRow, team: TeamRow) => {
  detail.value = { memberId: member.memberId, team }
  detailData.value = null
}

const closeMember = () => {
  detail.value = null
  detailData.value = null
}

// Reloading the detail on a status change used to be a watch here. It is now the dialog's own
// business: it depends on `spejder:{memberId}`, so a status change or a note — this operator's or a
// colleague's — invalidates it. That is strictly better than the watch, which only fired for changes
// visible in *this card's* live row and so missed anything done from another screen.

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

// Dimmed *and* struck through rather than hidden: a member who has left the race is still the
// patrol's member and still the person an operator may be asked about. The status icon keeps
// its own colour either way — a waiting member is out of the race *and* urgent, and the strike
// must not be allowed to quieten the second fact.
//
// The strike says "no longer in the race with this patrol" — which is only meaningful for
// somebody who was in it. A member of a patrol that has not started has lost nothing, so they
// are dimmed but not struck; striking every name on a case opened before the race would read
// as a list of casualties.
//
// gray-500 rather than a lighter grey deliberately: these are names read off a screen at
// three in the morning, and gray-400 on white falls below the contrast a small font needs.
const nameClass = (member: MemberRow) => [
  isActive(member) ? '' : 'text-gray-500',
  member.status && !isActive(member) ? 'line-through' : '',
]

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
    // and the two patrol lines they just changed are the confirmation. No reload needed — the
    // move publishes on the member's own subject, so the dialog's instance dependency refetches it.
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
            conversation the operator is having changes. A bare count in a badge: the
            colour carries whether it is a problem, so the "/3" it used to be measured
            against is only needed when it is one — and then the Under styrke tag says so
            in words. Absent at zero, where the Udgået tag has already said everything.
          -->
          <Badge v-if="team.activeMemberCount > 0" :value="team.activeMemberCount"
                 :severity="strengthSeverity(team)" class="ml-2"
                 v-tooltip.top="strengthTooltip(team)" />
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
        Member rows. The list is the union of everybody who started with this patrol and
        everybody attached to it now — including those who left the race while they were with
        it, since their row keeps pointing here. Three marks carry the difference:

          - the status icon on the left, whose colour is what an operator scans a list for — a
            waiting member needs a car, and that must stay the loudest thing on the row even
            though they are also struck through,
          - a struck-through name is somebody no longer in the race with this patrol, whether
            they left it or moved on to another,
          - a user-plus is somebody who did not start here.

        The label, the history and the actions on a status are one member at a time, which is
        what the modal is for. Here the icon is an index, not a control.
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
        <!-- Does this scout's phone report positions? The most useful thing to know on a SOS card. -->
        <PositionIndicator :person-id="member.memberId" />
        <!--
          Marked rather than left to be worked out: a name that is not on this patrol's own
          roster is otherwise indistinguishable from one that is, and "who did we put here?"
          is a question an operator asks out loud on the phone.
        -->
        <i v-if="member.movedIn" class="pi pi-user-plus text-xs text-gray-500"
           v-tooltip.top="'Startede i en anden patrulje'"
           aria-label="Startede i en anden patrulje" />
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
    <!--
      One member in full. Extracted to its own component in task 103 so Hønsegården and the patrol
      page show the same thing — and so the note trail (PRD 008) has one home rather than three.

      The card keeps what is the card's: which member is open, the live status the badge shows, and
      the actions, which are the nødtelefon's own and belong to no other host.
    -->
    <MemberDetailDialog
      v-if="detail"
      :member-id="detail.memberId"
      :team-id="detail.team.teamId"
      :status="detailMember?.status"
      :name="detailMember?.name"
      @close="closeMember"
      @loaded="detailData = $event"
    >
      <!--
        Every action a nødtelefon operator has over this member: two move the scout between patruljer
        and the third puts them back in the race. Same size and weight for all three, because a
        differently-shaped button in a row of three reads as "press this one" — and which one is right
        depends on the call, not on the layout.

        Which appear follows the status: `racing` can be switched or leave the race, `waiting` can
        carry on. From `transit` onwards there is deliberately nothing to press — the car and shelter
        interfaces record what happens next, and saying so is better than an empty space that looks
        unfinished.
      -->
      <template #actions>
        <template v-if="detailMember">
          <template v-if="canWithdraw(detailMember.status)">
            <Button label="Skift" size="small" severity="secondary" outlined @click="openSwitch" />
            <Button
              label="Udgår"
              size="small"
              severity="danger"
              outlined
              :loading="pending[detailMember.memberId]"
              @click="withdrawTeam && askWithdraw(detailMember, withdrawTeam)"
            />
          </template>
          <Button
            v-else-if="canResume(detailMember.status)"
            label="Fortsætter selv"
            size="small"
            severity="success"
            outlined
            :loading="pending[detailMember.memberId]"
            @click="act(detailMember.memberId, 'racing')"
          />
          <!--
            A car for this scout (PRD 009). Beside "Fortsætter selv" and only on a waiting member,
            because those are the two things that can happen next to somebody sitting by the
            trailside: they carry on, or somebody comes for them. One click, from the screen the
            operator is already on — which is the whole mitigation for the desk-discipline risk.
          -->
          <Button
            v-if="canResume(detailMember.status) && detail"
            label="Bestil kørsel"
            icon="pi pi-truck"
            size="small"
            severity="secondary"
            outlined
            @click="
              emit('request-transport', {
                teamId: detail.team.teamId,
                memberIds: [detail.memberId],
                label: `${detailMember.name} (${detail.team.name})`,
              })
            "
          />
          <span v-else-if="detailMember.status" class="text-xs italic text-gray-500">
            Bil og HQ registrerer selv de næste skridt
          </span>
        </template>
      </template>
    </MemberDetailDialog>

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

<style>
/*
  The member history's rail, mirroring the case timeline's in SosView.

  Unscoped deliberately: the dialog is teleported out of this component's tree, so a
  scoped attribute never reaches it, and these rules target PrimeVue's internals rather
  than markup of our own. The `.member-timeline` class is the scope.
*/
.member-timeline .p-timeline-event-opposite {
  /* Fixed and right-aligned so the timestamps form a column: "how long between these two
     things" is then read straight down the page. Wider than the case timeline's 3.5rem
     because these entries carry the date as well as the time — a member's history spans
     the whole event, while a case is usually one evening.

     nowrap because a timestamp is one token: "17.08 12:46" broken after the date reads as
     two facts, and it wrapped at every modal width where the column was a hair too narrow.
     The width then only has to be enough not to collide with the marker. */
  flex: 0 0 6.5rem;
  white-space: nowrap;
  padding-top: 0.15rem;
  text-align: right;
}

/* Aura sizes every event at `timeline.event.minHeight: 5rem`, which is the real source of
   the vertical spacing. Halved via the design token rather than fought with padding, so
   connector, marker and content shorten together and stay aligned. */
.member-timeline {
  --p-timeline-event-min-height: 2.75rem;
}

.member-timeline .p-timeline-event-content {
  padding-bottom: 0.375rem;
}
</style>
