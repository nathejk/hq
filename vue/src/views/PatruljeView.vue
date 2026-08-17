<script setup>
import { ref, computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';
import { daymonthhhmm } from '@/composables/datefilters';

const props = defineProps({
    teamId: {type: String, required: false},
})

const toast = useToast();

// Keyed by team, so several patrols can be cached side by side and opening a
// previously visited one is instant.
//
// The team itself is depended on by *instance*, so another patrol's edit does not
// refetch this page. Its members and orders cannot be: a spejder event names the
// member id and an order event names the order id, never the team, so there is no
// instance to key on — and a newly added member has an id this client has never
// seen. Those are therefore type-level, which does mean any scout or payment change
// anywhere revalidates an open patrol page. That is one small request on a page an
// operator has deliberately opened, and the alternative is a member list that
// silently stops updating.
//
// 'sos' was added for the "Kontakt med nødtelefon" card (PRD 001): the cases arrive
// in this same payload, so the card needs no resource of its own — only this token,
// which is also type-level because a case opened for this patrol has an id this
// client has never seen.
const { data, pending, error } = useLiveResource(
  `patrulje:detail:${props.teamId}`,
  async () => {
    const response = await http.get('/patrulje/' + props.teamId);
    return {
      team: response.data.team || {},
      members: response.data.members || [],
      orders: response.data.orders || [],
      sosCases: response.data.sosCases || [],
      config: response.data.config || {},
    };
  },
  { dependsOn: [`patrulje:${props.teamId}`, 'spejder', 'order', 'payment', 'sos'] },
);

const patrulje = computed(() => data.value?.team ?? {});
const spejdere = computed(() => data.value?.members ?? []);
const orders = computed(() => data.value?.orders ?? []);
const sosCases = computed(() => data.value?.sosCases ?? []);
const config = computed(() => data.value?.config ?? {});

// --- member lifecycle status (PRD 006) ---
//
// Labels come from the backend, never a map in this view. Two screens show these strings
// and they are persisted values, so a local copy is how the two drift apart until one says
// "waiting" to an operator at 3am.
const memberStatusLabel = (slug) => {
  const found = (config.value.memberStatuses ?? []).find((s) => s.slug === slug);
  // "Ikke startet" is the honest reading of an absent status: the member exists on the
  // roster and the race has not claimed them yet. Before PRD 006 this column said that
  // for *everybody*, regardless of status — which is what made the bug invisible.
  return found?.label ?? (slug ? slug : 'Ikke startet');
};

const memberStatusSeverity = (slug) => {
  switch (slug) {
    case 'racing':
    case 'finished':
      return 'success';
    case 'waiting':
      return 'danger';
    case 'transit':
    case 'sheltered':
      return 'warn';
    case 'reunited':
    case 'released':
      return 'secondary';
    default:
      return 'contrast';
  }
};

watch(error, (err) => {
  if (!err) return;
  console.log('patrulje load failed', err);
  toast.add({ severity: 'error', summary: 'Kunne ikke hente patruljen', life: 5000 });
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

const linkToSignUp = () => {
    window.open("http://tilmelding.nathejk.dk/patrulje/" + patrulje.value.id, '_blank')
}

const formatAmount = (value, currency) => {
    if (value == null) return ''
    return (value / 100).toLocaleString('da-DK', { style: 'currency', currency: currency || 'DKK' })
}
// Shared rather than parsed here: the order and payment tables serve Go's
// time.Time text form, which Safari refuses to parse. See parseApiDate.
const formatDateTime = (value) => daymonthhhmm(value)
const statusLabel = (status) => (status === 'PAID' ? 'Betalt' : 'Åben')
const statusSeverity = (status) => (status === 'PAID' ? 'success' : 'warn')
</script>

<template>
    <div class="card" id="patruljer">
        <h1 class="font-nathejk text-2xl">{{ patrulje.number || '×' }} - {{ patrulje.name }}</h1>

        <Button label="Tilmelding" icon="pi pi-external-link" iconPos="right" @click="linkToSignUp" />

        <DataTable :value="spejdere" :loading="pending" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :filters="filters"
            v-model:expandedRows="expandedRows" dataKey="id" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <Column expander />
            <Column field="name" header="Navn" sortable></Column>
            <Column field="phone" header="Telefon" sortable></Column>
            <Column field="phoneParent" header="Kontaktperson"></Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="memberStatusLabel(data.status)"
                         :severity="memberStatusSeverity(data.status)" />
                </template>
            </Column>
            <template #expansion="{data}">
                <div class="">
                    {{ data }}
                </div>
            </template>
        </DataTable>

        <!--
          Kontakt med nødtelefon (PRD 001). Hidden entirely when the patrol has never
          called: most never do, and an empty card on every patrol page is noise that
          trains operators to ignore the place where a real incident would appear.
        -->
        <template v-if="sosCases.length">
            <h1 class="font-nathejk text-2xl mt-5">Kontakt med nødtelefon</h1>
            <DataTable :value="sosCases" :stripedRows="true" dataKey="id" selectionMode="single"
                @row-click="$router.push({ name: 'sos', params: { id: $event.data.id } })">
                <Column field="createdAt" header="Oprettet">
                    <template #body="{data}">{{ formatDateTime(data.createdAt) }}</template>
                </Column>
                <Column field="headline" header="Overskrift" />
                <Column field="status" header="Status">
                    <template #body="{data}">
                        <Tag :value="data.status === 'closed' ? 'Afsluttet' : 'Åben'"
                             :severity="data.status === 'closed' ? 'secondary' : 'info'" />
                    </template>
                </Column>
            </DataTable>
        </template>

        <h1 class="font-nathejk text-2xl mt-5">Betalinger</h1>
        <DataTable :value="orders" sortMode="single" sortField="createdAt" :sortOrder="-1" :stripedRows="true" >
            <template #empty>Ingen bestillinger</template>
            <Column field="createdAt" header="Tidspunkt" sortable>
                <template #body="{data}">{{ formatDateTime(data.createdAt) }}</template>
            </Column>
            <Column field="totalAmount" header="Beløb" sortable>
                <template #body="{data}">{{ formatAmount(data.totalAmount, data.currency) }}</template>
            </Column>
            <Column field="paidAmount" header="Betalt">
                <template #body="{data}">{{ formatAmount(data.paidAmount, data.currency) }}</template>
            </Column>
            <Column field="dueAmount" header="Mangler">
                <template #body="{data}">{{ formatAmount(data.dueAmount, data.currency) }}</template>
            </Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
                </template>
            </Column>
        </DataTable>
    </div>
</template>

<style>
#patruljer td {
    padding: 0.25rem 0.75rem;
}
@media (min-width: 1024px) {
  .about {
    min-height: 100vh;
    display: flex;
    align-items: center;
  }
}
</style>
