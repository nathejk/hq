<script setup>
import { ref, computed, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';

const props = defineProps({
    teamId: {type: String, required: false},
})

const toast = useToast();

onMounted(() => load())

const patrulje = ref({})
const scans = ref([])

const load = async () => {
  try {
    const response = await http.get('/patrulje/' + props.teamId + '/scans');
    patrulje.value = response.data.team;
    scans.value = response.data.scans;
  } catch (error) {
    console.log('badut list load failed', error);
  }
}
const start = async () => {
  const payload = {
    teamId: props.teamId,
    members: [],
  }
  spejdere.value.forEach(s => payload.members.push({ memberId: s.memberId, name: s.name, phone: s.phone, phoneParent: s.phoneParent, starter: s.starter}))
  try {
    const response = await http.put('/patrulje/' + props.teamId + '/start', payload);
    if (response.status == 200) {
      toast.add({ severity: 'info', summary: 'Patrulje '+ patrulje.value.name + ' startet', detail: 'Videre til foto', life: 3000 });
    } else {
      toast.add({
        closable: true,
        life: 5000,
        severity: 'error',
        summary: 'Kunne ikke starte patrulje',
        detail: 'Kunne ikke starte patrulje',
      });
      console.log('respinse', response)
    }
  } catch (error) {
    toast.add({ severity: 'error', closable: true, life: 5000, summary: 'Kunne ikke starte patruljen', detail: error.message });
    console.log('start patrulje failed', error);
  }
}

const starterCount = computed(() => spejdere.value.filter(s => s.starter).length)

</script>

<template>
    <div class="card !bg-slate-300 pb-3" id="patruljer">

        <DataTable :value="scans" class="!bg-transparent" size="small">
            <Column field="uts" header="Navn"></Column>
            <Column field="scannerPhone" header="Telefon"></Column>
        </DataTable>

        <div class="grid mt-3">
        </div>
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
