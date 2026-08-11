<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { FilterMatchMode } from '@primevue/core/api'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { severityLabel, severityTagSeverity, formatDateTime } from '@/composables/sos'

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
