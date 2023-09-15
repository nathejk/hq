<template>
  <div id="view-poster" class="h-100">

    <div class="menu" v-if="false">
      <div class="container d-flex justify-content-between">
        <div v-if="false">
            <a v-for="grp of controlgroups" role="button" :class="{'selected':grp.controlGroupId == viewControlGroupId}" :title="grp.name" @click="showControlGroup(grp.controlGroupId)" :key="grp.controlGroupId">{{ grp.name }}</a>
        </div>
        <div>
          <a v-if="viewControlGroupId" role="button" @click="editControlGroup()"><i class="fas fa-cog"></i></a>
          <a role="button" @click="newControlGroup()"><i class="fas fa-plus"></i></a>
        </div>
      </div>
    </div>

    <div class="bg-white h-100">
      <div class="container">
        <div class="row">
            <div class="col">
                <b-table small hover :items="rows" :fields="fields" :tbody-tr-class="rowClass" @row-clicked="rowClicked">
                  <template #cell(name)="data">
                    <EditInline v-model="data.item.name" @input="log" size="sm" />
                  </template>
                  <template #cell(action)="data">
                    <i v-if="data.item._showDetails" class="fas fa-times-circle fa-lg"></i>
                    <i v-else class="fas fa-chevron-circle-right fa-lg"></i>
                  </template>

                <template #row-details="row" class="bg-success">
                  <b-card>
                    <div v-for="cp in row.item.controlpoints" class="pb-3">
                        <div class="row controlpoint" @click="cp._showScanners=!cp._showScanners">
                            <div class="col">{{ cp.name }}</div>
                            <div class="col"><i v-if="false" class="fas fa-map-marker-alt"></i></div>
                            <div class="col text-center">{{ cp.openFrom | dateHHmm }} <i class="far fa-clock mx-1"></i> {{ cp.openUntil | dateHHmm }}</div>
                            <div class="col text-right"><small>(- %) -</small></div>
                        </div>
                        <div class="row" v-if="cp._showScanners" v-for="scanner in cp.scanners">
                            <div class="col"><small class="px-2">{{ person(scanner.userId).name }}</small></div>
                            <div class="col"></div>
                            <div class="col text-center"><small>{{ scanner.start | dateHHmm }} <i class="far fa-clock mx-1"></i> {{ scanner.end | dateHHmm }}</small></div>
                            <div class="col text-right"><small>-</small></div>
                        </div>
                    </div>
                    <a role="button" class="mr-3 btn btn-outline-secondary btn-sm" @click="editControlGroup(row.item.rowId)"><i class="fas fa-plus"></i>  ret postlinje</a>
                    <small><a @click="deleteControlGroup(row.item.rowId)" href="#">fjern postlinie</a></small>
                  </b-card>
                </template>
              </b-table>

          <a role="button" @click="newControlGroup()" class="btn btn-outline-secondary"><i class="fas fa-plus"> tilføj postlinje</i></a>
            </div>
            <!--
            <div class="col-8 pt-2">
                <div v-if="controlgroup.controlGroupId" class="card mb-4">
                    <div class="card-header d-flex justify-content-between">
                        <small class="text-uppercase grey">Status</small>
                        <small><span>Patruljer</span></small>
                    </div>
                    <div class="card-body pb-0">
                        <table class="table table-borderless control-stats">
                            <tbody>
                                <tr><td>Ikke ankommet</td><td class="text-right"><span @click="show(counts.NotArrived)" role="button" class="h6 font-weight-bold">{{ counts.NotArrived.length }}</span></td></tr>
                                <tr><td>Passeret rettidigt</td><td class="text-right"><span @click="show(counts.OnTime)" role="button" class="h6 font-weight-bold">{{ counts.OnTime.length }}</span></td></tr>
                                <tr><td>Passeret uden for åbningstid</td><td class="text-right"><span @click="show(counts.OverTime)" role="button" class="h6 font-weight-bold">{{ counts.OverTime.length }}</span></td></tr>
                                <tr><td>Udgået/sammenlagt</td><td class="text-right"><span @click="show(counts.Inactive)" role="button" class="h6 font-weight-bold">{{ counts.Inactive.length }}</span></td></tr>
                                <tr><td colspan="2"><hr class="my-1"></td></tr>
                                <tr><td colspan="2" class="text-right"><span class="h4 font-weight-bold">{{ totalCount }}</span></td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
                <table v-if="list.length > 0" class="table table-sm">
  <thead>
    <tr>
      <th scope="col">#</th>
      <th scope="col">Patrulje</th>
      <th scope="col"></th>
      <th scope="col"></th>
    </tr>
  </thead>
  <tbody>
    <tr v-for="teamId in list">
        <th scope="row">{{ patrulje(teamId).teamNumber }}</th>
        <td>{{ patrulje(teamId).name}}</td>
      <td></td>
      <td></td>
    </tr>
  </tbody>
</table>
            </div>
            <div class="col-4" v-if="controlgroup">

        <div v-for="control of controlgroup.controls" class="card mt-2" :key="control.controlId">
          <div class="card-header d-flex justify-content-between grey">
            <small class="text-uppercase grey">{{ control.name }}</small>
            <small>{{ control.dateRange.startDate | dateHHmm }} <i class="far fa-clock mx-1"></i> {{ control.dateRange.endDate | dateHHmm }}</small>
          </div>
          <div class="card-body pt-1 pb-0">
            <table class="table table-borderless table-sm">
              <tbody>
                <tr v-for="scanner of control.scanners" :key="scanner.scannerId"><td><small>{{ scanner | scannerName(controlgroup.users) }}</small></td><td class="text-right"><small>0</small></td></tr>
                <tr v-if="control.scanners.length > 0"><td colspan="2"><hr class="my-1"></td></tr>
                <tr><td><small>I alt</small></td><td class="text-right"><small><span class="hazyblue mr-3">(0 %)</span>0</small></td></tr>
              </tbody>
            </table>
          </div>
        </div>



    </div>
            -->
    </div>
    </div>
    </div>


  <b-modal ref="modal" size="lg" header-class="hazyblue bg-midnightblue">
    <div slot="modal-title">
      <i class="fas fa-fw fa-map-marker-alt"></i> Postlinie
    </div>
    <form class="small">
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="modalName">Navn</label>
        <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalName" v-model="edit.name"></div>
      </div>
      <div v-for="(control, i) of edit.controls" :key="i" class="border col-sm-10 offset-sm-2 pt-2 mb-3 bg-light">
        <div class="form-group row">
          <label class="col-sm-2 col-form-label" for="modalName">Postnavn</label>
          <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalName" v-model="control.name"></div>
        </div>
        <div class="form-group row">
          <label class="col-sm-2 col-form-label" for="modalName">Åbningstider</label>
          <div class="col-sm-10">
            <div v-for="scheme of checkpointSchemes" class="form-check form-check-inline">
              <input class="form-check-input" type="radio" :id="scheme.key" v-model="control.scheme" :value="scheme.key">
              <label class="form-check-label" :for="scheme.key">{{ scheme.label }}</label>
            </div>
          </div>
        </div>
        <div v-if="control.scheme=='fixed'" class="form-group row mb-0">
          <div class="offset-sm-2 col-sm-5">
            <div class="form-group">
              <!--input type="text" class="form-control form-control-sm" id="modalName" vmodel="edit.name">
              <input type="text" class="form-control form-control-sm" id="modalName" vmodel="edit.name"-->
              <date-range-picker :time-picker="true" :locale-data="locale" :ranges="false" v-model="control.dateRange">
              <!--
                <template v-slot:input="picker" style="min-width: 350px;">{{ picker.startDate }} - {{ picker.endDate | date }}</template>
              -->
              </date-range-picker>
            </div>
          </div>
          <label class="col-sm-1 col-form-label" for="modalName">-/+ (minut)</label>
          <div class="col-sm-2 pl-0"><input type="number" min="0" class="form-control form-control-sm" id="modalName" v-model="control.minus"></div>
          <div class="col-sm-2 pl-0"><input type="number" min="0" class="form-control form-control-sm" id="modalName" v-model="control.plus"></div>
        </div>
        <div v-if="control.scheme=='relative'" class="form-group row mb-0">
          <div class="offset-sm-2 col-sm-5">
            <div class="form-group">
            <select class="form-control form-control-sm col mr-3" v-model="control.relativeControlGroupId">
              <option v-for="cg in controlgroups" :key="cg.controlGroupId" :value="cg.controlGroupId">{{ cg.name }}</option>
            </select>
            </div>
          </div>
          <label class="col-sm-3 col-form-label" for="modalName">afvigelse + (minut)</label>
          <div class="col-sm-2 pl-0"><input type="number" min="0" class="form-control form-control-sm" id="modalName" v-model="control.plus"></div>
        </div>
        <div class="form-group row border-top border-bottom pt-3">
          <label class="col-sm-2 col-form-label" for="modalName">Scannere</label>
          <div class="col-sm-10">
            <div class="row" v-for="(scanner, i) in control.scanners" :key="scanner.scannerId">

          <div class="col-sm-6">
            <div class="form-group mb-1">
              <date-range-picker :time-picker="true" :locale-data="locale" :ranges="false" v-model="scanner.dateRange">
              </date-range-picker>
            </div>
          </div>
          <div class="col-sm-5">
            <div class="form-group mb-1">
            <select class="form-control form-control-sm col mr-3" v-model="scanner.userId">
              <optgroup v-for="(group, name) in users" :label="group.label" :key="name">
                <option v-for="option in group.options" :key="option.slug" :value="option.slug">{{ option.label }}</option>
              </optgroup>
            </select>
            </div>
          </div>
          <div class="col-sm-1 p-0">
            <div class="form-group mb-1">
              <button type="button" @click="deleteScanner(control, i)" class="btn btn-sm btn-outline-danger"><i class="far fa-trash-alt text-dange"></i></button>
            </div>
          </div>

            </div>
            <div class="row">

              <div class="col-sm-12">
                <div class="form-group d-flex justify-content-end">
                  <button type="button" @click="addScanner(control)" class="btn btn-sm btn-outline-success ml-3"><i class="fa fa-plus"></i></button>
                </div>
              </div>

            </div>

          </div>
        </div>
        <div class="col-sm-12 p-0">
          <div class="form-group">
              <button type="button" @click="deleteControl(i)" class="btn btn-sm btn-outline-danger"><i class="fa fa-minus"></i> Fjern post</button>
          </div>
        </div>
      </div>
      <div class="form-group row">
        <div class="col-sm-10 offset-sm-2">
          <div class="form-group">
              <button type="button" @click="addControl" class="btn btn-sm btn-outline-success"><i class="fa fa-plus"></i> Tilføj post</button>
          </div>
        </div>
      </div>
    </form>
    <div slot="modal-footer">
      <button type="button" class="btn btn-sm btn-outline-secondary" @click="closeEdit">Luk</button>
      <button type="button" class="btn btn-sm btn-outline-danger ml-2" @click="deleteControlGroup">Slet</button>
      <button type="button" class="btn btn-sm btn-success ml-2" @click="saveControlGroup">Gem</button>
    </div>
  </b-modal>

    </div>
</template>

<style>
.controlpoint {
    background:#eee;
}
.controlpoint:hover {
    background:#ddd;
}
.controlpoint ~ .row:hover {
    background:#ddd;
}

.menu {
  background:#f5f5f5;
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

.menu a:hover, .menu a.selected {
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
  background:#f5f5f5;
}
.cg-passed, .cg-passed .btn-edit {
    color:#ccc!important;
}
.cg-passed .btn-edit, .cg-active .btn-edit, .cg-warning .btn-edit, .cg-next .btn-edit {
    display:none;
}
.cg-passed:hover .btn-edit, .cg-active:hover .btn-edit, .cg-warning:hover .btn-edit, .cg-next:hover .btn-edit {
    display:inline;
}
.cg-passed .btn-edit:hover {
    color:#999!important;
}
.cg-active, .cg-avtive.b-table-details:hover {
    background-color: #A2AEB2 !important;
        color:#445e65;
        font-weight:bold;
}
b-table-has-details, b-table-details:hover {
        background-color: inherit !important;

    /*pointer-events: none;*/
}
.cg-warning {
    background:orange;
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

<script>
import axios from 'axios';
import moment from 'moment'
import { BModal, BTable } from 'bootstrap-vue'
import DateRangePicker from 'vue2-daterange-picker'
import EditInline from '@/components/EditInline.vue'
//you need to import the CSS manually
import 'vue2-daterange-picker/dist/vue2-daterange-picker.css'


export default {
    data: () => ({
      editing: {},
      cgs: {},
      checkgroups: [],
      viewControlGroupId: String,
      edit:{ controls:Array },
      stats: {},
      list:[],
        test: 'HEAST',
      startedCount: 0,
      locale:{
        format:'ddd. HH:MM',
        firstDay: 1,
        daysOfWeek: ['Søn', 'Man', 'Tirs', 'Ons', 'Tors', 'Fre', 'Lør'],
        monthNames: ['Jan', 'Feb', 'Mar', 'Apr', 'Maj', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dec'],

        applyLabel: 'Ok',
        cancelLabel: 'Luk',
        weekLabel: 'W',
        customRangeLabel: 'Custom Range',
      },
      checkpointSchemes: [
          { key: 'fixed', label: 'Faste' },
          { key: 'relative', label: 'relative' },
          { key: 'none', label: 'ingen' },
      ],
      //rows: [],
      fields: [
              { key: 'name' },
              { key: 'notArrivedCount', label: 'Mangler', class:'text-right col-1'},
              { key: 'onTimeCount', label: 'Rettidig', class:'text-right col-1' },
              { key: 'overTimeCount', label: 'For sent', class:'text-right col-1' },
              { key: 'inactiveCount', label: 'Udgåede', class:'text-right col-1' },
              { key: 'totalCount', label: 'I alt', class:'text-right col-1' },
              { key: 'action', label: '', class:'text-right col-1 pr-3' },
      ],

    }),
    components: { BModal, BTable, DateRangePicker, EditInline },
    filters: {
      dateHHmm: function(value) {
        return moment(value).format('HH:mm')
      },
      scannerName: function(value, users) {
        if (users[value.userId]) {
          return users[value.userId].name
        }
        return '-'
      },
    },
    computed: {

      counts() {
        return this.stats[this.viewControlGroupId] || { NotArrived:[], OnTime:[], OverTime:[], Inactive:[] }
      },
      totalCount() {
        return this.counts.NotArrived.length + this.counts.OnTime.length + this.counts.OverTime.length + this.counts.Inactive.length
      },
      controlgroups() {
          return this.checkgroups;
        const cgs = this.$store.getters['dims/controlGroups']
          console.log('cgs', cgs)
        return cgs.sort((a, b) => (this.controlGroupStartDate(a) > this.controlGroupStartDate(b) ? 1 : -1))
      },
      controlgroup() {
        for (const grp of this.controlgroups) {
          if (grp.controlGroupId == this.viewControlGroupId) {
            return grp
          }
        }
        return {}
      },
      users() {
            const users = {}
            for (const user of this.$store.getters['dims/personnel']) {

                //const label = this.groupSlugs[user.group] || 'Andet'
                const label = user.group || 'Andet'
                if (!users[label]) {
                    users[label] = []
                }
                users[label].push(user)

            }
                console.log('users', users)
            const scanners = []
                for(    let [i, group] of Object.entries(users)) {
            //for (const group, i of users) {
                const scanner = {label:i, options:[]}
            for (const user of group) {
                    scanner.options.push({label:user.name, slug:user.userId})
            }
                scanners.push(scanner)
            }
                console.log('scanners', scanners)
            return scanners
      },
      rows() {
        const rows = []
        //const cgs = this.$store.getters['dims/controlGroups']
        //cgs.sort((a, b) => (this.controlGroupStartDate(a) > this.controlGroupStartDate(b) ? 1 : -1))
          console.log('stats', this.stats, this.checkgroups)
        for (const cg of this.checkgroups) {
            //const counts = this.stats[cg.controlGroupId] || { NotArrived:[], OnTime:[], OverTime:[], Inactive:[] }
            const cps = []
            for (const cp of cg.checkpoints) {
                cps.push({name:cp.name, scanners:cp.scanners, openFrom:cp.openFrom, openUntil:cp.openUntil, _showScanners:false})
            }
            const row = {
                rowId: cg.id,
                cgId: cg.id,
                name:cg.name,
                _edit: false,
                _showDetails: false,
                notArrivedCount: cg.notArrivedTeamIds.length,
                onTimeCount: (cg.onTimeTeamIds || []).length,
                overTimeCount: (cg.overTimeTeamIds || []).length,
                inactiveCount: cg.discontinuedTeamIds.length,
                controlpoints: cps,
            }
            
            row.totalCount = row.notArrivedCount + row.onTimeCount + row.overTimeCount + row.inactiveCount
            rows.push(row)
        }
        return rows
      },
    },
    methods: {
      async load () {
        try {
            const rsp = await axios.get('/api/checkgroups',
            { withCredentials: true }
            )
            if (rsp.status == 200) {
                const cgs = rsp.data.status.checkgroups
                this.checkgroups = cgs.sort((a, b) => (this.controlGroupStartDate(a) > this.controlGroupStartDate(b) ? 1 : -1))
                //this.startedCount = resp.data.startedCount
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
      },
      person (id) {
          return this.$store.getters['dims/person'](id)
      },
      patrulje(teamId) {
        return this.$store.getters['dims/patrulje'](teamId)
      },
      log (e) {
          console.log('input', e)
      },
      editControlGroupName(cg) {
          cg._edit = true
          return true
      },
      isEditControlGroupName(cgId) {
          return this.editing[cgId]
      },
        /*
      cgRows(cgId) {
          console.log('cgID', cgId)
          return this._controlgroup(cgId).controls

      },
        */
      rowClicked(item, index, event) {
              item._showDetails = !item._showDetails
          },
      rowClass(item, type) {
          if (!item || (type !== 'row' && type !== 'row-details')) {
              console.log('rowClass', item, type)
              return
          }
        if (item.notArrivedCount === 0) return 'cg-passed'
        if (item.onTimeCount == 0) return 'cg-next'
          //item._showDetails = true
              return 'cg-active'
      },
      _controlgroup(cgId) {
        const cgs = this.$store.getters['dims/controlGroups']
        for (const grp of cgs) {
            console.log(grp.controlGroupId, cgId)
          if (grp.controlGroupId == cgId) {
            return grp
          }
        }
        return {}
      },
      patrulje(teamId) {
        return this.$store.getters['dims/patrulje'](teamId)
      },
      show (list) {
              this.list = list
          },
      controlGroupStartDate(grp) {
        if (grp.checkpoints.length == 0) {
          return new Date().toISOString()
        }
        var startDate = grp.checkpoints[0].openFrom
        for (const cp of grp.checkpoints) {
          if (cp.openFrom < startDate) {
            startDate = cp.openFrom
          }
        }
        return startDate
      },
      activeScannerCount(scanners) {
        return 0
      },

      showControlGroup(controlGroupId) {
        this.list = []
        //this.viewControlGroupId = controlGroupId
      },
      editControlGroup(id) {
        const cg = this.$store.getters['dims/controlGroup'](id)
        this.edit = JSON.parse(JSON.stringify(cg))
        this.$refs['modal'].show()
      },
      closeEdit() {
        this.$refs['modal'].hide()
      },
      newControlGroup() {
          //this.rows.push({rowId:this.uuid(), _edit:true, _showDetails:true})
        //this.edit = { controls:[] }
        //this.$refs['modal'].show()
      },
      saveControlGroup() {
        const cps = []
        let cp = null
        while (cp = this.edit.controls.shift()) {
          cp.plus = parseInt(cp.plus)
          cp.minus = parseInt(cp.minus)
          cps.push(cp)
        }
        this.edit.controls = cps
        this.$store.dispatch("dims/updateControlGroup", this.edit);
        this.$refs['modal'].hide()
      },
      deleteControlGroup() {
        this.$store.dispatch("dims/deleteControlGroup", this.edit.controlGroupId);
        this.$refs['modal'].hide()
      },
      addControl() {
        this.edit.controls.push({ dateRange:{startDate: new Date(), endDate: new Date()}, scanners:[]})
      },
      deleteControl(i) {
        this.edit.controls.splice(i, 1)
      },
      addScanner(ctrl) {
        ctrl.scanners.push({
          dateRange: ctrl.dateRange,
          userId: '',
        })
      },
      deleteScanner(ctrl, i) {
        ctrl.scanners.splice(i, 1)
      },
      compilerows() {
        const rows = []
        //const cgs = this.$store.getters['dims/controlGroups']
        //cgs.sort((a, b) => (this.controlGroupStartDate(a) > this.controlGroupStartDate(b) ? 1 : -1))
          console.log('stats', this.stats, this.checkgroups)
        for (const cg of this.checkgroups) {
            //const counts = this.stats[cg.controlGroupId] || { NotArrived:[], OnTime:[], OverTime:[], Inactive:[] }
            const cps = []
            for (const cp of cg.checkpoints) {
                cps.push({name:cp.name, scanners:cp.scanners, dateRange:cp.dateRange, _showScanners:false})
            }
            const row = {
                rowId: cg.id,
                cgId: cg.id,
                name:cg.name,
                _edit: false,
                _showDetails: false,
                notArrivedCount: cg.notArrivedTeamIds.length,
                onTimeCount: cg.onTimeTeamIds.length,
                overTimeCount: cg.overTimeTeamIds.length,
                inactiveCount: cg.discontinuedTeamIds.length,
                controlpoints: cps,
            }
            
            row.totalCount = row.notArrivedCount + row.onTimeCount + row.overTimeCount + row.inactiveCount
            rows.push(row)
        }
        return rows
      },
      uuid(){
        var dt = new Date().getTime();
        var uuid = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
          var r = (dt + Math.random()*16)%16 | 0;
          dt = Math.floor(dt/16);
          return (c=='x' ? r :(r&0x3|0x8)).toString(16);
        });
        return uuid;
      },
    },
    async mounted() {
        this.load()
        //this.rows = this.compilerows()
        try {
            const rsp = await axios.get('/api/cgstatus',
            { withCredentials: true }
            )
            if (rsp.status == 200) {
                this.stats = rsp.data.controlGroups
                //this.startedCount = resp.data.startedCount
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
    },
    beforeDestroy() {
    }
}
</script>
