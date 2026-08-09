<script setup>
import { ref, computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';
import PatruljeEmbeddedView, {
   rewardLevel,
   rewardLevels,
} from '@/views/PatruljeEmbeddedView.vue';
import PatruljeActiveView from '@/views/PatruljeActiveView.vue';

const toast = useToast();

// Live, cached list. Two consequences worth knowing:
//
//  - Returning to this page renders from cache with no request, so navigating
//    away and back is instant rather than a round trip.
//  - dependsOn: ['patrulje'] is a dependency on the entity *type*, not on the ids
//    currently loaded. That is deliberate and it is what makes a newly signed-up
//    patrol appear: a new row has an id this client has never seen, so an
//    instance-keyed dependency could never catch it.
const { data, pending, error } = useLiveResource(
  'patrulje:list',
  async () => {
    const response = await http.get('/patrulje');
    return response.data.teams.filter((p) => p.name != '');
  },
  { dependsOn: ['patrulje'] },
);

const patruljer = computed(() => data.value ?? []);

// Keep the previous failure behaviour: the composable surfaces the error rather
// than swallowing it, so the view decides what the operator sees.
watch(error, (err) => {
  if (!err) return;
  console.log('patrulje list load failed', err);
  toast.add({
    severity: 'error',
    summary: 'Kunne ikke hente patruljer',
    life: 5000,
  });
});
const filters = ref({
    'global': {value: null, matchMode: FilterMatchMode.CONTAINS},
});
const selectedValue = ref(null);
const expandedRows = ref([]);
const onRowExpand = (event) => {
 console.log(expandedRows.value)
 //   toast.add({ severity: 'info', summary: 'Row Group Expanded', detail: 'Value: ' + event.data, life: 3000 });
};
const onRowCollapse = (event) => {
  //  toast.add({ severity: 'success', summary: 'Row Group Collapsed', detail: 'Value: ' + event.data, life: 3000 });
};

const getSeverity = (status) => {
    switch (status) {
        case 'unqualified', 'PAY':
            return 'danger';

        case 'qualified', 'STARTED':
            return 'success';

        case 'new', 'PAID':
            return 'info';

        case 'negotiation':
            return 'warn';

        case 'renewal':
            return null;
    }
};
</script>

<template>
    <div class="card" id="patruljer">
    <a href="/api/excel/patrulje">Eksport til Excel</a>
        <DataTable :value="patruljer" :loading="pending" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :filters="filters"
            v-model:expandedRows="expandedRows" dataKey="teamId" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <template #header>
        <div class="flex flex-wrap gap-2 items-center justify-between">
            <h1 class="font-nathejk text-2xl">Patruljer ({{ patruljer.length }})</h1>
            <IconField>
                <InputIcon>
                    <i class="pi pi-search" />
                </InputIcon>
                <InputText v-model="filters['global'].value" placeholder="Search..." />
            </IconField>
        </div>
            </template>
            <Column expander />
            <Column field="teamNumber" header="#" sortable></Column>
            <Column field="name" header="Navn" sortable>
                <template #body="{data}">
                    <router-link :to="{  name: `patrulje`, params: { teamId: data.teamId } }">{{ data.name }}</router-link>
                </template>
            </Column>
            <Column field="group" header="Gruppe / Division" sortable></Column>
            <Column field="korps" header="Korps"></Column>
            <Column field="memberCount" header="Spejdere" dataType="numeric" ></Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="data.paidAmount/100" :severity="getSeverity(data.signupStatus)" />
                </template>
            </Column>
            <Column field="date" header="Date"></Column>
            <template #expansion="{data}">
                <PatruljeEmbeddedView v-if="data.signupStatus != 'STARTED'" :teamId="data.teamId" />
                <PatruljeActiveView v-else :teamId="data.teamId" />
            </template>
        </DataTable>
    </div>
</template>

<style>
#patruljer td {
    padding: 0.25rem 0.75rem;
}
#patruljer a:hover {
    color: #0000cc;
    text-decoration:underline;
}
@media (min-width: 1024px) {
  .about {
    min-height: 100vh;
    display: flex;
    align-items: center;
  }
}
</style>
