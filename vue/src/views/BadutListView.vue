<script setup>
import { ref, computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';
import PositionIndicator from '@/components/PositionIndicator.vue';

const toast = useToast();

// dependsOn, taken from what the personnel projection actually consumes
// (go/nathejk/table/personnel/consumer.go) rather than from the table's name:
// the subject token is `gøgler`, not `personnel`. `bandit` carries armNumber
// assignments, which the expanded row shows. `order`/`payment` are here because
// the Status column is paidAmount, which the query derives by joining payments to
// orders — so a payment landing must refresh this list even though no gøgler event
// occurred.
const { data, pending, error } = useLiveResource(
  'badut:list',
  async () => {
    const response = await http.get('/badut');
    return (response.data.personnel || []).filter((p) => p.paidAmount > 0);
  },
  { dependsOn: ['gøgler', 'bandit', 'order', 'payment'] },
);

const badutter = computed(() => data.value ?? []);

watch(error, (err) => {
  if (!err) return;
  console.log('badut list load failed', err);
  toast.add({ severity: 'error', summary: 'Kunne ikke hente gøglere', life: 5000 });
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
    <h1 class="font-nathejk text-2xl">Gøglere ({{ badutter.length }})</h1>
    <a href="/api/excel/personnel">Eksport til Excel</a>
    <div class="card" id="badut">
        <DataTable :value="badutter" :loading="pending" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true"
            v-model:expandedRows="expandedRows" dataKey="id" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <Column expander />
            <!--
              The glyph rides in the name cell rather than a column of its own: a column would be
              mostly empty, and the fact belongs to the person rather than being a field of theirs.
              `data.id` is the userId, which is the same id space telemetry reports under.
            -->
            <Column field="name" header="Navn">
                <template #body="{data}">
                    <span class="inline-flex items-center gap-1">
                        {{ data.name }}
                        <PositionIndicator :person-id="data.id" />
                    </span>
                </template>
            </Column>
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
