<script setup lang="ts">
// The table behind the bingo curve: every patrol on the route against every obligatorisk
// postlinje, tick or cross.
//
// The curve answers "how many"; this answers "which, and where did it go wrong". A cross is
// the interesting cell, so the hover on it says *why* — too late, never seen, or the team
// went home — and gives the two times that decide it: when they came through, and when they
// were due. Without those an operator cannot tell a team that missed by four minutes from
// one that never turned up, and those are not the same phone call.
//
// The green rows are the answer to the question the dialog was opened for, so they are
// sorted to the top by the API rather than left to be hunted for.

import { computed } from 'vue'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { dddhhmm } from '@/composables/datefilters'

const emit = defineEmits<{ close: [] }>()

type CellStatus = 'onTime' | 'late' | 'retired' | 'missing'

interface Cell {
  status: CellStatus
  /** When the team came through. Absent if it never did. */
  scannedAtUts?: number
  /**
   * When it was due. Per team, not per line: on a relative postlinje the clock starts when
   * that team left the line it measures from. Absent when the line keeps no time, or when a
   * relative line's reference scan has not happened yet — there is no deadline to state.
   */
  deadlineUts?: number
}

interface TeamRow {
  teamId: string
  teamNumber: string
  name: string
  group: string
  activeMemberCount: number
  /** Parallel to `lines`, by index. Always the same length, so no cell can be missing. */
  cells: Cell[]
  catchCount: number
  firstCatchUts?: number
  bingo: boolean
}

interface Payload {
  lines: { checkgroupId: string; name: string }[]
  teams: TeamRow[]
  bingoCount: number
}

// dependsOn, each token from the projection that owns the event and not from the names on
// this screen:
//   patrulje        who started
//   spejder         activeMemberCount — what makes a cross "udgået" rather than "mangler"
//   qr              the scans. NOTE the token is `qr`, not `scan`
//   senior          the banditter, whose identity is what makes a scan a catch
//   checkgroup      which lines are obligatoriske, and their schemes
//   checkpoint      the open windows a scan is judged against
//   checkpersonnel  the rota — a scan reaches a post through the scanner's shift, so
//                   correcting a shift moves every tick in that column
const { data, pending, error, refresh } = useLiveResource(
  'bingo:teams',
  async () => {
    const response = await http.get('/bingo/teams')
    return response.data as Payload
  },
  { dependsOn: ['patrulje', 'spejder', 'qr', 'senior', 'checkgroup', 'checkpoint', 'checkpersonnel'] }
)

const detail = computed<Payload | null>(() => data.value ?? null)
const lines = computed(() => detail.value?.lines ?? [])
const teams = computed(() => detail.value?.teams ?? [])

/**
 * What a cell shows. Only `onTime` is a tick; everything else is a cross.
 *
 * Three ways of not being on time, drawn as one cross but hovered as three different
 * sentences — the distinction already exists in resolveTeamStatus and the table would be
 * lying to flatten it. The fallback keeps a status this build has never heard of rendering
 * as itself instead of crashing the table: DataTable types its row slot as `any`, so the
 * template cannot narrow the key.
 */
const CELL_META: Record<CellStatus, { icon: string; klass: string; label: string }> = {
  onTime: { icon: 'pi pi-check', klass: 'text-green-600', label: 'Til tiden' },
  late: { icon: 'pi pi-times', klass: 'text-red-600', label: 'For sent' },
  missing: { icon: 'pi pi-times', klass: 'text-red-600', label: 'Ikke set' },
  retired: { icon: 'pi pi-minus', klass: 'text-gray-400', label: 'Udgået' }
}

const cellMeta = (status: string) => CELL_META[status as CellStatus] ?? { icon: 'pi pi-question', klass: 'text-gray-400', label: status }

/**
 * The cell at a column index, tolerating a row that has fewer than the API promised.
 *
 * The server builds `cells` exactly as long as `lines`, so this never fires — but the two
 * arrive in one payload and are indexed positionally, and a template that dereferences
 * `undefined.status` takes the whole dialog down rather than showing one odd cell.
 */
const cellAt = (row: TeamRow, index: number): Cell => row.cells?.[index] ?? { status: 'missing' as CellStatus }

/**
 * The hover text: the verdict, then the times that justify it.
 *
 * Both times whenever both exist, because "23.04 (frist 23.00)" is a fact an operator can
 * act on and "for sent" on its own is not. A line with no deadline says so rather than
 * omitting the word, so a blank is never mistaken for a missing lookup.
 */
const cellTitle = (cell: Cell) => {
  const parts = [cellMeta(cell.status).label]
  if (cell.scannedAtUts) parts.push(`scannet ${dddhhmm(new Date(cell.scannedAtUts * 1000))}`)
  parts.push(cell.deadlineUts ? `frist ${dddhhmm(new Date(cell.deadlineUts * 1000))}` : 'ingen frist')
  return parts.join(' · ')
}

/** The green background. Computed by the API so the highlight cannot drift from the graph. */
const rowClass = (row: TeamRow) => (row.bingo ? 'bg-green-50' : '')

const bingoCount = computed(() => detail.value?.bingoCount ?? 0)
</script>

<template>
  <Dialog visible modal maximizable :style="{ width: '90rem' }" :breakpoints="{ '1599px': '95vw' }" @update:visible="emit('close')">
    <template #header>
      <div class="inline-flex items-center gap-2 text-2xl">
        <i class="fas fa-fw fa-th"></i>
        <h1 class="font-nathejk">Bingo-patruljer</h1>
      </div>
    </template>

    <div v-if="pending && !detail" class="text-sm text-gray-500">Henter …</div>
    <div v-else-if="error && !detail" class="text-sm text-red-600">
      Kunne ikke hente listen.
      <Button label="Prøv igen" text size="small" @click="refresh" />
    </div>

    <template v-if="detail">
      <p class="text-sm text-gray-500 pb-3">
        <strong>{{ bingoCount }}</strong> af {{ teams.length }} startede patruljer har bingo: igennem alle {{ lines.length }} obligatoriske postlinjer til tiden og aldrig fanget af banditter. De står øverst med grøn baggrund. Hold musen over et felt for tider.
      </p>

      <DataTable :value="teams" :rowClass="rowClass" dataKey="teamId" size="small" scrollable scrollHeight="flex" stripedRows sortMode="single">
        <Column field="teamNumber" header="Nr." style="width: 4rem" frozen>
          <template #body="{ data: row }">
            <span class="font-bold">{{ row.teamNumber || '—' }}</span>
          </template>
        </Column>
        <Column field="name" header="Patrulje" style="min-width: 14rem" frozen>
          <template #body="{ data: row }">
            <router-link class="underline" :to="{ name: 'patrulje', params: { teamId: row.teamId } }">
              {{ row.name }}
            </router-link>
            <div class="text-xs text-gray-500">{{ row.group }}</div>
          </template>
        </Column>

        <!--
          One column per obligatorisk postlinje, in the route's own order. Narrow and centred:
          the whole point is that a row can be read across at a glance, and eight columns of
          text would not fit beside the names.
        -->
        <Column v-for="(line, index) in lines" :key="line.checkgroupId" :header="line.name" style="width: 4.5rem">
          <template #body="{ data: row }">
            <span class="inline-flex justify-center w-full" :class="cellMeta(cellAt(row, index).status).klass" v-tooltip.top="cellTitle(cellAt(row, index))" :aria-label="`${line.name}: ${cellTitle(cellAt(row, index))}`">
              <i :class="cellMeta(cellAt(row, index).status).icon"></i>
            </span>
          </template>
        </Column>

        <Column field="catchCount" header="Fangster" sortable style="width: 6rem">
          <template #body="{ data: row }">
            <span :class="row.catchCount > 0 ? 'text-red-600 font-medium' : 'text-gray-400'">
              {{ row.catchCount }}
            </span>
          </template>
        </Column>
        <Column field="firstCatchUts" header="Først fanget" sortable style="width: 9rem">
          <template #body="{ data: row }">
            <span v-if="row.firstCatchUts" v-tooltip.top="dddhhmm(new Date(row.firstCatchUts * 1000))">
              {{ dddhhmm(new Date(row.firstCatchUts * 1000)) }}
            </span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </Column>
        <!-- Zero here is udgået, which is why a row can be all crosses without anybody erring. -->
        <Column header="I løbet" style="width: 5rem">
          <template #body="{ data: row }">
            <span :class="row.activeMemberCount === 0 ? 'text-gray-400' : ''">{{ row.activeMemberCount }}</span>
          </template>
        </Column>

        <template #empty>
          <span class="text-gray-500">Ingen patruljer er startet endnu.</span>
        </template>
      </DataTable>

      <p v-if="lines.length === 0" class="pt-2 text-sm text-gray-600">Ingen postlinjer er markeret obligatoriske, så ingen patrulje kan få bingo. Markér de obligatoriske postlinjer på Poster.</p>
      <p class="pt-2 text-xs text-gray-500">En scanning tælles på en post, hvis scanneren stod på en registreret vagt der på det tidspunkt — mangler vagterne, kan scanningerne ikke placeres, og felterne bliver røde.</p>
    </template>

    <template #footer>
      <Button label="Luk" text @click="emit('close')" />
    </template>
  </Dialog>
</template>
