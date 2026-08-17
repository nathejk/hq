<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { FilterMatchMode } from '@primevue/core/api'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { severityLabel, severityTagSeverity, formatDateTime } from '@/composables/sos'
import { useConnectionState } from '@/composables/useConnectionState'

const router = useRouter()
const toast = useToast()

// Live, cached list.
//
// dependsOn is the entity **type** `sos`, not any instance: a case created by
// another operator has an id this client has never seen, so an instance-keyed
// dependency could never make it appear. All three SOS projections publish on
// NATHEJK.{year}.sos.{id}.*, so one token covers the case, its timeline and its
// team associations.
const { data, pending, error } = useLiveResource(
  'sos:list',
  async () => {
    const response = await http.get('/sos')
    return response.data as { open: SosCase[]; closed: SosCase[] }
  },
  { dependsOn: ['sos'] },
)

interface SosCase {
  id: string
  headline: string
  description: string
  status: 'open' | 'closed'
  severity: '' | 'green' | 'yellow' | 'red'
  assigneeSectionSlug: string
  createdAt: string
  lastActivityAt: string
}

const open = computed<SosCase[]>(() => data.value?.open ?? [])
const closed = computed<SosCase[]>(() => data.value?.closed ?? [])

// --- I vores varetægt (PRD 006) ---
//
// The number that has to reach zero before the organisers can go home. Event-wide rather
// than per case, and permanently on screen, because a member we are responsible for is
// our problem whether or not anybody opened a case about them.
//
// dependsOn is the entity **type** `spejder`: a member whose id this client has never
// seen must still move the number, and a count has no id to key on at all.
const { data: careData, error: careError } = useLiveResource(
  'member:care',
  async () => {
    const response = await http.get('/members/care')
    return response.data.care as {
      total: number
      byStatus: Record<string, number>
      oldestWaitingAt: string | null
    }
  },
  { dependsOn: ['spejder'] },
)

// The three in-care states, in lifecycle order, so the breakdown reads as the journey a
// member is on. The server includes each at zero, so the row count never changes shape.
const careBreakdown = computed(() => {
  const by = careData.value?.byStatus ?? {}
  return [
    { slug: 'waiting', label: 'Venter', count: by.waiting ?? 0 },
    { slug: 'transit', label: 'I bil', count: by.transit ?? 0 },
    { slug: 'sheltered', label: 'På HQ', count: by.sheltered ?? 0 },
  ]
})

// **A wrong number here is worse than no number.** If the screen cannot reach the server
// — or the read model may be mid-rebuild after a restart, until PRD 005's boot gate ships
// — say so rather than render a figure the operator would trust. Reuses the existing
// connection state rather than inventing a second signal.
const { isDisconnected } = useConnectionState()
const careUnavailable = computed(() => isDisconnected.value || !!careError.value || !careData.value)

// Deliberately no alarm on oldestWaitingAt: nothing in this feature resolves a `waiting`
// member — the car and shelter interfaces do not exist — so an alarm here would fire for
// everybody and stay firing. It belongs to the dispatch dashboard PRD (task 082).

watch(error, (err) => {
  if (!err) return
  toast.add({ severity: 'error', summary: 'Kunne ikke hente nødråb', life: 5000 })
})

// Section labels for the Tildelt column. Loaded from the same endpoint the
// Organisation page uses, and live for the same reason the list is: a section
// renamed while a case is open should not keep showing its old name.
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

const sectionLabels = computed<Record<string, string>>(() => {
  const labels: Record<string, string> = {}
  for (const section of orgData.value?.sections ?? []) labels[section.slug] = section.label
  return labels
})

// A section that has been deleted keeps its slug on the case rather than the
// assignment vanishing — so it is shown as such rather than as a blank cell.
const assigneeLabel = (slug: string) => {
  if (!slug) return ''
  return sectionLabels.value[slug] ?? `${slug} (slettet sektion)`
}

const filters = ref({ global: { value: null, matchMode: FilterMatchMode.CONTAINS } })

const openCase = (id: string) => router.push({ name: 'sos', params: { id } })
const newCase = () => router.push({ name: 'sos-new' })
</script>

<template>
  <div class="card" id="sos-list">
    <div class="flex flex-wrap gap-2 items-center justify-between mb-4">
      <h1 class="font-nathejk text-2xl">Nødtelefon</h1>

      <!--
        I vores varetægt — the go-home number. Placed in the header rather than in a card
        of its own so it is visible without opening a case, which is the whole point of
        counting it event-wide.
      -->
      <div class="flex items-center gap-2 rounded border px-3 py-1"
           :class="careUnavailable ? 'border-gray-300 bg-gray-50' : 'border-amber-300 bg-amber-50'">
        <span class="text-sm text-gray-700">I vores varetægt</span>
        <span v-if="careUnavailable" class="text-sm italic text-gray-500">ingen forbindelse</span>
        <template v-else>
          <span class="font-nathejk text-xl">{{ careData?.total ?? 0 }}</span>
          <span class="text-sm text-gray-600">
            <template v-for="(b, i) in careBreakdown" :key="b.slug">
              <span v-if="i > 0"> · </span>{{ b.label }} {{ b.count }}
            </template>
          </span>
        </template>
      </div>
      <div class="flex gap-2 items-center">
        <IconField>
          <InputIcon><i class="pi pi-search" /></InputIcon>
          <InputText v-model="filters['global'].value" placeholder="Søg i sager..." />
        </IconField>
        <Button label="Ny sag" icon="pi pi-plus" @click="newCase" />
      </div>
    </div>

    <!--
      Two tables rather than one grouped table: open and closed cases are read for
      different reasons — what needs handling now, versus what happened earlier —
      and an operator scanning the open list should never have a closed case in it.
    -->
    <h2 class="font-nathejk text-xl mt-2 mb-1">Åbne sager ({{ open.length }})</h2>
    <DataTable
      :value="open"
      :loading="pending"
      :filters="filters"
      :stripedRows="true"
      dataKey="id"
      selectionMode="single"
      @row-click="openCase($event.data.id)"
    >
      <template #empty>Ingen åbne nødråb</template>
      <Column field="headline" header="Overskrift" sortable />
      <Column field="createdAt" header="Oprettet" sortable>
        <template #body="{ data }">{{ formatDateTime(data.createdAt) }}</template>
      </Column>
      <Column field="lastActivityAt" header="Sidst opdateret" sortable>
        <template #body="{ data }">{{ formatDateTime(data.lastActivityAt) }}</template>
      </Column>
      <Column field="severity" header="Prioritet" sortable>
        <template #body="{ data }">
          <Tag
            v-if="data.severity"
            :value="severityLabel(data.severity)"
            :severity="severityTagSeverity(data.severity)"
          />
        </template>
      </Column>
      <Column field="assigneeSectionSlug" header="Tildelt" sortable>
        <template #body="{ data }">{{ assigneeLabel(data.assigneeSectionSlug) }}</template>
      </Column>
    </DataTable>

    <h2 class="font-nathejk text-xl mt-6 mb-1">Lukkede sager ({{ closed.length }})</h2>
    <DataTable
      :value="closed"
      :loading="pending"
      :filters="filters"
      :stripedRows="true"
      dataKey="id"
      selectionMode="single"
      @row-click="openCase($event.data.id)"
    >
      <template #empty>Ingen nødråb fundet</template>
      <Column field="headline" header="Overskrift" sortable />
      <Column field="createdAt" header="Oprettet" sortable>
        <template #body="{ data }">{{ formatDateTime(data.createdAt) }}</template>
      </Column>
      <Column field="lastActivityAt" header="Sidst opdateret" sortable>
        <template #body="{ data }">{{ formatDateTime(data.lastActivityAt) }}</template>
      </Column>
      <Column field="severity" header="Prioritet" sortable>
        <template #body="{ data }">
          <Tag
            v-if="data.severity"
            :value="severityLabel(data.severity)"
            :severity="severityTagSeverity(data.severity)"
          />
        </template>
      </Column>
      <Column field="assigneeSectionSlug" header="Tildelt" sortable>
        <template #body="{ data }">{{ assigneeLabel(data.assigneeSectionSlug) }}</template>
      </Column>
    </DataTable>
  </div>
</template>

<style>
#sos-list td {
  padding: 0.25rem 0.75rem;
}
#sos-list tr {
  cursor: pointer;
}
</style>
