<script setup>
import { computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';

const props = defineProps({
    teamId: {type: String, required: false},
})

const toast = useToast();

// Read-only, and the one place in the SPA where a *scan* landing is the whole
// point: this is the running patrol's checkpoint trail. Hence the `qr` dependency
// (the subject is NATHEJK.*.qr.*.scanned; `scan` is only the table's name), which
// has to be type-level because a scan event names the qr, never the team.
//
// Cost, measured rather than assumed: this endpoint answers in ~3.4ms with a 5.5KB
// body, and the busiest minute in the existing scan data is 17 scans, so even a
// dozen expanded rows revalidating on every scan is negligible.
const { data, error } = useLiveResource(
  `patrulje:scans:${props.teamId}`,
  async () => {
    const response = await http.get('/patrulje/' + props.teamId + '/scans');
    return {
      team: response.data.team || {},
      scans: response.data.scans || [],
    };
  },
  { dependsOn: [`patrulje:${props.teamId}`, 'qr'] },
);

const patrulje = computed(() => data.value?.team ?? {});
const scans = computed(() => data.value?.scans ?? []);

watch(error, (err) => {
  if (err) console.log('patrulje scans load failed', err);
});
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
