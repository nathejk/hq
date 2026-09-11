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
import PatruljePhoto from '@/components/PatruljePhoto.vue';

const toast = useToast();

// Live, cached list. Two consequences worth knowing:
//
//  - Returning to this page renders from cache with no request, so navigating
//    away and back is instant rather than a round trip.
//  - dependsOn: ['patrulje'] is a dependency on the entity *type*, not on the ids
//    currently loaded. That is deliberate and it is what makes a newly signed-up
//    patrol appear: a new row has an id this client has never seen, so an
//    instance-keyed dependency could never catch it.
//
// Everything the endpoint returns is shown. This used to drop nameless rows, which
// was a client-side guess at "not a real signup" — the server decides that now
// (paid and numbered, the same set the front-page count and the Excel export use),
// so filtering again here would only hide teams the rest of HQ counts.
const { data, pending, error } = useLiveResource(
  'patrulje:list',
  async () => {
    const response = await http.get('/patrulje');
    return response.data.teams;
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

// The status tag for a row. Four states an operator cares about, by colour:
//
//   aktiv     (green)      racing right now — started, with members still on the route
//   betalt    (blue)       fully paid, ready, not yet started
//   delbetalt (yellow)     accepted and paid for the minimum, but an added member is
//                          not yet paid for ("delvis betalt", shortened)
//   udgået    (light grey) started, but nobody is left racing — discontinued
//
// udgået is not a stored status: a started team whose activeMemberCount has reached
// zero *is* discontinued (there is no event for it — see the spejderstatus
// projection), so it is derived here rather than read from a column.
const statusTag = (row) => {
    if (row.signupStatus === 'STARTED') {
        return row.activeMemberCount > 0
            ? { value: 'aktiv', severity: 'success' }
            : { value: 'udgået', severity: 'secondary' };
    }
    if (row.signupStatus === 'PAID') return { value: 'betalt', severity: 'info' };
    if (row.signupStatus === 'SEMIPAID') return { value: 'delbetalt', severity: 'warn' };
    // The list is filtered to participants, so nothing else is expected here; show the
    // raw status rather than hiding a state we did not anticipate.
    return { value: (row.signupStatus || '').toLowerCase(), severity: 'contrast' };
};
</script>

<template>
    <div class="card" id="patruljer">
    <a href="/api/excel/patrulje">Eksport til Excel</a>
        <!-- Sorted by team number by default: it is the identifier organizers use
             to talk about a patrulje, and it reflects acceptance order. Numeric
             strings compare correctly — the comparator is an Intl.Collator with
             numeric: true — so 9 precedes 10 rather than following it. -->
        <DataTable :value="patruljer" :loading="pending" sortMode="single" sortField="teamNumber" :sortOrder="1" :stripedRows="true" :filters="filters"
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
            <!-- The patrol's photograph. One year-wide request feeds every row (see
                 usePatruljeCovers), and rows with no photograph render nothing rather
                 than a placeholder. -->
            <Column header="Foto" class="w-16">
                <template #body="{data}">
                    <PatruljePhoto :teamId="data.teamId" :teamName="data.name" size="sm" />
                </template>
            </Column>
            <Column field="teamNumber" header="#" sortable></Column>
            <Column field="name" header="Navn" sortable>
                <template #body="{data}">
                    <router-link :to="{  name: `patrulje`, params: { teamId: data.teamId } }">{{ data.name }}</router-link>
                </template>
            </Column>
            <Column field="group" header="Gruppe / Division" sortable></Column>
            <Column field="korps" header="Korps"></Column>
            <Column field="memberCount" header="Spejdere" dataType="numeric" ></Column>
            <Column field="signupStatus" header="Status" sortable>
                <template #body="{data}">
                    <Tag v-bind="statusTag(data)" />
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
