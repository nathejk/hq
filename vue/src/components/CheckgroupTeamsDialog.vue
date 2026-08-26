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
        <!-- Strength on the route: zero is why a team reads as udgået rather than missing. -->
        <Column header="I løbet" style="width: 5rem">
          <template #body="{ data: row }">
            <span :class="row.activeMemberCount === 0 ? 'text-gray-400' : ''">
              {{ row.activeMemberCount }}
            </span>
          </template>
        </Column>
        <!--
          The contact, as a tel: link. This is what turns the missing list from a report into
          something an operator can act on without looking the patrol up somewhere else.
        -->
        <Column header="Kontakt">
          <template #body="{ data: row }">
            <a v-if="row.contactPhone" :href="`tel:${row.contactPhone}`" class="underline">
              {{ row.contactPhone }}
            </a>
            <span v-else class="text-gray-400">—</span>
            <div v-if="row.contactName" class="text-xs text-gray-500">{{ row.contactName }}</div>
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
