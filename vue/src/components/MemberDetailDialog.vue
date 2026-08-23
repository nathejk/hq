<script setup lang="ts">
// One scout, in full: who they are, how to reach them, how they got here, and what has been agreed
// about them.
//
// Extracted from `SosTeamCard.vue` (task 103), where it was a `detail` ref and a `<Dialog>` in the
// middle of a 956-line component. Two reasons, and the second is the one that made it urgent:
//
//   1. PRD 008 needs the same view from three screens — the case card, Hønsegården and the patrol
//      page. One trail about a scout is worth having only if every screen shows it.
//   2. It was **not live**. `loadDetail()` was a plain `http.get` into a local ref, so two crew
//      members looking at the same scout could not see each other's work. For a shared note trail
//      that is disqualifying.
//
// # Actions belong to the host, not here
//
// The nødtelefon can withdraw a scout, switch their patrol and resume them; the shelter can accept
// and hand over; the patrol page can do neither. So this component owns the *view* and takes the
// buttons through an `actions` slot. The alternative — every host's commands imported here, behind
// flags saying which screen we are on — is how a shared component becomes three components in a
// trenchcoat.

import { computed, watch } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { parseApiDate } from '@/composables/datefilters'
import { memberEventPhrase, formatDateTime, memberStatusBadge, memberStatusColour } from '@/composables/sos'
import MemberNotes from '@/components/MemberNotes.vue'

const props = defineProps<{
  memberId: string
  /**
   * The member's team, needed by the endpoint: a scout the lifecycle has never touched has no status
   * row, so the server cannot infer which roster to find them on.
   */
  teamId: string
  /**
   * The scout's current status, from the host's own live row. Passed in rather than read from the
   * payload here so the badge cannot disagree with the row the dialog was opened from — the
   * behaviour `SosTeamCard` had, and worth keeping.
   */
  status?: string
  /** Fallback name, shown until the payload arrives. */
  name?: string
}>()

const emit = defineEmits<{
  close: []
  /**
   * Forwarded from the note form.
   *
   * The dialog itself needs no deferral: its own payload can be replaced while somebody types,
   * because `MemberNotes` owns the textarea's text and `v-if="detail"` stays true, so the component
   * is never unmounted. What *does* need it is whatever list the host is rendering behind this
   * dialog — Hønsegården's table reorders when a scout changes section. So the signal is passed
   * outward rather than consumed here.
   */
  dirty: [boolean]

  /**
   * The loaded payload.
   *
   * Emitted because a host may need facts from it that it cannot derive itself — the case card's
   * "switch patrol" dialog needs the scout's *current* patrol, which differs from the card it was
   * opened from for anybody who was moved. Without this the card would have to fetch the same member
   * a second time, and the two copies would disagree the moment one of them revalidated.
   */
  loaded: [MemberDetail]
}>()

interface TeamRef {
  teamId: string
  teamNumber: string
  name: string
}

interface MemberDetail {
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
}

// Live, and keyed per member so navigating between scouts renders from cache.
//
// `spejder:{id}` is the instance dependency: this member's own events — a status change, a note —
// invalidate it, while another scout's do not, so an open dialog is not refetched every time anybody
// anywhere is accepted. `patrulje` is type-level because the payload joins in team names and numbers.
const { data, pending, error } = useLiveResource(
  `member:${props.memberId}`,
  async () => {
    const response = await http.get(`/member/${props.memberId}`, { params: { teamId: props.teamId } })
    return response.data as MemberDetail
  },
  { dependsOn: [`spejder:${props.memberId}`, 'patrulje'] },
)

const detail = computed<MemberDetail | null>(() => data.value ?? null)

// Forwarded on every load, including revalidations, so a host holding facts from the payload cannot
// end up with a stale copy of them.
watch(
  data,
  (payload) => {
    if (payload) emit('loaded', payload)
  },
  { immediate: true },
)

const statusLabel = (slug: string) => memberStatusBadge(slug).label
const statusSeverity = (slug: string) => memberStatusBadge(slug).severity
const statusIcon = (slug: string) => memberStatusBadge(slug).icon

// Moved verbatim from SosTeamCard — as `memberStatusColour` in composables/sos.ts, because the card's
// member rows need the same mapping and a copy in each component is how the two were going to drift.
// The full severity set is kept deliberately: dropping the `info`/`secondary` arms while extracting
// would have silently changed the colour of two statuses in the history timeline, which is the kind of
// regression a refactor is supposed to be incapable of.
const statusColour = memberStatusColour

// Birthday is a date, not a moment. The API serves it as an instant, so formatting it as a
// datetime would show the day before for anybody born in the evening — a scout born on the
// 5th at 23:00 UTC is a 5th-of-December birthday in Copenhagen, not a 4th.
//
// Moved verbatim from SosTeamCard, format included: rendering it as 05.12.2026 instead of
// "5. december 2026" would have been a quiet regression dressed up as a refactor.
const birthday = (value: string | null) => {
  const date = parseApiDate(value ?? undefined)
  return date
    ? date.toLocaleDateString('da-DK', { day: 'numeric', month: 'long', year: 'numeric' })
    : ''
}

watch(error, (err) => {
  if (err) console.log('member detail load failed', err)
})
</script>

<template>
  <Dialog
    :visible="true"
    modal
    :style="{ width: '48rem' }"
    :breakpoints="{ '768px': '95vw' }"
    @update:visible="emit('close')"
  >
    <!--
      A titled header in the house style: the person is the subject of this dialog, so their name is
      a heading rather than dialog chrome. The status rides beside it because it is the one fact that
      changes while the dialog is open, and it belongs next to the name it qualifies.
    -->
    <template #header>
      <div class="inline-flex items-center gap-2 text-2xl">
        <i class="fas fa-fw fa-user"></i>
        <h1 class="font-nathejk">{{ detail?.member.name || name || 'Deltager' }}</h1>
        <Tag
          v-if="status"
          :icon="statusIcon(status)"
          :value="statusLabel(status)"
          :severity="statusSeverity(status)"
          class="text-base"
        />
      </div>
    </template>

    <!-- Only when nothing is cached: reopening a scout must not flash. -->
    <div v-if="pending && !detail" class="text-sm text-gray-500">Henter…</div>

    <template v-if="detail">
      <!--
        The profile first, and within it the two patrol lines before the address, because that is
        what an operator on the phone needs first: which patrol the scout set out with, and which one
        they are with now. Both are always shown, identical name and all — the ordinary case is a
        fact worth stating, not a value to hide.
      -->
      <Fieldset legend="Oplysninger" class="mb-4">
        <dl class="grid grid-cols-[8rem_1fr] items-baseline gap-x-3 gap-y-1 text-sm">
          <dt class="text-gray-500">Startpatrulje</dt>
          <dd>
            <router-link
              v-if="detail.startTeam"
              class="underline"
              :to="{ name: 'patrulje', params: { teamId: detail.startTeam.teamId } }"
            >
              <span v-if="detail.startTeam.teamNumber">{{ detail.startTeam.teamNumber }} · </span>
              {{ detail.startTeam.name || 'Patrulje' }}
            </router-link>
            <span v-else class="text-gray-400">—</span>
          </dd>

          <dt class="text-gray-500">Nuværende patrulje</dt>
          <dd class="flex flex-wrap items-center gap-2">
            <router-link
              v-if="detail.currentTeam"
              class="underline"
              :to="{ name: 'patrulje', params: { teamId: detail.currentTeam.teamId } }"
            >
              <span v-if="detail.currentTeam.teamNumber">{{ detail.currentTeam.teamNumber }} · </span>
              {{ detail.currentTeam.name || 'Patrulje' }}
            </router-link>
            <span v-else class="text-gray-400">—</span>

            <!--
              The host's actions, right-aligned at the end of the line they all change. Which buttons
              exist, and what they do, is the host's business: the nødtelefon withdraws and switches,
              the shelter receives and hands over, the patrol page does neither.
            -->
            <span class="ml-auto flex gap-1">
              <slot name="actions" />
            </span>
          </dd>

          <dt class="text-gray-500">Telefon</dt>
          <dd>
            <!-- tel: so an operator on a phone taps to call rather than copying digits -->
            <a v-if="detail.member.phone" :href="`tel:${detail.member.phone}`" class="underline">
              {{ detail.member.phone }}
            </a>
            <span v-else class="text-gray-400">—</span>
          </dd>

          <dt class="text-gray-500">Kontaktperson</dt>
          <dd>
            <a
              v-if="detail.member.phoneParent"
              :href="`tel:${detail.member.phoneParent}`"
              class="underline"
            >
              {{ detail.member.phoneParent }}
            </a>
            <span v-else class="text-gray-400">—</span>
          </dd>

          <dt class="text-gray-500">Adresse</dt>
          <dd>
            <span v-if="detail.member.address">
              {{ detail.member.address }}<br />
              {{ detail.member.postalCode }} {{ detail.member.city }}
            </span>
            <span v-else class="text-gray-400">—</span>
          </dd>

          <dt class="text-gray-500">Fødselsdag</dt>
          <dd>
            <span v-if="detail.member.birthday">{{ birthday(detail.member.birthday) }}</span>
            <span v-else class="text-gray-400">—</span>
          </dd>
        </dl>
      </Fieldset>

      <!--
        Notes above the history: what was agreed is what the next crew member needs, and the history
        is the reference behind it. This is the whole point of the extraction (PRD 008) — the same
        trail from every screen that shows a scout.
      -->
      <Fieldset legend="Noter" class="mb-4">
        <MemberNotes :member-id="memberId" @dirty="emit('dirty', $event)" />
      </Fieldset>

      <!--
        The lifecycle, oldest first, because it reads as a story: started, asked to leave, carried on.
        Both status and event are shown — they answer different questions, and "racing" reached by
        carrying on is a different fact from "racing" reached by being moved to another patrol.
      -->
      <Fieldset legend="Historik">
        <Timeline v-if="detail.history.length" :value="detail.history" class="member-timeline text-sm">
          <template #marker="{ item }">
            <span
              class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-gray-300 bg-white"
            >
              <i :class="[statusIcon(item.status), statusColour(item.status)]" class="text-xs" />
            </span>
          </template>
          <template #opposite="{ item }">
            <!--
              The time in its own column, which is what the timeline gives that a list did not: "how
              long between these two things" is read down the page rather than hunted for mid-sentence.
            -->
            <span class="text-gray-500">{{ formatDateTime(item.createdAt) }}</span>
          </template>
          <template #content="{ item }">
            <div>{{ memberEventPhrase(item.event) }}</div>
            <div class="text-gray-500">→ {{ statusLabel(item.status) }}</div>
          </template>
        </Timeline>
        <p v-else class="text-sm text-gray-500">
          Ingen statusskift registreret — deltageren er ikke startet.
        </p>
      </Fieldset>
    </template>
  </Dialog>
</template>
