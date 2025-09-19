<script>
export const rewardLevels = [
  { name: 'Paid', slug: 'paid', icon: 'pi pi-play text-green-500' },
  { name: 'On hold', slug: 'hold', icon: 'pi pi-pause text-yellow-400' },
  { name: "Don't pay", slug: 'nopay', icon: 'pi pi-stop text-red-400' },
];

export const rewardLevel = (slug) => {
  return ""
};
</script>
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
const spejdere = ref([])

const load = async () => {
  try {
    const response = await http.get('/patrulje/' + props.teamId);
    patrulje.value = response.data.team;
    spejdere.value = response.data.members;
    spejdere.value.map(s => s.starter = true)
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

        <DataTable :value="spejdere" class="!bg-transparent" size="small">
            <Column field="name" header="Navn">
                <template #body="{data}">
                    <span :class="{'line-through':!data.starter}">{{ data.name }}</span>
                </template>
            </Column>
            <Column field="phone" header="Telefon">
                <template #body="{data}">
                    <InputText v-if="data.starter" type="text" v-model="data.phone" variant="filled" />
                </template>
            </Column>
            <Column field="phoneParent" header="Kontaktperson">
                <template #body="{data}">
                    <InputText v-if="data.starter" type="text" v-model="data.phoneParent" variant="filled" />
                </template>
            </Column>
            <Column field="status" header="Starter">
                <template #body="{data}">
                    <ToggleSwitch v-model="data.starter">
                        <template #handle="{ checked }">
                            <i :class="['!text-xs pi', { 'pi-check': checked, 'pi-times': !checked }]" />
                        </template>
                    </ToggleSwitch>
                </template>
            </Column>
        </DataTable>

        <div class="grid mt-3">
            <Button label="Start patrulje" :badge="String(starterCount)" :disabled="starterCount < 3" class="justify-self-end" icon="pi pi-check" iconPos="right" raised @click="start" />
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
