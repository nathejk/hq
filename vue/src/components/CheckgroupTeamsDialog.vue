<script setup lang="ts">
// Every started team's standing at one postlinje, behind the four numbers on the post list.
//
// The numbers alone answer "how many", which is enough until it isn't: with a line about to
// close, the only useful question is *which* patrols are still out there and who to ring about
// them. So the counts are clickable and this is what they open — the same arithmetic, listed.
//
// The filter defaults to whichever number was clicked and hides the rest, because that is the
// question that was asked; the other three chips stay on screen with their counts so widening
// back out is one click and the whole picture is never more than a glance away.

import { computed, ref, watch } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { useNow } from '@/composables/shelter'

const props = defineProps<{
  checkgroupId: string
  /** Fallback name, shown until the payload arrives. */
  name?: string
  /** The status whose number was clicked; the filter starts here. */
  initialStatus?: string
}>()

const emit = defineEmits<{ close: [] }>()

type TeamStatus = 'onTime' | 'late' | 'retired' | 'missing'

interface TeamRow {
  teamId: string
  teamNumber: string
  name: string
  group: string
  contactName: string
  contactPhone: string
  memberCount: number
  activeMemberCount: number
  status: TeamStatus
  scannedAtUts?: number
  /**
   * When this team was due here. Per team, not per line: on a relative postlinje the clock
   * starts when that team left the previous line, so two patrols an hour apart have deadlines
   * an hour apart. Absent when the line keeps no time, or when a relative line's reference
   * scan has not happened yet — there is no deadline to state, and inventing one from the
   * line's own opening hours is exactly the mistake this replaced.
   */
  deadlineUts?: number
  /**
   * How far this team is from the line, for a team that has not arrived. Absent when it has, and
   * absent when HQ does not know where it is — which is the majority case, not an error.
   *
   * `ts` is epoch milliseconds and is not decoration: HQ's knowledge of a patrol's position goes
   * stale by hours between posts, so the distance is only ever shown together with its age.
   */
  distance?: {
    meters: number
    checkpointName: string
    ts: number
    source: 'track' | 'scan'
  }
}

interface Payload {
  checkgroup: { id: string; name: string }
  teams: TeamRow[]
  stats: { onTime: number; late: number; retired: number; missing: number }
  closesAtUts: number
}

// Live, keyed per line.
//
// dependsOn, each token taken from the projection that owns the event rather than from the
// names on this screen:
//   checkgroup:{id}  this line's own windows and scheme
//   checkpoint       the open windows a scan is judged against
//   checkpersonnel   the rota — a scan is attributed to a post through the scanner's shift,
//                    so a shift being corrected moves these numbers
//   qr               the scans themselves. NOTE the token is `qr`, not `scan`
//   track            telemetry positions, which the distance column rests on. From
//                    TELEMETRY.{year}.track.{personId}.reported — not `telemetry`, not `position`
//   patrulje         who started
//   spejder          activeMemberCount, which is what makes a team retired rather than
//                    missing. Maintained by the spejderstatus projection, whose subjects are
//                    spejder — so a withdrawal reaches this list only through this token.
const { data, pending, error, refresh } = useLiveResource(
  `checkgroup:${props.checkgroupId}:teams`,
  async () => {
    const response = await http.get(`/checkgroup/${props.checkgroupId}/teams`)
    return response.data as Payload
  },
  {
    dependsOn: [
      `checkgroup:${props.checkgroupId}`,
      'checkpoint',
      'checkpersonnel',
      'qr',
      'track',
      'patrulje',
      'spejder',
    ],
  },
)

const detail = computed<Payload | null>(() => data.value ?? null)
const teams = computed(() => detail.value?.teams ?? [])
const stats = computed(() => detail.value?.stats ?? { onTime: 0, late: 0, retired: 0, missing: 0 })

const STATUS_META: Record<TeamStatus, { label: string; severity: string; icon: string }> = {
  onTime: { label: 'Til tiden', severity: 'success', icon: 'pi pi-bolt' },
  late: { label: 'For sent', severity: 'warn', icon: 'pi pi-clock' },
  retired: { label: 'Udgået', severity: 'contrast', icon: 'pi pi-heart' },
  missing: { label: 'Mangler', severity: 'danger', icon: 'pi pi-eye' },
}

const ORDER: TeamStatus[] = ['onTime', 'late', 'retired', 'missing']

/**
 * Presentation for a status, tolerating one this build has never heard of.
 *
 * A function rather than indexing the map in the template: DataTable's slot types its row as
 * `any`, so the template cannot narrow the key — and the fallback means a status added to the
 * API before this component knows about it renders as itself instead of crashing the table.
 */
const statusMeta = (status: string) =>
  STATUS_META[status as TeamStatus] ?? { label: status, severity: 'secondary', icon: '' }

/**
 * Which statuses are shown. Starts at the clicked number, or everything.
 *
 * Never allowed to become empty: an empty selection would render a table with no rows and no
 * explanation, which reads as a broken screen rather than as a filter.
 */
const selected = ref<TeamStatus[]>(
  props.initialStatus && props.initialStatus in STATUS_META
    ? [props.initialStatus as TeamStatus]
    : [...ORDER],
)

watch(selected, (value) => {
  if (!value || value.length === 0) selected.value = [...ORDER]
})

const filterOptions = computed(() =>
  ORDER.map((status) => ({
    value: status,
    label: `${STATUS_META[status].label} (${stats.value[status]})`,
  })),
)

const visibleTeams = computed(() => teams.value.filter((t) => selected.value.includes(t.status)))

/** True when everything is shown, so the header can say so rather than listing four chips back. */
const showingAll = computed(() => selected.value.length === ORDER.length)

const now = useNow()

const clock = (uts?: number) =>
  uts
    ? new Date(uts * 1000).toLocaleTimeString('da-DK', { hour: '2-digit', minute: '2-digit' })
    : ''

/** Weekday and time, because a line can close after midnight. */
const closesAt = computed(() => {
  const uts = detail.value?.closesAtUts
  if (!uts) return ''
  return new Date(uts * 1000).toLocaleString('da-DK', {
    weekday: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
})

const minutesToClose = computed(() => {
  const uts = detail.value?.closesAtUts
  if (!uts) return null
  return Math.round((uts * 1000 - now.value) / 60000)
})

/** Minutes late, negative for time still left. Null when the team has no deadline. */
const overrun = (row: TeamRow): number | null => {
  if (!row.deadlineUts) return null
  const against = row.scannedAtUts ? row.scannedAtUts * 1000 : now.value
  return Math.round((against - row.deadlineUts * 1000) / 60000)
}

/**
 * Patrols past their own deadline and still not through.
 *
 * The relative line's equivalent of "the line is closing": there is no single closing time to
 * warn about, so what is left to say is how many teams are individually overdue right now.
 * Recomputed off the ticking clock, so a patrol joins this the minute its own time runs out.
 */
const overdue = computed(() =>
  teams.value.filter((t) => {
    if (t.status !== 'missing' || !t.deadlineUts) return false
    return t.deadlineUts * 1000 <= now.value
  }),
)

/**
 * The line is closing and somebody is still out.
 *
 * The whole point of the screen at this moment: once the last post shuts, a missing patrol can
 * no longer come through anywhere on the line, so the hour before that is when the list has to
 * be in front of somebody. An hour ahead rather than minutes, because ringing a contact and
 * getting an answer takes longer than that.
 */
const closingSoon = computed(() => {
  const minutes = minutesToClose.value
  return minutes !== null && minutes > 0 && minutes <= 60 && stats.value.missing > 0
})

const closed = computed(() => {
  const minutes = minutesToClose.value
  return minutes !== null && minutes <= 0
})

const showOnlyMissing = () => (selected.value = ['missing'])

/**
 * Distance, in the units the number deserves.
 *
 * Kilometres with one decimal above a kilometre, whole tens of metres below it: a patrol reported
 * as "1.83 km" invites arithmetic the underlying position does not support — it is a straight line
 * from a phone fix that may be minutes old — while "350 m" reads as "basically here", which is the
 * decision being made.
 */
const formatDistance = (meters: number) => {
  if (meters >= 1000) return `${(meters / 1000).toLocaleString('da-DK', { maximumFractionDigits: 1, minimumFractionDigits: 1 })} km`
  return `${Math.round(meters / 10) * 10} m`
}

/** Compact age of a position: "14 min siden". Ticks, so a distance visibly ages on screen. */
const formatAge = (ts: number) => {
  const minutes = Math.floor(Math.max(now.value - ts, 0) / 60000)
  if (minutes < 1) return 'lige nu'
  if (minutes < 60) return `${minutes} min siden`
  const hours = Math.floor(minutes / 60)
  return `${hours} t ${minutes % 60} min siden`
}

/**
 * A position old enough that the distance beside it should not be trusted.
 *
 * Half an hour on foot is well over a kilometre, so past that the number says where the patrol
 * *was*, not where it is. Greyed rather than hidden: knowing they were 2 km out at midnight is
 * still worth more than a dash.
 */
const STALE_POSITION_MS = 30 * 60 * 1000
const positionIsStale = (ts: number) => now.value - ts > STALE_POSITION_MS
</script>

<template>
  <Dialog
    :visible="true"
    modal
    maximizable
    :style="{ width: '58rem' }"
    :breakpoints="{ '1199px': '90vw', '575px': '95vw' }"
    @update:visible="emit('close')"
  >
    <template #header>
      <div class="inline-flex items-center gap-2 text-2xl">
        <i class="fas fa-fw fa-flag-checkered"></i>
        <h1 class="font-nathejk">{{ detail?.checkgroup?.name || name || 'Postlinje' }}</h1>
      </div>
    </template>

    <div v-if="pending && !detail" class="text-sm text-gray-500">Henter…</div>
    <div v-else-if="error && !detail" class="text-sm text-red-600">
      Kunne ikke hente holdene. <Button label="Prøv igen" text size="small" @click="refresh" />
    </div>

    <template v-if="detail">
      <!--
        Said before the list, not after it: an operator opening this in the last hour needs to
        know the deadline before they start reading names. Actionable, too — the shortcut is
        there because "which ones are missing" is the next thing they will do anyway.
      -->
      <Message v-if="closingSoon" severity="warn" :closable="false" class="mb-3">
        Postlinjen lukker {{ closesAt }} — om {{ minutesToClose }} min.
        <strong>{{ stats.missing }}</strong>
        {{ stats.missing === 1 ? 'patrulje mangler' : 'patruljer mangler' }} stadig.
        <Button
          v-if="!(selected.length === 1 && selected[0] === 'missing')"
          label="Vis kun dem der mangler"
          text
          size="small"
          @click="showOnlyMissing"
        />
      </Message>
      <Message v-else-if="closed && stats.missing > 0" severity="secondary" :closable="false" class="mb-3">
        Postlinjen lukkede {{ closesAt }}.
        <strong>{{ stats.missing }}</strong>
        {{ stats.missing === 1 ? 'patrulje' : 'patruljer' }} kom aldrig igennem.
      </Message>
      <p v-else-if="closesAt" class="text-sm text-gray-500 pb-3">Postlinjen lukker {{ closesAt }}.</p>
      <!--
        A relative line has no closing time to warn about — each patrol has its own — so the
        warning is about the patrols themselves. Same shortcut, same reason as above.
      -->
      <Message
        v-else-if="!closesAt && overdue.length > 0"
        severity="warn"
        :closable="false"
        class="mb-3"
      >
        <strong>{{ overdue.length }}</strong>
        {{ overdue.length === 1 ? 'patrulje har' : 'patruljer har' }} overskredet sin frist og er
        ikke kommet igennem.
        <Button
          v-if="!(selected.length === 1 && selected[0] === 'missing')"
          label="Vis kun dem der mangler"
          text
          size="small"
          @click="showOnlyMissing"
        />
      </Message>

      <!--
        The filter above the list, as a row of counts. Doubles as the summary: the four numbers
        from the post list are here too, so the dialog answers "how many" and "which" at once
        and an operator never has to close it to see the shape of things again.
      -->
      <div class="flex flex-wrap items-center gap-2 pb-3">
        <SelectButton
          v-model="selected"
          :options="filterOptions"
          optionLabel="label"
          optionValue="value"
          multiple
          size="small"
        />
        <Button
          v-if="!showingAll"
          label="Vis alle"
          icon="pi pi-filter-slash"
          text
          size="small"
          @click="selected = [...ORDER]"
        />
      </div>

      <DataTable
        :value="visibleTeams"
        size="small"
        scrollable
        scrollHeight="50vh"
        :loading="pending"
        class="text-sm"
      >
        <Column field="teamNumber" header="Nr." style="width: 4rem">
          <template #body="{ data: row }">
            <span class="font-bold">{{ row.teamNumber || '×' }}</span>
          </template>
        </Column>
        <Column field="name" header="Patrulje">
          <template #body="{ data: row }">
            <router-link class="underline" :to="{ name: 'patrulje', params: { teamId: row.teamId } }">
              {{ row.name || 'Patrulje' }}
            </router-link>
            <div class="text-xs text-gray-500">{{ row.group }}</div>
          </template>
        </Column>
        <Column field="status" header="Status" style="width: 9rem">
          <template #body="{ data: row }">
            <Tag
              :value="statusMeta(row.status).label"
              :severity="statusMeta(row.status).severity"
              :icon="statusMeta(row.status).icon"
            />
          </template>
        </Column>
        <Column header="Scannet" style="width: 7rem">
          <template #body="{ data: row }">
            <span v-if="row.scannedAtUts">{{ clock(row.scannedAtUts) }}</span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </Column>
        <!--
          The team's own deadline, next to its own scan, so "for sent" can be read rather than
          taken on trust — and so a patrol still out can be seen running out of time.
        -->
        <Column header="Frist" style="width: 7rem">
          <template #body="{ data: row }">
            <template v-if="row.deadlineUts">
              <span>{{ clock(row.deadlineUts) }}</span>
              <div
                v-if="overrun(row) !== null && overrun(row)! > 0"
                class="text-xs"
                :class="row.scannedAtUts ? 'text-amber-600' : 'text-red-600'"
              >
                +{{ overrun(row) }} min
              </div>
            </template>
            <span v-else class="text-gray-400">—</span>
          </template>
        </Column>
        <!-- Strength on the route: zero is why a team reads as udgået rather than missing. -->
        <Column header="I løbet" style="width: 5rem">
          <template #body="{ data: row }">
            <span :class="row.activeMemberCount === 0 ? 'text-gray-400' : ''">
              {{ row.activeMemberCount }}
            </span>
          </template>
        </Column>
        <!--
          How far the ones still out are, and from which post. This is what separates a patrol
          walking in from one that will not make it — same red tag, opposite response — and it is
          why the contact column gave up its space: the phone number is one click away on the
          patrol page, the distance is nowhere else.
        -->
        <Column header="Afstand">
          <template #body="{ data: row }">
            <template v-if="row.distance">
              <span :class="positionIsStale(row.distance.ts) ? 'text-gray-400' : ''">
                {{ formatDistance(row.distance.meters) }} fra {{ row.distance.checkpointName }}
              </span>
              <div class="text-xs text-gray-500">
                ({{ formatAge(row.distance.ts) }}<template v-if="row.distance.source === 'scan'">, sidste scan</template>)
              </div>
            </template>
            <span v-else class="text-gray-400">—</span>
          </template>
        </Column>
        <template #empty>
          <span class="text-gray-500">Ingen patruljer i det valgte filter.</span>
        </template>
      </DataTable>

      <p class="pt-2 text-xs text-gray-500">
        Viser {{ visibleTeams.length }} af {{ teams.length }} startede patruljer.
      </p>
    </template>

    <template #footer>
      <Button label="Luk" text @click="emit('close')" />
    </template>
  </Dialog>
</template>
