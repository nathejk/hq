<script setup>
import { ref, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';

const toast = useToast();

onMounted(() => load())

const patruljer = ref([])
const load = async () => {
  try {
    const response = await http.get('/patrulje');
    patruljer.value = response.data.teams.filter(p => p.name != '');
    console.log("patruljer", patruljer)
  } catch (error) {
    console.log('badut list load failed', error);
  }
}
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
        case 'unqualified':
            return 'danger';

        case 'qualified':
            return 'success';

        case 'new':
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
        <DataTable :value="patruljer" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :filters="filters"
            v-model:expandedRows="expandedRows" dataKey="teamId" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <template #header>
        <div class="flex flex-wrap gap-2 items-center justify-between">
            <h1 class="font-nathejk text-2xl">Patruljer</h1>
            <IconField>
                <InputIcon>
                    <i class="pi pi-search" />
                </InputIcon>
                <InputText v-model="filters['global'].value" placeholder="Search..." />
            </IconField>
        </div>
            </template>
            <Column expander />
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
                    <Tag :value="data.paidAmount/100" :severity="getSeverity(data.status)" />
                </template>
            </Column>
            <Column field="date" header="Date"></Column>
            <template #expansion="{data}">
                <div class="">
                    {{ data }}
                </div>
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
