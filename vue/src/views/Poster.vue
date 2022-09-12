<template>
  <div id="view-poster" class="h-100">

    <div class="menu">
      <div class="container d-flex justify-content-between">
        <div>
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
            <div class="col-8 pt-2">
                <div v-if="controlgroup.controlGroupId" class="card mb-4">
                    <div class="card-header d-flex justify-content-between">
                        <small class="text-uppercase grey">Status</small>
                        <small><span>Patruljer</span></small>
                    </div>
                    <div class="card-body pb-0">
                        <table class="table table-borderless control-stats">
                            <tbody>
                                <tr><td>Ikke ankommet</td><td class="text-right"><span class="h6 font-weight-bold">88</span></td></tr>
                                <tr><td>Passeret rettidigt</td><td class="text-right"><span class="h6 font-weight-bold">20</span></td></tr>
                                <tr><td>Passeret uden for åbningstid</td><td class="text-right"><span class="h6 font-weight-bold">0</span></td></tr>
                                <tr><td>Udgået/sammenlagt</td><td class="text-right"><span class="h6 font-weight-bold">17</span></td></tr>
                                <tr><td colspan="2"><hr class="my-1"></td></tr>
                                <tr><td colspan="2" class="text-right"><span class="h4 font-weight-bold">125</span></td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>
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
        <div class="form-group row mb-0">
          <label class="col-sm-2 col-form-label" for="modalName">Åbningstider</label>
          <div class="col-sm-5">
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
          <label class="col-sm-3 col-form-label" for="modalName">Afvigelse (minut)</label>
          <div class="col-sm-1 pl-0"><input type="number" min="0" class="form-control form-control-sm" id="modalName" v-model="control.minus"></div>
          <div class="col-sm-1 pl-0"><input type="number" min="0" class="form-control form-control-sm" id="modalName" v-model="control.plus"></div>
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

<style scoped>
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
import moment from 'moment'
import { BModal } from 'bootstrap-vue'
import DateRangePicker from 'vue2-daterange-picker'
//you need to import the CSS manually
import 'vue2-daterange-picker/dist/vue2-daterange-picker.css'


export default {
    data: () => ({
      viewControlGroupId: String,
      edit:{ controls:Array },

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

    }),
    components: { BModal, DateRangePicker },
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
      controlgroups() {
        const cgs = this.$store.getters['dims/controlGroups']
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
            for (const user of this.$store.getters['dims/users']) {

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
    },
    methods: {
      controlGroupStartDate(grp) {
        if (grp.controls.length == 0) {
          return new Date().toISOString()
        }
        var startDate = grp.controls[0].dateRange.startDate
        for (const control of grp.controls) {
          if (control.dateRange.startDate < startDate) {
            startDate = control.dateRange.startDate
          }
        }
        return startDate
      },

      showControlGroup(controlGroupId) {
        this.viewControlGroupId = controlGroupId
      },
      editControlGroup() {
        this.edit = JSON.parse(JSON.stringify(this.controlgroup))
        this.$refs['modal'].show()
      },
      closeEdit() {
        this.$refs['modal'].hide()
      },
      newControlGroup() {
        this.edit = { controls:[] }
        this.$refs['modal'].show()
      },
      saveControlGroup() {
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
    },
    mounted: function () {
    },
    beforeDestroy() {
    }
}
</script>
