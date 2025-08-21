<script setup>
import { ref, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';

const props = defineProps({
    teamId: {type: String, required: false},
})

const toast = useToast();

onMounted(() => load())

const patrulje = ref({})
const spejdere = ref([])
const payments = ref([])
const load = async () => {
  try {
    const response = await http.get('/patrulje/' + props.teamId);
    console.log("patrulje", response.data)
    patrulje.value = response.data.team;
    spejdere.value = response.data.members;
    payments.value = response.data.payments;
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

        case 'negotiation', 'requested':
            return 'warn';

        case 'renewal':
            return null;
    }
};
const linkToSignUp = () => {
    window.open("http://tilmelding.nathejk.dk/patrulje/" + patrulje.value.id, '_blank')
}
</script>

<template>
    <div class="card" id="patruljer">
        <h1 class="font-nathejk text-2xl">&times;&times;&times; - &times; {{ patrulje.name }}</h1>

        <Button label="Tilmelding" icon="pi pi-external-link" iconPos="right" @click="linkToSignUp" />

        <DataTable :value="spejdere" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :filters="filters"
            v-model:expandedRows="expandedRows" dataKey="id" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <Column expander />
            <Column field="name" header="Navn" sortable></Column>
            <Column field="phone" header="Telefon" sortable></Column>
            <Column field="phoneParent" header="Kontaktperson"></Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag value="ikke startet" :severity="getSeverity(data.status)" />
                </template>
            </Column>
            <template #expansion="{data}">
                <div class="">
                    {{ data }}
                </div>
            </template>
        </DataTable>

        <h1 class="font-nathejk text-2xl mt-5">Betalinger</h1>
        <DataTable :value="payments" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" >
            <Column field="createdAt" header="Tidspunkt" sortable></Column>
            <Column field="amount" header="Beløb" sortable></Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="data.status" :severity="getSeverity(data.status)" />
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
