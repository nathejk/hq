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
import VerifiedMark from '@/components/VerifiedMark.vue';

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

// Which of the two numbers on a member's row the member has verified themselves in the
// hej app (hej, PRD 015). A verified number is one question the counter does not have to
// ask at check-in, so the shield is there to be skipped past — hence a mark on the field
// itself rather than a column of its own.
//
// The check is "is the value on screen still the number that was proven?", not "has this
// member ever verified anything". The field is editable, and the moment an operator
// corrects it the verification no longer vouches for what it says — so the shield must
// disappear as they type. That is why the server sends the verified number rather than a
// boolean.
//
// Compared on digits only: the register holds numbers as they were typed ("12 34 56 78",
// "+45 12345678"), the events carry them normalized, and formatting is not a difference an
// operator should have to see.
//
// Nor is a country code. hej verifies through the SMS gateway and publishes what it dialled,
// so a verified Danish number arrives international — 4523102001 for the 23102001 the patrol
// leader typed. Both are the same phone, and an operator shown an unmarked field would ask a
// question that was already answered. So the two are reduced to their last 8 digits when both
// are at least that long, which is what "same Danish number, differently prefixed" means;
// anything shorter (a partially typed number, a foreign one) is compared in full rather than
// guessed at.
const digits = (value) => String(value ?? '').replace(/\D/g, '')
const isVerified = (value, verified) => {
  const proven = digits(verified)
  const shown = digits(value)
  if (proven === '' || shown === '') return false
  if (proven.length >= 8 && shown.length >= 8) {
    return proven.slice(-8) === shown.slice(-8)
  }
  return proven === shown
}

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
                    <InputGroup v-if="data.starter">
                        <InputText type="text" v-model="data.phone" variant="filled" />
                        <VerifiedMark :verified="isVerified(data.phone, data.verifiedPhone)" />
                    </InputGroup>
                </template>
            </Column>
            <Column field="phoneParent" header="Kontaktperson">
                <template #body="{data}">
                    <InputGroup v-if="data.starter">
                        <InputText type="text" v-model="data.phoneParent" variant="filled" />
                        <VerifiedMark :verified="isVerified(data.phoneParent, data.verifiedPhoneParent)" />
                    </InputGroup>
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
