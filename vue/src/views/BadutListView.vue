<script setup>
import { ref, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';

const toast = useToast();

onMounted(() => load())

const badutter = ref([])
const load = async () => {
  try {
    const response = await http.get('/badut');
    badutter.value = response.data.personnel.filter(p => p.paidAmount > 0);
    console.log("badutter", badutter)
  } catch (error) {
    console.log('badut list load failed', error);
  }
}
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
    <h1 class="font-nathejk text-2xl">Gøglere</h1>
    <a href="/api/excel/personnel">Eksport til Excel</a>
    <div class="card" id="badut">
        <DataTable :value="badutter" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true"
            v-model:expandedRows="expandedRows" dataKey="id" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <Column expander />
            <Column field="name" header="Navn"></Column>
            <Column field="group" header="Gruppe / Division"></Column>
            <Column field="korps" header="Korps"></Column>
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
#badut td {
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
