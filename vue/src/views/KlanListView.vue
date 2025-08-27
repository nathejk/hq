<script setup>
import { ref, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';

onMounted(() => load())
const klans = ref([])

const load = async () => {
  try {
    const response = await http.get('/klan');
    klans.value = response.data.teams.filter(k => k.paidAmount > 0);
    console.log("læams", klans)
  } catch (error) {
    console.log('klan list load failed', error);
  }
}
const selectedValue = ref(null);
const expandedRowGroups = ref(['1','2','3','4','5']);
const toast = useToast();
const onRowGroupExpand = (event) => {
 console.log(expandedRowGroups.value)
 //   toast.add({ severity: 'info', summary: 'Row Group Expanded', detail: 'Value: ' + event.data, life: 3000 });
};
const onRowGroupCollapse = (event) => {
  //  toast.add({ severity: 'success', summary: 'Row Group Collapsed', detail: 'Value: ' + event.data, life: 3000 });
};
const calculateCustomerTotal = (lok) => {
    let total = 0;
    if (klans.value) {
        for (let klan of klans.value) {
            if (klan.lok === lok) {
                total++;
            }
        }
    }

    return total;
};
const calculateMemberCount = (lok) => {
    let total = 0;
    if (klans.value) {
        for (let klan of klans.value) {
            if (klan.lok === lok) {
                total+=klan.memberCount;
            }
        }
    }

    return total;
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
const lokoptions = [
    { label: 'LOK 1', value:1 },
    { label: 'LOK 2', value:2 },
    { label: 'LOK 3', value:3 },
    { label: 'LOK 4', value:4 },
    { label: 'LOK 5', value:5 },
    { label: 'Intet', value:null },
];
const updateLok = async (e, o) => {
  try {
  console.log(e.value)
    await http.patch(`/klan/${o.data.id}`, {
      lok: ''+e.value.value,
    });
  } catch (error) {
    console.log('udate lok failed', error);
  }
    klans.value.map(t => {
        t.lok = (t.id == o.data.id) ? e.value.value : t.lok
    })
}
</script>

<template>
    <h1 class="font-nathejk text-2xl">Banditter</h1>
    <a href="/api/excel/klan">Eksport til Excel</a>
    <div class="card">
        <DataTable v-model:expandedRowGroups="expandedRowGroups" :value="klans"  editMode="cell" tableStyle="min-width: 50rem"
                expandableRowGroups rowGroupMode="subheader" groupRowsBy="lok" @rowgroup-expand="onRowGroupExpand" @rowgroup-collapse="onRowGroupCollapse"
                sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" rowGroupHeader="bg-sky-600"
                :pt="{
        _root: 'border border-primary rounded-xl p-4',
        cell: 'bg-sky-600',
        rowGroupHeader: { class: 'bg-sky-600' },
        header: (options) => ({
            id: 'myPanelHeader',
            style: {
                'user-select': 'none'
            },
            class: ['flex items-center justify-between text-primary font-bold bg-sky-600']
        }),
        rowGroupHeader: (options) => ({
            id: 'myPanelHeader',
            style: {
                'user-select': 'none'
            },
            class: ['bg-sky-600 text-slate-600']

        }),
        xcontent: { class: 'text-primary-700 dark:text-primary-200 mt-4' },
        xtitle: 'text-xl',
        xtoggler: () => 'bg-primary text-primary-contrast hover:text-primary hover:bg-primary-contrast'
    }">
    <template #groupheader="row" style="background:yellow" :pt="{class:'bg-sky-600'}">
                    <span class="align-middle ml-2 leading-normal"><span class="font-bold" v-if="row.data.lok > 0">LOK {{ row.data.lok }}</span><span v-else>ingen placering</span></span>
                    <div class="float-right"><span class="font-bold">{{ calculateCustomerTotal(row.data.lok) }}</span><span class="pl-1">klaner</span></div>
                    <div class="float-right"><span class="font-bold">{{ calculateMemberCount(row.data.lok) }}</span><span class="pl-1 pr-5">banditter</span></div>
            </template>
            <Column dataType="numeric" header="LOK">
                <template #body="row">
                    {{ row.data.lok || '&times;' }}
                </template>
                <template #editor="row">
                    <FloatLabel>
    <Select v-model="row.data.lok" inputId="dd-city" :options="lokoptions" optionLabel="label" variant="filled" @change="updateLok($event, row)" />
    <label for="dd-city">LOK</label>
</FloatLabel>

                </template>
            </Column>
            <Column field="name" header="Klan" style="width: 20%"></Column>
            <Column field="group" header="Gruppe / Division" style="width: 20%">
                <!--template #body="slotProps">
                    <div class="flex items-center gap-2">
                        <img alt="flag" src="https://primefaces.org/cdn/primevue/images/flag/flag_placeholder.png" :class="`flag flag-${slotProps.data.country.code}`" style="width: 24px" />
                        <span>{{ slotProps.data.country.name }}</span>
                    </div>
                </template-->
            </Column>
            <Column field="korps" header="Korps" style="width: 20%"></Column>
            <Column field="memberCount" header="Seniore" style="width: 20%"></Column>
            <Column field="status" header="Status" style="width: 20%">
                <template #body="slotProps">
                    <Tag :value="slotProps.data.status" :severity="getSeverity(slotProps.data.status)" />
                </template>
            </Column>
            <Column field="date" header="Date" style="width: 20%"></Column>
        </DataTable>
    </div>
</template>

<style>
@media (min-width: 1024px) {
  .about {
    min-height: 100vh;
    display: flex;
    align-items: center;
  }
}
</style>
