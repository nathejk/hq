<script setup>
import { ref, computed, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import draggable from 'vuedraggable'
import Menu from 'primevue/menu'
import LokEditArmNumber from '@/views/LokEditArmNumber.vue'

const toast = useToast()

const users = ref([])
const teams = ref([])
const loks = ref([])

const user = (id) => users.value.filter((t) => id == t.id).shift()
const team = (id) => teams.value.filter((t) => id == t.id).shift()
const unassignedUsers = computed(() => {
  let ua = users.value
  for (const lok of loks.value) ua = ua.filter((u) => !lok.users.map((u) => u.id).includes(u.id))
  return ua
})
const unassignedTeams = computed(() => {
  let ua = teams.value
  for (const lok of loks.value) ua = ua.filter((t) => !lok.teams.map((t) => t.id).includes(t.id))
  return ua
})

// Live and cached, but with a guard the read-only pages do not need: this is an
// *editor*. The LOK arrangement lives in `loks` and is only persisted when the
// operator presses "Gem LOKs", so applying an incoming revalidation mid-edit would
// silently throw away their work. Updates are therefore deferred while unsaved
// changes exist, and applied on save.
//
// dependsOn, from the handler (showLoksHandler) and the projections behind it:
//   lok               the assignments being edited
//   klan              the klans listed, and the paidAmount>0 filter below
//   senior            each klan's memberCount is a COUNT over seniors
//   order/payment     paidAmount is derived by joining payments to orders
//   gøgler/friend     the "Banditter" personnel offered as lok/bandit chiefs
//   bandit            armNumber assignments, edited from this page's dialog
const { data: lokData, error: loadError, refresh } = useLiveResource(
  'klan:loks',
  async () => {
    const response = await http.get('/lok')
    return {
      teams: response.data.teams || [],
      users: response.data.users || [],
      loks: response.data.loks || []
    }
  },
  { dependsOn: ['lok', 'klan', 'senior', 'order', 'payment', 'gøgler', 'friend', 'bandit'] }
)

/** Unsaved local edits exist, so incoming data must not overwrite them. */
const dirty = ref(false)

const applyPayload = (payload) => {
  // Rebuilt, never appended to: the previous version pushed into `loks`, which
  // duplicated every LOK if the load ever ran twice — harmless when it only ran
  // on mount, fatal now that revalidation is a thing.
  teams.value = payload.teams.filter((k) => k.paidAmount > 0)
  // Copied: with no LOKs defined, `unassignedUsers` returns this very array, and
  // dragging out of it would splice the value held in the cache.
  users.value = [...payload.users]
  loks.value = payload.loks.map((lok) => ({
    lokId: lok.lokId,
    name: lok.name,
    users: lok.userIds.map((id) => user(id)).filter((u) => u),
    teams: lok.teamIds.map((id) => team(id)).filter((t) => t)
  }))
  dirty.value = false
}

watch(
  lokData,
  (payload) => {
    if (!payload) return
    // Deliberately dropped while dirty rather than queued: saving refetches, so
    // the next apply comes from a payload that already includes our own changes.
    if (dirty.value) return
    applyPayload(payload)
  },
  { immediate: true }
)

watch(loadError, (err) => {
  if (err) console.log('klan list load failed', err)
})

const save = async () => {
  try {
    const payload = { loks: [] }
    for (const lok of loks.value) {
      payload.loks.push({
        lokId: lok.lokId,
        name: lok.name,
        userIds: lok.users.map((u) => u.id),
        teamIds: lok.teams.map((t) => t.id)
      })
    }
    await http.put(`/lok`, payload)
    dirty.value = false
    toast.add({ severity: 'success', summary: 'LOK fordeling gemt', detail: 'OK', life: 3000 })
    // Our own PUT will arrive as a signal too, but ask directly rather than wait.
    void refresh()
  } catch (error) {
    console.log('udate loks failed', error)
  }
}
const add = () => {
  loks.value.push({ name: 'LOK ' + loks.value.length, users: [], teams: [] })
  dirty.value = true
}
/** Any drag, rename or addition means the arrangement on screen is unsaved. */
const markDirty = () => (dirty.value = true)
const deleteLok = async (lok) => {
  loks.value = loks.value.filter((l) => l != lok)
  if (lok.lokId) {
    try {
      const response = await http.delete(`/lok/` + lok.lokId)
      if (response.status != 200) {
        throw new Error(response.data)
      }
    } catch (error) {
      toast.add({ severity: 'error', summary: 'LOK blev ikke fjernet', detail: error, life: 3000 })
      return
    }
  }
  toast.add({ severity: 'success', summary: 'LOK fjernet', detail: 'OK', life: 3000 })
}
const memberCount = (lok) => lok.teams.reduce((sum, team) => sum + (team.memberCount || 0), lok.users.length)

//const klans = ref([])
//const selectedValue = ref(null)
//const expandedRowGroups = ref(['1', '2', '3', '4', '5'])
/*const onRowGroupExpand = (event) => {
  console.log(expandedRowGroups.value)
  //   toast.add({ severity: 'info', summary: 'Row Group Expanded', detail: 'Value: ' + event.data, life: 3000 });
}*/
/*const onRowGroupCollapse = (event) => {
  //  toast.add({ severity: 'success', summary: 'Row Group Collapsed', detail: 'Value: ' + event.data, life: 3000 });
}*/
/*const calculateCustomerTotal = (lok) => {
  let total = 0
  if (klans.value) {
    for (let klan of klans.value) {
      if (klan.lok === lok) {
        total++
      }
    }
  }

  return total
}*/
/*const calculateMemberCount = (lok) => {
  let total = 0
  if (klans.value) {
    for (let klan of klans.value) {
      if (klan.lok === lok) {
        total += klan.memberCount
      }
    }
  }

  return total
}*/
/*const getSeverity = (status) => {
  switch (status) {
    case 'unqualified':
      return 'danger'

    case 'qualified':
      return 'success'

    case 'new':
      return 'info'

    case 'negotiation':
      return 'warn'

    case 'renewal':
      return null
  }
}*/
/*const updateLok = async (e, o) => {
  try {
    console.log(e.value)
    await http.patch(`/klan/${o.data.id}`, {
      lok: '' + e.value.value
    })
  } catch (error) {
    console.log('udate lok failed', error)
  }
  klans.value.map((t) => {
    t.lok = t.id == o.data.id ? e.value.value : t.lok
  })
}*/
/*const saveArmNumbers = async () => {
  const dirty = seniors.value.filter((s) => s.loadedArmNumber != s.armNumber)
}*/
/*const linkToSignUp = (id) => {
  window.open('http://tilmelding.nathejk.dk/klan/' + id, '_blank')
}*/
const lokEditArmNumber = ref('')
const visible = ref(false)
//const seniors = ref([])
const showDialog = async (lokId) => {
  visible.value = true
  lokEditArmNumber.value = lokId
  /*
  try {
    const response = await http.get('/bandit?lok=' + lok);
    seniors.value = response.data.members;
    seniors.value.map(s => s.armNumber = s.armNumber || '000')
    seniors.value.map(s => s.loadedArmNumber = s.armNumber)
  } catch (error) {
    console.log('member list load failed', error);
  }
*/
}
//const dirtyCount = () => seniors.value.filter((s) => s.loadedArmNumber != s.armNumber).length
/*const onCellEditComplete = (event) => {
  let { data, newValue, field } = event

  switch (field) {
    case 'quantity':
    case 'price':
      if (isPositiveInteger(newValue)) data[field] = newValue
      else event.preventDefault()
      break

    default:
      if (newValue.trim().length > 0) data[field] = newValue
      else event.preventDefault()
      break
  }
}*/
/*const dtEditCellStyling = {
  column: {
    bodycell: ({ state }) => ({ class: ['group/cell', { '!py-0': state['d_editing'] }] })
  },
  bodyRow: () => ({ class: 'group/row' })
}*/
const items = ref([
  {
    label: 'Refresh',
    icon: 'pi pi-refresh',
    disabled: true
  },
  {
    label: 'Tildel banditnumre',
    icon: 'pi pi-hashtag',
    disabled: () => !menu.value.lok.lokId,
    command: () => showDialog(menu.value.lok.lokId)
  },
  {
    separator: true
  },
  {
    label: 'Fjern LOK',
    icon: 'pi pi-times',
    command: () => deleteLok(menu.value.lok),
    test: async () => {
      loks.value = loks.value.filter((lok) => lok != menu.value.lok)
      if (menu.value.lok.lokId) {
        try {
          const response = await http.delete(`/lok/` + menu.value.lok.lokId)
          if (response.status != 200) {
            throw new Error(response.data)
          }
        } catch (error) {
          toast.add({ severity: 'error', summary: 'LOK blev ikke fjernet', detail: error, life: 3000 })
          return
        }
      }
      toast.add({ severity: 'success', summary: 'LOK fjernet', detail: 'OK', life: 3000 })
    }
  }
])

const menu = ref()
const toggle = (event, lok) => {
  menu.value.lok = lok
  menu.value.toggle(event)
}
const closeInplace = (expr, next) => next()
</script>

<template>
  <h1 class="font-nathejk text-2xl">Banditter</h1>
  <a href="/api/excel/klan">Eksport til Excel</a>
  <div class="card grid grid-cols-3 gap-2">
    <div class="col-span-2">
      <div class="grid gap-2">
        <Menu ref="menu" id="config_menu" :model="items" popup />
        <Panel toggleable v-for="lok in loks" :key="lok.id">
          <template #header>
            <div class="flex items-center gap-2">
              <Avatar image="https://primefaces.org/cdn/primevue/images/avatar/amyelsner.png" shape="circle" />
              <Inplace @open="lok._ = lok.name">
                <template #display>
                  <span class="font-bold">{{ lok.name }}</span>
                </template>
                <template #content="{ closeCallback }">
                  <span class="inline-flex items-center gap-2">
                    <InputText v-model="lok.name" autofocus />
                    <Button icon="pi pi-check" text severity="success" @click="closeInplace(markDirty(), closeCallback)" />
                    <Button icon="pi pi-times" text severity="danger" @click="closeInplace((lok.name = lok._), closeCallback)" />
                  </span>
                </template>
              </Inplace>
            </div>
          </template>
          <template #_footer>
            <div class="flex flex-wrap items-center justify-between gap-4">
              <div class="flex items-center gap-2">
                <Button icon="pi pi-user" rounded text></Button>
                <Button icon="pi pi-bookmark" severity="secondary" rounded text></Button>
              </div>
              <span class="text-surface-500 dark:text-surface-400">Updated 2 hours ago</span>
            </div>
          </template>
          <template #icons>
            <Button icon="pi pi-cog" severity="secondary" rounded text @click="toggle($event, lok)" arii-haspopup="true" ariia-controls="config_menu" />
          </template>
          <div class="flex justify-between">
            <strong v-if="lok.teams.length == 0 && lok.lokId">Banditchefer</strong>
            <strong v-else>Lokchefer</strong>
            <span class="font-bold sm:ml-8 px-1">{{ lok.users.length }}</span>
          </div>
          <draggable :list="lok.users" handle=".handle" group="chief" item-key="id" @change="markDirty">
            <template #item="{ element }">
              <div class="flex flex-wrap p-1 items-center gap-4 w-full border-b border-slate-200 last:border-0 select-none">
                <i class="w-6 shrink-0 rounded pi pi-bars handle cursor-move"></i>
                <div class="flex-1 flex flex-col">
                  <span class="font-medium">{{ element.name }}</span>
                </div>
              </div>
            </template>
          </draggable>
          <strong v-if="lok.teams.length > 0 || !lok.lokId">Klaner</strong>
          <div v-if="lok.teams.length == 0 && !lok.lokId" class="italic text-slate-500 pl-5">- ingen klaner -</div>
          <draggable :list="lok.teams" handle=".handle" group="a" item-key="id" @change="markDirty">
            <template #item="{ element }">
              <div class="flex flex-wrap px-1 items-center gap-4 w-full border-b border-slate-200 last:border-0 select-none hover:bg-slate-50">
                <i class="w-6 shrink-0 rounded pi pi-bars handle cursor-move"></i>
                <div class="flex-1 flex flex-col">
                  <span class="font-medium">{{ element.name }}</span>
                  <span class="text-xs font-thin text-slate-900 upper">{{ element.group }}</span>
                </div>
                <span class="font-bold sm:ml-8">{{ element.memberCount }}</span>
              </div>
            </template>
          </draggable>
          <div class="flex justify-end" v-if="lok.teams.length > 0">
            <div class="border-t border-slate-200 pl-5">
              <span>Total</span>
              <span class="font-bold sm:ml-8 px-1">{{ memberCount(lok) }}</span>
            </div>
          </div>
        </Panel>
      </div>
      <div class="flex gap-2 pt-2">
        <Button icon="pi pi-plus" label="Tilføj LOK" size="small" @click="add" />
        <Button icon="pi pi-send" label="Gem LOKs" size="small" @click="save" />
        <span v-if="dirty" class="self-center text-sm italic text-amber-700">
          Ugemte ændringer — opdateringer fra andre er sat på pause indtil du gemmer
        </span>
      </div>
    </div>
    <div class="grid gap-2">
      <Panel toggleable header="Tilmeldte hjælpere">
        <draggable :list="unassignedUsers" handle=".handle" group="chief" item-key="id" @change="markDirty">
          <template #item="{ element }">
            <div class="select-none"><i class="pi pi-bars handle cursor-move"></i> {{ element.name }}</div>
          </template>
        </draggable>
      </Panel>
      <Panel toggleable header="Tilmeldte klaner">
        <draggable :list="unassignedTeams" handle=".handle" group="a" item-key="id" @change="markDirty">
          <template #item="{ element }">
            <div class="select-none"><i class="pi pi-bars handle cursor-move"></i> {{ element.name }}</div>
          </template>
        </draggable>
      </Panel>
    </div>
  </div>

  <br />

  <Dialog v-model:visible="visible" maximizable modal :style="{ width: '50rem' }" :breakpoints="{ '1199px': '75vw', '575px': '90vw' }">
    <template #header>
      <h1 class="font-nathejk text-2xl">Banditter</h1>
    </template>
    <LokEditArmNumber :lok="lokEditArmNumber" @saved="visible = false" />
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
tr:hover .lok {
  color: #000099;
  text-decoration: underline;
}
</style>
