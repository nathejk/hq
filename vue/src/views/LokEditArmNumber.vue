<script setup>
import { ref, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
//import draggable from "vuedraggable";
//import Menu from 'primevue/menu';

const props = defineProps({
  lok: { type: String, required: false }
})
const emit = defineEmits({
  saved: null
})

const toast = useToast()
onMounted(() => load())

const bandits = ref([])
const teams = ref([])
const team = (id) => teams.value.filter((t) => id == t.id).shift()

const load = async () => {
  try {
    const response = await http.get('/lok/' + props.lok)
    teams.value = response.data.teams
    for (const user of response.data.users) {
      bandits.value.push({ id: user.id, armNumber: user.armNumber, name: user.name, team: '' })
    }
    for (const member of response.data.members) {
      bandits.value.push({ id: member.memberId, armNumber: member.armNumber, name: member.name, teamId: member.teamId, team: team(member.teamId).name })
    }
    bandits.value.map((s) => (s.loadedArmNumber = s.armNumber))
  } catch (error) {
    console.log('member list load failed', error)
  }

  //toast.add({ severity: 'success', summary: 'LOK fordeling gemt', detail: 'OK', life: 3000 });
}
const save = async () => {
  const payload = { users: [], members: [] }
  for (const bandit of bandits.value) {
    if (bandit.loadedArmNumber == bandit.armNumber) continue
    if (bandit.teamId) {
      payload.members.push({ id: bandit.id, armNumber: bandit.armNumber })
    } else {
      payload.users.push({ id: bandit.id, armNumber: bandit.armNumber })
    }
  }
  try {
    const response = await http.patch(`/lok/${props.lok}`, payload)
    if (response.status != 200) {
      throw new Error(response.data)
    }
  } catch (error) {
    toast.add({ severity: 'error', summary: 'Banditnumre blev ikke opdateret', detail: error, life: 3000 })
    return
  }
  toast.add({ severity: 'success', summary: 'Banditnumre blev opdateret', detail: 'OK', life: 3000 })
  emit('saved')
}
const dirtyCount = () => bandits.value.filter((s) => s.loadedArmNumber != s.armNumber).length

const onCellEditComplete = (event) => {
  let { data, newValue, field } = event
  switch (field) {
    case 'quantity':
    case 'price':
      if (isPositiveInteger(newValue)) data[field] = newValue
      else event.preventDefault()
      break

    default:
      if (!newValue) break
      if (newValue.trim().length > 0) data[field] = newValue
      else event.preventDefault()
      break
  }
}
const dtEditCellStyling = {
  column: {
    bodycell: ({ state }) => ({ class: ['group/cell', { '!py-0': state['d_editing'] }] })
  },
  bodyRow: () => ({ class: 'group/row' })
}
</script>

<template>
  <p>Banditter skal have tildelt banditnummer før de kan scanne spejderpatruljer.</p>
  <DataTable :value="bandits" editMode="cell" @cell-edit-complete="onCellEditComplete" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :pt="dtEditCellStyling">
    <Column field="armNumber" header="#" style="width: 15%" class="cursor-pointer">
      <template #editor="{ data, field }">
        <InputText v-model="data[field]" fluid autofocus onfocus="this.select()" />
      </template>
      <template #body="{ data, field }">
        <span :class="['group-hover/cell:text-slate-900', { 'text-slate-300': !data[field] }]">{{ data[field] || '000' }}</span>
        <span class="pl-3 hidden group-hover/row:inline"><i class="pi pi-pencil group-hover/cell:text-slate-900 text-slate-300"></i></span>
      </template>
    </Column>
    <Column field="name" header="Bandit"></Column>
    <Column field="team" header="Klan"></Column>
  </DataTable>
  <div class="grid mt-3">
    <Button label="Opdater" :badge="String(dirtyCount())" :disabled="dirtyCount() == 0" class="justify-self-end" icon="pi pi-check" iconPos="right" raised @click="save" />
  </div>
</template>

<style></style>
