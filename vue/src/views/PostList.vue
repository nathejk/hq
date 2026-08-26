<script setup>
import { ref, watch } from 'vue'
import draggable from 'vuedraggable'

//import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { hhmm } from '@/composables/datefilters'
/*
import ScreeningSwitch from '@/components/ScreeningSwitch.vue'
import MultiSwitch from '@/components/MultiSwitch.vue'
import DayTimePicker from '@/components/DayTimePicker.vue'
*/
import Checkgroup from '@/views/PostlinjeModal.vue'
import CheckgroupPersonnel from '@/views/PostmandskabModal.vue'
import CheckgroupTeamsDialog from '@/components/CheckgroupTeamsDialog.vue'

/*
const checkpointSchemes = [
  { key: 'fixed', label: 'Faste' },
  { key: 'relative', label: 'relative' },
  { key: 'none', label: 'ingen' }
]
*/
//const toast = useToast()
const scanners = ref([])
const checkgroups = ref([])
const startedTeamCount = ref(0)
const checkgroupStats = ref([])
const checkgroup = (id) => checkgroups.value.filter((cg) => cg.id == id).shift()
const edit = ref({ controls: [] })
const expandedPanels = ref(new Set())
const assignedPersonnel = ref([])
const personnel = ref([])
const personName = (userId) => {
  const p = personnel.value.find((p) => p.id === userId)
  return p ? p.name : 'Ukendt'
}

// Live, cached page state.
//
// The fetcher only *shapes* the response; the refs above are filled from it by a
// watcher rather than being computed off the cache, because the drag-to-reorder
// binds `:list="checkgroups"` and splices that array in place. A computed cannot
// be spliced, and the cache holds its value in a shallowRef, so mutating the
// cached array would not re-render either. Local copies keep the reorder working
// while still being replaced wholesale on every revalidation.
//
// dependsOn, all verified against the consumers' Consumes() rather than guessed:
//   checkgroup/checkgroups  the lines themselves, incl. `checkgroups.sorted`,
//                           which is collection-level and carries no id
//   checkpoint              the posts within a line
//   checkpersonnel          the staffing shown per post
//   qr                      scans. NOTE the token is `qr` (NATHEJK.*.qr.*.scanned),
//                           not `scan`: a projection's name is not its subject's.
//                           Needed by the per-line progress meters, which are the
//                           whole point of this page during the race.
//   patrulje                startedTeamCount, from patrulje.*.started
//   spejder                 activeMemberCount, which is what makes a team read as udgået
//                           rather than missing in the four numbers per line. Maintained by
//                           the spejderstatus projection, whose subjects are spejder — so a
//                           withdrawal reaches these counts only through this token, and
//                           without it a patrol that went home would go on being counted as
//                           somebody to chase.
//   crewmember/crew         the names shown for assigned staff. Anyone on a post is a
//                           crew member now (see assignablePersonnel in the API): a
//                           rename, or a newly enrolled member being put on a post,
//                           has to reach this page or it prints "Ukendt" for an id it
//                           cannot resolve. `crew` because a crew *signup* mints the
//                           member — the signedup subject is the crew entity, not
//                           crewmember.
//   section                 the section labels those names are grouped by, from
//                           section.*.added|moved|deleted. Not `sections`: the sort
//                           event changes order, which this page does not show.
//   friend                  the signed-up helpers, still offered for the years that
//                           have them. NOTE the token is the signup's team type,
//                           `friend` — there is no `personnel` token, despite the
//                           projection being called that.
//
// On the cost of depending on scans: measured, not assumed. Peak scan rate in the
// existing data is 17/minute (2025-09-20 13:45) and /checkgroups answers in ~3.5ms
// with a 10KB body, so a revalidation per scan is negligible. Revisit if either
// number changes by an order of magnitude.
const { data: postData, error, refresh } = useLiveResource(
  'post:list',
  async () => {
    const rsp = await http.get('/checkgroups', { withCredentials: true })
    const allPersonnel = rsp.data.assignedPersonnel || []
    const cgs = (rsp.data.checkgroups || []).map((cg) => ({
      ...cg,
      checkpoints: (rsp.data.checkpoints || [])
        .filter((cp) => cp.checkgroupId == cg.id)
        .map((cp) => ({ ...cp, personnel: allPersonnel.filter((p) => p.checkpointId === cp.id) }))
    }))
    return {
      checkgroups: cgs.sort((a, b) => a.sortOrder - b.sortOrder),
      assignedPersonnel: allPersonnel,
      personnel: rsp.data.personnel || [],
      startedTeamCount: rsp.data.startedTeamCount || 0,
      checkgroupStats: rsp.data.checkgroupStats || []
    }
  },
  { dependsOn: ['checkgroup', 'checkgroups', 'checkpoint', 'checkpersonnel', 'qr', 'patrulje', 'spejder', 'crewmember', 'crew', 'section', 'friend'] }
)

watch(
  postData,
  (payload) => {
    if (!payload) return
    // Copied, not aliased: the drag-to-reorder splices `checkgroups`, and doing
    // that to the array the cache holds would quietly reorder the cached value too.
    checkgroups.value = [...payload.checkgroups]
    assignedPersonnel.value = payload.assignedPersonnel
    personnel.value = payload.personnel
    startedTeamCount.value = payload.startedTeamCount
    checkgroupStats.value = payload.checkgroupStats
    scanners.value = [
      { name: 'Søren Sølvmus', id: 1 },
      { name: 'Hanne Sjakke', id: 2 }
    ]
  },
  { immediate: true }
)

watch(error, (err) => {
  if (err) console.log('checkgroups load failed', err)
})
const persistSortOrder = async () => {
  try {
    const ids = checkgroups.value.map((cg) => cg.id)
    await http.put('checkgroups/sorted', { checkgroupIds: ids }, { withCredentials: true })
  } catch (error) {
    console.log('error persisting sort order', error)
  }
}

const addCheckgroup = () => {
  edit.value = {}
  openEdit()
}
const openEdit = () => (addCheckgroupModal.value = true)
const closeEdit = () => (addCheckgroupModal.value = false)
const openPersonnel = () => (addCheckgroupPersonnelModal.value = true)
const closePersonnel = () => (addCheckgroupPersonnelModal.value = false)
const saved = (/*checkgroupId*/) => {
  refresh()
  closeEdit()
  closePersonnel()
}
const deleted = (/*checkgroupId*/) => {
  refresh()
  closeEdit()
}
const addCheckgroupModal = ref(false)
const addCheckgroupPersonnelModal = ref(false)
//const addControl = () => edit.value.checkpoints.push({ name: 'Post', dateRange: { startDate: new Date(), endDate: new Date() }, scanners: [] })
//const deleteControl = (i) => edit.value.controls.splice(i, 1)
//const addScanner = (p) => p.scanners.push({ dateRange: p.dateRange, userId: '' })
//const deleteScanner = (ctrl, i) => ctrl.scanners.splice(i, 1)
/*const deleteControlGroup = () => {
  //this.$store.dispatch("dims/deleteControlGroup", this.edit.controlGroupId);
  deleteCheckgroup(edit.value.controlGroupId)
  closeEdit()
}*/
const deleteCheckgroup = async (id) => {
  if (!window.confirm('Er du sikker på du vil slette denne postlinje?')) return
  try {
    const rsp = await http.delete('/checkgroup/' + id, { withCredentials: true })
    if (rsp.status != 200) {
      throw new Error(rsp.data)
    }
    refresh()
  } catch (error) {
    console.log('error deleting checkgroup', error)
  }
}

const editControlGroup = (id) => {
  //        const cg = this.$store.getters['dims/controlGroup'](id)
  //        this.edit = JSON.parse(JSON.stringify(cg))
  edit.value = checkgroup(id)
  console.log('edit', edit.value)
  //newControlGroup()
  openEdit()
}
const editPersonnel = (id) => {
  edit.value = checkgroup(id)
  openPersonnel()
}
const dt = ref(null)
//const dato = new Date()
//dato.parse("2025-09-19T21:00:00+02:00")
dt.value = new Date('2025-09-19T19:00:00Z')
//const iso = computed(() => new Date(dt.value).toISOString())

//const selectedCategory = ref('')

const meterForCheckgroup = (cgId) => {
  const stats = checkgroupStats.value.find((s) => s.checkgroupId === cgId)
  const onTime = stats ? stats.onTime : 0
  const late = stats ? stats.late : 0
  const retired = stats ? stats.retired : 0
  const missing = stats ? stats.missing : 0
  // `status` is the token the API uses for a team's standing, carried here so a click can open
  // the dialog already filtered to the number that was clicked.
  return [
    { label: 'Til tiden', status: 'onTime', color: '#34d399', text: '#fff', value: onTime, icon: 'pi pi-bolt' },
    { label: 'For sent', status: 'late', color: '#fbbf24', text: '#fff', value: late, icon: 'pi pi-clock' },
    { label: 'Udgåede', status: 'retired', color: '#999', text: '#fff', value: retired, icon: 'pi pi-heart' },
    { label: 'Mangler', status: 'missing', color: 'rgb(229, 231, 235)', text: '#666', value: missing, icon: 'pi pi-eye' }
  ]
}
const meterTotal = (m) => m.reduce((s, o) => s + o.value, 0)
const percent = (f, t) => (t === 0 ? 0 : Math.round((100 * f) / t))

// The line whose teams are being listed, and which of its four numbers was clicked.
//
// The numbers answer "how many", which stops being enough the moment a line is about to close:
// then the only useful question is *which* patruljer are still out and who to ring. The dialog
// is that list, filtered to the number clicked.
const openTeams = ref(null)

const showTeams = (checkgroup, status) => {
  openTeams.value = { checkgroupId: checkgroup.id, name: checkgroup.name, status }
}

const menu = ref(null)
const menuItems = ref([
  {
    label: 'Ret poster',
    icon: 'pi pi-pen-to-square',
    command: () => editControlGroup(menu.value.checkgroup.id)
  },
  {
    label: 'Ret postmandskab',
    icon: 'pi pi-mobile',
    command: () => editPersonnel(menu.value.checkgroup.id)
  },
  {
    separator: true
  },
  {
    label: 'Slet postlinje',
    icon: 'pi pi-times',
    command: () => deleteCheckgroup(menu.value.checkgroup.id)
  }
])

const toggle = (event, checkgroup) => {
  menu.value.checkgroup = checkgroup
  menu.value.toggle(event)
}

//const menu = ref(null)
</script>
<template>
  <h1 class="font-nathejk text-2xl">Postoverblik</h1>
  <div class="card grid grid-cols-3 gap-2">
    <div class="col-span-3">
      <draggable :list="checkgroups" handle=".handle" group="checkgroup" item-key="id" ghost-class="drag-ghost" @update="persistSortOrder">
        <template #item="{ element }">
          <Panel
            toggleable
            :collapsed="!expandedPanels.has(element.id)"
            class="checkpoint mb-2"
            :pt="{
              header: { class: '!bg-slate-50 group/header' },
              headerIcon: { class: 'hover:border hover:border-gray-300' }
            }"
            @update:collapsed="(val) => (val ? expandedPanels.delete(element.id) : expandedPanels.add(element.id))"
          >
            <template #header>
              <div class="flex grow items-center gap-2 group/title">
                <span class="w-1/4 font-bold"><i class="w-6 shrink-0 rounded pi pi-bars handle cursor-move"></i>{{ element.name }}</span>
                <MeterGroup v-if="element.mandatory" class="w-1/2" :value="meterForCheckgroup(element.id)" :max="meterTotal(meterForCheckgroup(element.id))" :pt="{ labelList: { style: expandedPanels.has(element.id) ? '' : 'display: none', class: 'gap-2 text-sm' }, root: { style: 'gap: 0;' } }" />
              </div>
            </template>
            <template #_footer>
              <div class="flex flex-wrap items-center justify-between gap-4">
                <div class="flex items-center gap-2">
                  <Button icon="pi pi-plus" @click="editControlGroup(element.id)" size="small" label="Ret postlinje" />
                  <Button icon="pi pi-mobile" @click="editPersonnel(element.id)" size="small" label="Ret postmandskab" />
                  <!--a role="button" class="mr-3 btn btn-outline-secondary btn-sm" @click="editControlGroup(element.id)"><i class="fas fa-plus"></i> ret postlinje</a>
                  <Button icon="pi pi-user" rounded text></Button>
                  <Button icon="pi pi-map-marker" severity="secondary" rounded text></Button-->
                </div>
                <span class="text-surface-500 dark:text-surface-400">Updated 2 hours ago</span>
              </div>
            </template>
            <template #icons>
              <Button icon="pi pi-cog" severity="secondary" rounded text @click="toggle($event, element)" />
            </template>
            <div class="flex flex-wrap gap-4 pt-3">
              <template v-for="val of meterForCheckgroup(element.id)" :key="val.label">
                <!--
                  Clickable: the number is the question "how many", and the answer an operator
                  needs next is "which ones". Rendered as a button for the keyboard and for the
                  cursor to say so, but styled as the card it already was.
                -->
                <Card
                  class="flex-1 border border-surface shadow-none cursor-pointer hover:border-primary-400"
                  role="button"
                  tabindex="0"
                  v-tooltip.bottom="`Vis ${val.label.toLowerCase()}`"
                  @click="showTeams(element, val.status)"
                  @keydown.enter="showTeams(element, val.status)"
                  @keydown.space.prevent="showTeams(element, val.status)"
                >
                  <template #content>
                    <div class="flex justify-between gap-8">
                      <div class="flex flex-col gap-1">
                        <span class="text-surface-500 dark:text-surface-400 text-sm">{{ val.label }}</span>
                        <span>
                          <span class="font-bold text-lg pr-3">{{ val.value }}</span>
                          <span class="font-thin text-lg">({{ percent(val.value, meterTotal(meterForCheckgroup(element.id))) }}%)</span>
                        </span>
                      </div>
                      <span class="w-8 h-8 rounded-full inline-flex justify-center items-center text-center" :style="{ backgroundColor: `${val.color}`, color: `${val.text}` }">
                        <i :class="val.icon" />
                      </span>
                    </div>
                  </template>
                </Card>
              </template>
            </div>

            <div class="grid grid-cols-3 gap-4 pt-3">
              <Fieldset v-for="cp in element.checkpoints" :key="cp.id" :legend="cp.name" class="pb-3 min-h-32">
                <!-- div class="grid grid-cols-3 controlpoint p-2">
                  <div class="col">{{ cp.name }}</div>
                  <div class="col text-center">{{ dddhhmm(cp.openFrom) }} <i class="far fa-clock mx-1"></i> {{ hhmm(cp.openUntil) }}</div>
                  <div class="col text-right">
                    <small>({{ cp.scanPercent }} %) {{ cp.scanCount }}</small>
                  </div>
                </div-->
                <div class="flex items-center justify-between pb-2 mb-2 border-b border-gray-200">
                  <span class="text-sm font-semibold text-gray-600">Åbningstid</span>
                  <span class="text-sm text-gray-500">{{ hhmm(cp.openFrom) }} – {{ hhmm(cp.openUntil) }}</span>
                </div>
                <p v-if="!cp.personnel || cp.personnel.length === 0" class="italic">Intet postmandskab</p>
                <div v-for="p in cp.personnel" :key="p.id" class="flex items-center justify-between py-1">
                  <div class="flex items-center gap-2">
                    <i class="pi pi-user text-sm text-gray-400"></i>
                    <span class="text-sm">{{ personName(p.userId) }}</span>
                  </div>
                  <span class="text-sm text-gray-500">{{ hhmm(p.start) }} – {{ hhmm(p.end) }}</span>
                </div>
              </Fieldset>
            </div>
          </Panel>
        </template>
      </draggable>
      <Button icon="pi pi-plus" @click="addCheckgroup" size="small" label="Tilføj postlinje" />
      <!--a role="button" @click="addCheckgroup" class="btn btn-outline-secondary"><i class="fas fa-plus"> tilføj postlinje</i></a-->
    </div>
    <Menu ref="menu" id="config_menu" :model="menuItems" popup />
    <Dialog v-model:visible="addCheckgroupModal" modal header="Postlinie" :style="{ width: '70rem' }">
      <template #header>
        <div class="inline-flex items-center justify-center gap-2 text-2xl">
          <i class="fas fa-fw fa-map-marker-alt 2xl"></i>
          <h1 class="font-nathejk">Postlinie</h1>
        </div>
      </template>
      <Checkgroup :checkgroupId="edit.id" :checkgroups="checkgroups" @canceled="closeEdit" @deleted="deleted" @saved="saved" />
    </Dialog>
    <Dialog v-model:visible="addCheckgroupPersonnelModal" modal header="Postmandskab" :style="{ width: '70rem' }">
      <template #header>
        <div class="inline-flex items-center justify-center gap-2 text-2xl">
          <i class="fas fa-fw fa-map-marker-alt 2xl"></i>
          <h1 class="font-nathejk">Postmandskab</h1>
        </div>
      </template>
      <CheckgroupPersonnel :checkgroupId="edit.id" @canceled="closeEdit" @deleted="deleted" @saved="saved" />
    </Dialog>

    <!-- `key` so clicking a number on a different line remounts on that line's resource. -->
    <CheckgroupTeamsDialog
      v-if="openTeams"
      :key="openTeams.checkgroupId"
      :checkgroup-id="openTeams.checkgroupId"
      :name="openTeams.name"
      :initial-status="openTeams.status"
      @close="openTeams = null"
    />
  </div>
</template>

<style scoped>
.drag-ghost {
  position: relative !important;
  overflow: hidden !important;
  height: 3rem !important;
  opacity: 1 !important;
}
.drag-ghost :deep(*) {
  visibility: hidden !important;
}
.drag-ghost::after {
  content: '';
  position: absolute;
  inset: 0;
  visibility: visible;
  background-color: #e5e7eb;
  border: 2px dashed #9ca3af;
  border-radius: 0.375rem;
  pointer-events: none;
}
/*
.xx_checkpoint :deep(.p-panel-header-actions button:hover) {

  @apply border border-gray-300 transition-colors;
}
*/

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

.menu {
  background: #f5f5f5;
  border-bottom: 1px solid #ccc;
  padding: 1rem 0 0;
  /*box-shadow: 0 1px 1px rgb(0 0 0 / 15%)*/
}
.menu a {
  display: inline-block;
  padding: 0 1rem;
  color: #888;
  text-decoration: none;
  text-transform: uppercase;
  font-weight: 100;
  /*line-height: 3.2rem;*/
}
.menu a::after {
  display: block;
  content: attr(title);
  text-transform: uppercase;
  font-weight: 600;
  height: 1px;
  color: transparent;
  overflow: hidden;
  visibility: hidden;
}

.menu a:hover,
.menu a.selected {
  color: #666;
  font-weight: 600;
}
/*
.text-muted {
    background-color: #ececec;
    color: #b9b9b9;
}*/
/*
.cg-passed, .cg-active, .cg-warning, .cg-next {
  padding-left:10px;
  padding-right:10px;
}*/

.cg-passed {
  background: #f5f5f5;
}
.cg-passed,
.cg-passed .btn-edit {
  color: #ccc !important;
}
.cg-passed .btn-edit,
.cg-active .btn-edit,
.cg-warning .btn-edit,
.cg-next .btn-edit {
  display: none;
}
.cg-passed:hover .btn-edit,
.cg-active:hover .btn-edit,
.cg-warning:hover .btn-edit,
.cg-next:hover .btn-edit {
  display: inline;
}
.cg-passed .btn-edit:hover {
  color: #999 !important;
}
.cg-active,
.cg-avtive.b-table-details:hover {
  background-color: #a2aeb2 !important;
  color: #445e65;
  font-weight: bold;
}
b-table-has-details,
b-table-details:hover {
  background-color: inherit !important;

  /*pointer-events: none;*/
}
.cg-warning {
  background: orange;
}
.cg-next {
  color: #999;
}

table.control-stats td {
  padding-bottom: 5px;
  padding-top: 5px;
}
table.control-stats td:first-child {
  padding-left: 0;
}
table.control-stats td:last-child {
  padding-right: 0;
}
</style>
