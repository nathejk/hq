<script setup>
import { ref, toRaw, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { http } from '@/plugins/axios';
import { useGlobalState } from '@/composables/globalstate'

//import { NodeService } from './service/NodeService';
const { setYearSlug } = useGlobalState()

const toast = useToast();

const years = ref([
    {
        slug: '2025',
        headline: 'Modstandsbevægelsen',
        route: { start: 'Tølløse', destination: 'Køge'},
        description: 'Spejderne skal hjælpe modstandsbevægelsen',
        signupStart: '2025-05-02T20:25:00Z',
        startdate: '2025-09-18',
        enddate: '2025-09-20',
    },
    {
        slug: '2024',
        headline: 'Skattejagt',
        route: { start: 'Lundby', destination: 'Glumsø'},
        description: 'Guldhornene er blevet stjålet',
        signupStart: '2024-05-02T20:25:00Z',
        startdate: '2024-09-20',
        enddate: '2024-09-22',
    },
    {
        slug: '2023',
        headline: 'Ex Nihilo',
        route: { start: 'Kalundborg', destination: 'Stenlille'},
        description: 'En ond kult vil gennemføre et ritual som vil frembringe dæmonen Moloch der vil lægge verden i lænker.',
        signupStart: '2023-05-02T20:25:00Z',
        startdate: '2023-09-15',
        enddate: '2023-09-17',
    },
]);
const load = async () => {
  try {
    const response = await http.get('/years');
    years.value = response.data.years
  } catch (error) {
    console.log('year list load failed', error);
  }
}
const save = async () => {
  try {
    const pad = (n) => String(n).padStart(2, '0')
    const formatLocal = (d) => [d.getFullYear(), pad(d.getMonth() + 1), pad(d.getDate())].join('-')

    const [ds, de] = year.value.daterange
    year.value.dateStart = formatLocal(ds)
    year.value.dateEnd = formatLocal(de)

    await http.patch(`/year/` + year.value.slug, year.value)
    toast.add({ severity: 'success', summary: 'Nathejk ' + year.value.slug, detail: 'Ændringer gemt', life: 3000 });
    close()
    load()
  } catch (error) {
    toast.add({ severity: 'error', summary: 'Opdatering fejlede', detail: error, life: 3000 });
  }
}
const add = async () => {
    const slug = prompt("Årstal")
  try {
    await http.post(`/year/`+slug, { slug: slug })
    toast.add({ severity: 'success', summary: 'Nathejk ' + slug, detail: 'Oprettet', life: 3000 });
    load()
  } catch (error) {
    toast.add({ severity: 'error', summary: 'Oprettelse fejlede', detail: error, life: 3000 });
  }
}
const year = ref({  daterange: [new Date(), new Date()] })
const visible  = ref(false);
const edit = async payload => {
    const  raw = toRaw(payload)
    if (typeof raw.dateStart == 'string') {
        raw.dateStart = (raw.dateStart.length > 0) ? new Date(raw.dateStart) : null
    }
    if (typeof raw.dateEnd == 'string') {
        raw.dateEnd = (raw.dateEnd.length > 0) ? new Date(raw.dateEnd) : null
    }

    if (raw.dateEnd != null && raw.dateStart != null) {
        raw.daterange = [raw.dateStart, raw.dateEnd ]
    }
    year.value = raw
    visible.value = true
}
const pick = async payload => {
    setYearSlug( payload.slug)
}
const close = async () => visible.value = false


onMounted(() => load());
</script>

<template>
    <div class="flex justify-between py-2">
        <h1 class="font-nathejk text-2xl">Alle Nathejk</h1>
        <Button icon="pi pi-plus" label="Ny udgave" size="small" @click="add" />
    </div>
    <div class="card">
<Accordion :value="['0']" multiple>
    <AccordionPanel v-for="year in years" :value="year.slug">
        <AccordionHeader>
        <div class="flex flex-wrap px-1 items-center gap-4 w-full ">
            <span class="text-2xl font-nathejk">{{ year.slug }}</span>
           <div class="flex-1 flex flex-col">
               <span class="font-medium">{{ year.headline }}</span>
               <span class="text-xs font-thin text-slate-900 upper">{{ year.cityDeparture }} &#8594; {{ year.cityDestination }}</span>
           </div>
           <span class="font-bold pr-3">&times;&times; patruljer</span>
       </div>
        </AccordionHeader>
        <AccordionContent>
            <p class="m-0">{{ year.description }}</p>
            <div class="flex gap-2 pt-2">
                <Button icon="pi pi-pencil" label="Ret" size="small" @click="edit(year)" />
                <Button icon="pi pi-history" label="Vælg" size="small" @click="pick(year)" />
            </div>
        </AccordionContent>
    </AccordionPanel>
</Accordion>

    </div>

    <Dialog v-model:visible="visible" maximizable modal :style="{ width: '50rem' }" >
        <template #header>
            <h1 class="font-nathejk text-2xl">Nathejk {{ year.slug }}</h1>
        </template>
        <div class="pt-3 grid gap-y-2">
            <FloatLabel variant="on">
                <InputText id="cityfrom" v-model="year.cityDeparture" autocomplete="off" class="w-full" />
                <label for="cityfrom">Startby</label>
            </FloatLabel>
            <FloatLabel variant="on">
                <InputText id="citydest" v-model="year.cityDestination" autocomplete="off" class="w-full" />
                <label for="citydest">Startby</label>
            </FloatLabel>
            <FloatLabel variant="on">
                <DatePicker v-model="year.daterange" inputId="daterange" showIcon iconDisplay="input" selectionMode="range" :manualInput="false" dateFormat="d MM yy" fluid />
                <label for="daterange">Start- / slutdatoer</label>
            </FloatLabel>
            {{ year.daterange }}
            <FloatLabel variant="on">
                <InputText id="headline" v-model="year.headline" autocomplete="off" class="w-full" />
                <label for="headline">Overskrift</label>
            </FloatLabel>
            <FloatLabel variant="on">
                <Textarea id="description" v-model="year.description" rows="5" style="resize: none" class="w-full"  />
                <label for="description">Kort intern beskrivelse</label>
            </FloatLabel>
        </div>
        <div class="flex gap-2 pt-2">
            <Button icon="pi pi-send" label="Gem" size="small" @click="save" />
            <Button icon="pi pi-times" label="Afbryd" size="small" @click="close" />
        </div>
    </Dialog>
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
