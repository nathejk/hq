<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import Textarea from 'primevue/textarea'

import DayTimePicker from '@/components/DayTimePicker.vue'
import { dddhhmm, hhmm } from '@/composables/datefilters'

const props = withDefaults(
  defineProps<{
    checkgroupId?: string
    checkgroups?: array
  }>(),
  {
    checkgroupId: '',
    checkgroups: []
  }
)
const emit = defineEmits<{
  saved: [id: String]
  deleted: [id: String]
  canceled: null
}>()

const toast = useToast()
//const users = ref([])
//const user = (id) => users.value.filter((s) => s.id == id).shift()
const checkgroup = ref({ checkpoints: [] })
const year = ref({})
const checkpoints = ref([])
const selectedScheme = ref({})

onMounted(() => load())

const load = async () => {
  if (props.checkgroupId.length == 0) return
  try {
    const rsp = await http.get('/checkgroup/' + props.checkgroupId, {
      withCredentials: true
    })
    if (rsp.status == 200) {
      //users.value = rsp.data.users
      checkgroup.value = rsp.data.checkgroup
      year.value = rsp.data.year
      const ss = {}
      ss[checkgroup.value.scheme + (checkgroup.value.scheme == 'relative' ? ':' + checkgroup.value.relativeCheckgroupId : '')] = true
      selectedScheme.value = ss
      checkpoints.value = rsp.data.checkpoints
      checkpoints.value.map((cp) => {
        cp.openUntil = new Date(cp.openUntil)
        cp.openFrom = new Date(cp.openFrom)
        cp.address += ''
      })
      //const scanners = rsp.data.scanners.map(s => ({ start: new Date(s.start), end: new Date(s.end), name:user(s.userId)?.name, id:s.userId, cpIndex: s.ControlIndex }))
      //checkpoints.value.map(cp => cp.scanners = scanners.filter(s => s.cpIndex == cp.index))
    }
  } catch (error) {
    console.log('error happend', error)
    throw new Error(error.response.data)
  }
}
const create = async () => {
  try {
    const rsp = await http.post('/checkgroup', {}, { withCredentials: true })
    if (rsp.status == 200) {
      return rsp.data.checkgroupId
    }
    throw new Error('server responded with status: ' + rsp.statusCode)
  } catch (error) {
    throw new Error(error.message)
  }
}
const save = async () => {
  const payload = {
    name: checkgroup.value.name,
    showOnMap: !!checkgroup.value.showOnMap,
    mandatory: !!checkgroup.value.mandatory,
    scheme: Object.keys(selectedScheme.value ?? []).shift(),
    checkpoints: []
  }
  let cp
  while ((cp = checkpoints.value.shift())) {
    //cp.plus = parseInt(cp.plus)
    //cp.minus = parseInt(cp.minus)
    if (cp.created) {
      delete cp.id
    }
    payload.checkpoints.push(cp)
  }
  let checkgroupId = props.checkgroupId
  try {
    console.log('saving', props.checkgroupId, props.checkgroupId.length)
    if (checkgroupId.length == 0) {
      checkgroupId = await create()
    }
    const rsp = await http.put('/checkgroup/' + checkgroupId, payload, {
      withCredentials: true
    })
    if (rsp.status != 200) {
      throw new Error(rsp.data)
    }
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: 'Postlinje blev ikke gemt',
      detail: error,
      life: 3000
    })
    return
  }
  toast.add({
    severity: 'success',
    summary: 'Postlinje gemt',
    detail: 'OK',
    life: 3000
  })

  emit('saved', checkgroupId)
}
const discard = async () => {
  try {
    const rsp = await http.delete('/checkgroup/' + props.checkgroupId, {
      withCredentials: true
    })
    if (rsp.status != 200) {
      throw new Error(rsp.data)
    }
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: 'Postlinje blev ikke fjernet',
      detail: error,
      life: 3000
    })
    return
  }
  toast.add({
    severity: 'success',
    summary: 'Postlinje blev fjernet',
    detail: 'OK',
    life: 3000
  })

  emit('deleted', props.checkgroupId)
}
const cancel = () => emit('canceled')

/*
const usergroups = computed(() => {
  console.log('inside groups')
  var groups = {}
  for (const user of users.value) {
    console.log('user', user, user.additionals, user.additionals?.department)
    groups[user.additionals?.department] = []
  }
  for (const user of users.value) groups[user.additionals?.department].push(user)
  console.log('groups', groups)
  return groups
})
*/

//const dateHHmm = v => moment(value).format('HH:mm')

const addCheckpoint = () =>
  checkpoints.value.push({
    id: new Date().getTime(),
    created: true,
    name: 'Post',
    dateRange: { startDate: new Date(), endDate: new Date() },
    scanners: []
  })
const deleteCheckpoint = (id) => checkpoints.value.map((cp) => (cp.deleted = cp.id == id ? true : cp.deleted))
//const addScanner = (p) => p.scanners.push({ dateRange: p.dateRange, userId: '' })
//const deleteScanner = (ctrl, i) => ctrl.scanners.splice(i, 1)

//const edit = ref({ controls: [] })

/*const dtEditCellStyling = {
  column: {
    bodycell: ({ state }) => ({
      class: ['group/cell border-0', { '!py-0': state['d_editing'] }]
    })
  },
  bodyRow: () => ({ class: 'group/row' })
}*/

const menu = ref(null)

const checkpointMenuItems = ref([
  /*
  {
    label: 'Refresh',
    icon: 'pi pi-refresh'
  },
  {
    label: 'Tilføj scanner',
    icon: 'pi pi-mobile',
    command: () => menu.checkpoint.scanners.push({ name: 'Skannemand' })
  },
  {
    separator: true
    },*/
  {
    label: 'Slet post',
    icon: 'pi pi-times',
    command: () => deleteCheckpoint(menu.value.checkpoint.id)
  }
])

const toggle = (event, checkpoint) => {
  menu.value.checkpoint = checkpoint
  menu.value.toggle(event)
}

//const editingRows = ref([])
/*
const onRowEditSave = (event) => {
  let { newData, index } = event
  //products.value[index] = newData;
}
const getStatusLabel = (status) => {
  switch (status) {
    case 'INSTOCK':
      return 'success'

    case 'LOWSTOCK':
      return 'warn'

    case 'OUTOFSTOCK':
      return 'danger'

    default:
      return null
  }
  }*/
//const dt = ref(new Date())
/*const ddddhhmm = (ts) => {
  const d = new Date(ts)
  const dayNames = ['Søndag', 'Mandag', 'Tirsdag', 'Onsdag', 'Torsdag', 'Fredag', 'Lørdag']
  return dayNames[d.getDay()] + ' ' + d.getHours() + ':' + d.getMinutes()
}*/
//const exkeys = ref({ post: true })
//var dt = new Date().getTime();
const schemes = computed(() => {
  return [
    {
      key: 'fixed',
      label: 'Faste åbningstider',
      icon: 'pi pi-fw pi-flag'
    },
    {
      key: 'relative',
      label: 'Relative åbningstider',
      icon: 'pi pi-fw pi-stopwatch',
      selectable: false,
      children: props.checkgroups.map((cg) => ({
        key: 'relative:' + cg.id,
        label: 'Relativ til ' + cg.name,
        icon: 'pi pi-fw pi-map-marker'
      }))
    },
    {
      key: 'none',
      label: 'Ingen åbningstider',
      icon: 'pi pi-fw pi-exclamation-circle'
    }
  ]
})
const scheme = computed(() =>
  Object.keys(selectedScheme.value ?? [])
    .shift()
    ?.split(':')
    .shift()
)
//const address = ref('')
const closeInplace = (expr, next) => next()
</script>

<template>
  <form class="small">
    <div class="grid gap-2">
      <div class="flex flex-row px-1 gap-4 w-full items-center select-none">
        <div class="basis-1/6">Postlinjenavn</div>
        <div class="basis-5/6">
          <InputText id="checkgroupname" class="w-full" v-model="checkgroup.name" />
        </div>
      </div>
      <div class="flex flex-row px-1 gap-4 w-full items-center select-none">
        <div class="basis-1/6">Åbningstider</div>
        <div class="basis-5/6">
          <TreeSelect v-model="selectedScheme" :options="schemes" :expandedKeys="{ relative: true }" placeholder="Vælg" class="w-full" />
        </div>
      </div>
      <div class="flex flex-row px-1 gap-4 w-full select-none">
        <div class="basis-1/6">Obligatorisk</div>
        <ToggleSwitch v-model="checkgroup.mandatory" class="scale-75 origin-center">
          <template #handle="{ checked }">
            <i :class="['!text-xs pi', { 'pi-check': checked, 'pi-times': !checked }]" />
          </template>
        </ToggleSwitch>
      </div>
      <div class="flex flex-wrap px-1 gap-4 w-full select-none">
        <div class="basis-1/6">Vis på kort</div>
        <ToggleSwitch v-model="checkgroup.showOnMap" class="scale-75 origin-center">
          <template #handle="{ checked }">
            <i :class="['!text-xs pi', { 'pi-check': checked, 'pi-times': !checked }]" />
          </template>
        </ToggleSwitch>
      </div>
    </div>

    <Fieldset legend="Poster" class="min-h-32">
      <Panel
        v-for="cp in checkpoints"
        :key="cp.id"
        toggleable
        :collapsed="true"
        class="checkpoint mb-2"
        :pt="{
          header: { class: '!bg-slate-50 group/header' },
          headerIcon: { class: 'hover:border hover:border-gray-300' }
        }"
      >
        <template #header>
          <div class="flex items-center gap-2 group/title">
            <Inplace @open="cp._name = cp.name">
              <template #display>
                <div class="inline-flex gap-3 items-center">
                  <span class="font-bold" :class="{ 'line-through': cp.deleted }">{{ cp.name }}</span>
                  <i class="pi pi-pencil text-transparent group-hover/header:text-slate-300 group-hover/title:!text-slate-600" />
                </div>
              </template>
              <template #content="{ closeCallback }">
                <span class="inline-flex items-center gap-2">
                  <InputText v-model="cp.name" autofocus />
                  <Button icon="pi pi-check" text severity="success" @click="closeCallback" />
                  <Button icon="pi pi-times" text severity="danger" @click="closeInplace((cp.name = cp._name), closeCallback)" />
                </span>
              </template>
            </Inplace>

            <Inplace v-if="scheme == 'fixed'" @open="(cp._openFrom = cp.openFrom), (cp._openUntil = cp.openUntil)">
              <template #display>
                <div class="inline-flex gap-3 items-center">
                  <span class="font-thin">{{ dddhhmm(cp.openFrom) }}<span class="px-3">&rarr;</span>{{ hhmm(cp.openUntil) }}</span>
                  <i class="pi pi-pencil text-transparent group-hover/header:text-slate-300 group-hover/title:!text-slate-600" />
                </div>
              </template>
              <template #content="{ closeCallback }">
                <span class="inline-flex items-center gap-2">
                  <DayTimePicker v-model:value="cp.openFrom" :offset="year.dateStart" />
                  <span>&rarr;</span>
                  <DayTimePicker v-model:value="cp.openUntil" :offset="year.dateStart" />

                  <Button icon="pi pi-check" text severity="success" @click="closeCallback" />
                  <Button icon="pi pi-times" text severity="danger" @click="closeInplace((cp.openFrom = cp._openFrom) && (cp.openUntil = cp._openUntil), closeCallback)" />
                </span>
              </template>
            </Inplace>

            <Inplace v-if="scheme == 'relative'" @open="cp._openDuration = cp.openDuration">
              <template #display>
                <div class="inline-flex gap-3 items-center">
                  <span class="font-thin"><span class="px-3">&plus;</span>{{ cp.openDuration }} minutter</span>
                  <i class="pi pi-pencil text-transparent group-hover/header:text-slate-300 group-hover/title:!text-slate-600" />
                </div>
              </template>
              <template #content="{ closeCallback }">
                <span class="inline-flex items-center gap-2">
                  <InputNumber v-model="cp.openDuration" prefix="&plus; " :min="0" showButtons />
                  <span class="px-3">minutter</span>

                  <Button icon="pi pi-check" text severity="success" @click="closeCallback" />
                  <Button icon="pi pi-times" text severity="danger" @click="closeInplace((cp.openDuration = cp._openDuration), closeCallback)" />
                </span>
              </template>
            </Inplace>
          </div>
        </template>
        <template #footer>
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-2">
              <Button icon="pi pi-user" rounded text></Button>
              <Button icon="pi pi-map-marker" severity="secondary" rounded text></Button>
            </div>
            <span class="text-surface-500 dark:text-surface-400">Updated 2 hours ago</span>
          </div>
        </template>
        <template #icons>
          <Button icon="pi pi-cog" severity="secondary" rounded text @click="toggle($event, cp)" />
        </template>
        <div class="grid grid-cols-3 gap-4 pt-3">
          <div class="...">
            <FloatLabel variant="on">
              <Textarea id="on_label0" v-model="cp.address" rows="5" class="w-full" style="resize: none" />
              <label for="on_label0">Adresse</label>
            </FloatLabel>
          </div>
          <div class="col-span-2">
            <FloatLabel variant="on">
              <Textarea id="on_label1" v-model="cp.description" rows="5" class="w-full" style="resize: none" />
              <label for="on_label1">Postbeskrivelse</label>
            </FloatLabel>
          </div>
        </div>
      </Panel>
      <Menu ref="menu" id="config_menu" :model="checkpointMenuItems" popup />
      <Button icon="pi pi-plus" @click="addCheckpoint" size="small" label="Tilføj post" />
    </Fieldset>

    <div class="flex justify-end gap-2 pt-2">
      <Button icon="pi pi-times" label="Afbryd" severity="secondary" size="small" @click="cancel" />
      <Button icon="pi pi-trash" label="Slet" severity="danger" size="small" @click="discard" />
      <Button icon="pi pi-send" label="Gem" size="small" @click="save" />
    </div>
  </form>
</template>

<style scoped>
/*
.checkpoint :deep(.p-panel-header-actions button:hover) {
  @apply border border-gray-300 transition-colors;
}*/

.no-last-border .p-datatable-tbody {
  background-color: yellow !important;
  display: none !important;
}
.no-last-border > tbody > tr:last-child > td {
  border-bottom: none !important;
}
:deep(.p-datatable .p-datatable-tbody > tr.p-datatable-row-expanded > td) {
  border-bottom: none !important;
}
/* remove border of expanded row */
.p-datatable .p-datatable-tbody > tr.p-datatable-row-expanded > td {
  border-bottom: none !important;
  border-bottom-color: transparent !important;
  box-shadow: none !important;
}
:deep(.p-datatable .p-datatable-tbody > tr.p-datatable-row-expansion > td) {
  border-top: none !important;
}
.p-datatable .p-datatable-tbody > tr.p-datatable-row-expansion > td {
  border-top: none !important;
}
.controlpoint {
  background: #eee;
}
.controlpoint:hover {
  background: #ddd;
}
.controlpoint ~ .row:hover {
  background: #ddd;
}
</style>
