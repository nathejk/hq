<template>
  <div id="view-organisation">

    <div class="menu">
      <div class="container d-flex justify-content-between align-items-center">
        <div>
        </div>
        <div>
            <button v-if="false" type="button" class="btn btn-outline-secondary btn-circle btn-sm ml-2" @click="newUnit"><i class="fas fa-cog"></i></button>
            <button type="button" class="btn btn-outline-secondary btn-circle btn-sm ml-2" @click="newUser"><i class="fas fa-user-plus"></i></button>
        </div>
      </div>
    </div>

    <div class="container p-3">
        <vue-good-table ref="teamlist" styleClass="vgt-table condensed"
            :columns="columns"
            :rows="users"
            @on-row-click="showUser"
            :search-options="{enabled: true}"
            :group-options="{enabled: true}"
            :sort-options="{
                enabled: true,
                initialSortBy: {field: 'department', type: 'asc'}
            }"
            >
            <div slot="table-actions">
                <div class="btn-group" role="toolbar" >
                    <button class="btn btn-sm btn-outline-success float-right mr-2" @click="newUser"><i class="fas fa-plus"></i> ny</button>
                </div>
            </div>
            <template slot="table-row" slot-scope="props">
            <span v-if="props.column.field == 'name'">
                {{props.formattedRow[props.column.field]}} <i v-if="props.row.hqAccess" class="fas fa-xs fa-key text-warning"></i>
            </span>
            <span v-else>
                {{props.formattedRow[props.column.field]}}
            </span>
        </template>
        <div lot="emptystate"><em class="text-warning text-italic">- Telefonlisten er tom -</em></div>
    </vue-good-table>

<b-modal ref="unitModal" size="lg" header-class="hazyblue bg-midnightblue">
<div slot="modal-title">
    <i class="fas fa-fw fa-user"></i> Hold og enheder
</div>
<template class="small">
    <EditUnits />
<!--
  <draggable v-for="department in departments" v-model="department.units" draggable=".item" group="departments">
    <strong>{{ department.name }}</strong>
    <transition-group>
    <div v-for="unit in department.units" :key="unit.id" class="item"  style="padding:5px;border:1px solid #999">
        {{unit.name}}
    </div>
    </transition-group>

  </draggable>
  <button slot="footer" @click="addUnit">Add</button>
-->
</template>
<div slot="modal-footer">
  <button type="button" class="btn btn-sm btn-outline-secondary" @click="closeUnit">Luk</button>
  <button type="button" class="btn btn-sm btn-outline-danger ml-2" click="deleteUser">Slet</button>
  <button type="button" class="btn btn-sm btn-success ml-2" click="saveUser">Gem</button>
</div>
</b-modal>

<b-modal ref="userModal" size="lg" header-class="hazyblue bg-midnightblue">
<div slot="modal-title">
    <i class="fas fa-fw fa-user"></i> Personoplysninger
</div>
<form class="small">
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalUserName">Navn</label>
    <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalUserName" v-model="user.name"></div>
  </div>
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalPhone">Telefon</label>
    <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalPhone" v-model="user.phone"></div>
  </div>
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalEmail">E-mail</label>
    <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalEmail" v-model="user.email"></div>
  </div>
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalMedlem">Medlemsnummer</label>
    <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalMedlem" v-model="user.medlemnr"></div>
  </div>
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalKorps">Korps</label>
    <div class="col-sm-10"><input type="text" class="form-control form-control-sm" id="modalKorps" v-model="user.corps"></div>
  </div>
  <div class="form-group formcheck row">
    <label class="col-sm-2 col-form-label" for="modalHqAccess">HQ adgang</label>
    <div class="col-sm-10"><div class="form-check py-1">
        <input type="checkbox" class="form-check-input" id="modalHqAccess" v-model="user.hqAccess">
        <label class="form-check-label" for="modalHqAccess">Ja</label>
    </div></div>
  </div>
  <div class="form-group row">
    <label class="col-sm-2 col-form-label" for="modalFunction">Funktion</label>
    <div class="col-sm-10">
      <select class="form-control form-control-sm" v-model="user.department">
        <optgroup v-for="group in groupOptions" :label="group.label" :key="group.label">
          <option v-for="option in group.options" :value="option.slug" :key="option.slug">{{ option.text }}</option>
        </optgroup>
      </select>
    </div>
  </div>
</form>
<div slot="modal-footer">
  <button type="button" class="btn btn-sm btn-outline-secondary" @click="closeUser">Luk</button>
  <button type="button" class="btn btn-sm btn-outline-danger ml-2" @click="deleteUser">Slet</button>
  <button type="button" class="btn btn-sm btn-success ml-2" @click="saveUser">Gem</button>
</div>
</b-modal>
</div>
</div>
</template>

<style>
.menu .btn-outline-secondary:hover {
   /* color: black !important;*/
}
.btn-circle.btn-sm {
  width: 30px;
  height: 30px;
  border-radius: 15px;
  font-size: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
        .btn-circle.btn-md {
            width: 50px;
            height: 50px;
            padding: 7px 10px;
            border-radius: 25px;
            font-size: 10px;
            text-align: center;
        }
        .btn-circle.btn-xl {
            width: 70px;
            height: 70px;
            padding: 10px 16px;
            border-radius: 35px;
            font-size: 12px;
            text-align: center;
        }
.menu {
  background:#f5f5f5;
  border-bottom: 1px solid #ccc;
  padding: 0.3rem 0;
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

.vgt-global-search {
    border:0;
    background:none;
}
</style>

<script>
import axios from 'axios';
import { BModal } from 'bootstrap-vue'
import EditUnits from './EditUnits.vue'
//import { corps } from '@constants'
//Vue.use(ModalPlugin)


export default {
    components: {
        //Modal: () => import('@/components/Modal'),
        BModal,
        draggable: () => import('vuedraggable'),
        EditUnits,
    },
    data: () => ({
            myArray:[
                    {id:1, name:"One"},
                    {id:2, name:"Two"},
                    {id:3, name:"Three"},
                    {id:4, name:"Four"},
                ],
      date:{},
      title: 'Nathejk 2019',
      team: {},
      columns: [
        {label: 'Hold', field: 'department'},
        {label: 'Navn', field: 'name'},
        {label: 'Telefon', field: 'phone'},
        {label: 'E-mail', field: 'email'},
        {label: 'Korps', field: 'corps'},
        {label: 'medlemsnr.', field: 'medlemnr'},
        //{label: 'Antal scanninger', field: 'scanCount', type:'number'},
      ],
      departments: [
        {name:'Banditter', slug:'bandit', units:[{id:1, name:"One"}]},
        {name:'Postmandskab', slug:'post', units:[]},
        {name:'Logistik', slug:'logistik', units:[]},
        {name:'Guides', slug:'guide', units:[]},
        {name:'Andet', slug:'', units:[]},
      ],
      groupOptions: [
        {
            'label': 'Banditter',
            'options': [
              { text: 'BHQ', slug: 'bhq' },
              { text: 'LOK 1', slug: 'lok1' },
              { text: 'LOK 2', slug: 'lok2' },
              { text: 'LOK 3', slug: 'lok3' },
              { text: 'LOK 4', slug: 'lok4' },
              { text: 'LOK 5', slug: 'lok5' },
            ],
        },
        {
            'label': 'Guides',
            'options': [
              { text: 'Anders And', slug: 'andersand' },
              { text: 'Bedstemor And', slug: 'bedstemorand' },
              { text: 'Dumbo', slug: 'dumbo' },
              { text: 'Fedtmule', slug: 'fedtmule' },
              { text: 'Georg Gearløs', slug: 'gearløs' },
              { text: 'Ståland', slug: 'ståland' },
              { text: 'Andeby Nærradio', slug: 'andeby' },
            ],
        },
        {
            'label': 'Postmandskab',
            'options': [
              { text: 'Postmandskab', slug: 'post' },
            ],
        },
        {
            'label': 'Logistik',
            'options': [
              { text: 'Galaxy', slug: 'galaxy' },
              { text: 'Rover 1', slug: 'rov1' },
              { text: 'Rover 2', slug: 'rov2' },
              { text: 'Rover 3', slug: 'rov3' },
              { text: 'Rover 4', slug: 'rov4' },
              { text: 'Rover 5', slug: 'rov5' },
              { text: 'Madbil', slug: 'madbil' },
              { text: 'Special Posttjenesten', slug: 'specialpost' },
            ],
        },
        {
            'label': 'Teknisk Tjeneste',
            'options': [
              { text: 'Teknik 1', slug: 'tek1' },
              { text: 'Teknik 2', slug: 'tek2' },
              { text: 'Skadestuebil', slug: 'skade' },
              { text: 'Samarit 1', slug: 'sam1' },
              { text: 'Samarit 2', slug: 'sam2' },
              { text: 'Samarit 3', slug: 'sam3' },
              { text: 'Samarit 4', slug: 'sam4' },
              { text: 'Høne funktion', slug: 'høne' },
            ],
        },
        {
            'label': 'PR og Kommunikation',
            'options': [
              { text: 'Foto 1', slug: 'foto1' },
              { text: 'Foto 2', slug: 'foto2' },
              { text: 'Mediefraktionen', slug: 'mediefraktion' },
              { text: 'SoMe', slug: 'some' },
              { text: 'Andet', slug: 'pr' },
            ],
        },
        {
            'label': 'Andet',
            'options': [
              { text: 'Gøjl', slug: 'gøjl' },
              { text: 'Merchandise', slug: 'merchandise' },
              { text: 'Madhold', slug: 'mad' },
            ],
        },
      ],
      user: { name:'' },
    }),
    computed: {
        groupSlugs() {
            const slugs = {}
            for (const group of this.groupOptions) {
                for (const option of group.options) {
                    slugs[option.slug] = group.label
                }
            }
            return slugs
        },
        users() {
            const users = {}
            for (const person of this.$store.getters['dims/personnel']) {
                const label = this.groupSlugs[person.department] || 'Andet'
                if (!users[label]) {
                    users[label] = []
                }
                users[label].push(person)

            }
            const groups = []
            for (const group of this.groupOptions) {
                if (users[group.label]) {
                    groups.push({mode:'span', label:group.label, children: users[group.label]})
                }
            }
            return groups
        },
        paidUts() {
            let paid = 0
            return paid
        },
    },
    methods: {
        addUnit() {
            this.departments[this.departments.length-1].units.push({id:123, name:"Enhed"})
        //        this.myArray.push({id:this.myArray.length, name:this.myArray.length})
        },
        addControl() {
            this.user.controls.push({})
        },
        deleteControl(index) {
            this.user.controls.splice(index, 1)
        },
        newUnit() {
          this.$refs['unitModal'].show()
        },
        closeUnit() {
          this.$refs['unitModal'].hide()
        },
        newUser() {
            for (let key in this.user) {
                this.user[key] = ''
            }
            this.$refs['userModal'].show()

        },
        closeUser() {
          this.$refs['userModal'].hide()
        },
        showUser(params) {
            Object.assign(this.user, params.row)
                    this.$refs['userModal'].show()

            //this.$bvModal.show('userModal')	
            //$('#userModal').modal({})
        },
        async saveUser() {
            try {
                const rsp = await axios.post('/api/personnel', this.user, { withCredentials: true })
                if (rsp.status == 200) {
                    //$('#userModal').modal('hide')
                    this.$refs['userModal'].hide()
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
        async deleteUser() {
            if (!confirm('Er du sikker på at brugeren skal slettes?')) {
                return
            }
            try {
                const rsp = await axios.delete('/api/personnel', { withCredentials: true, data: this.user })
                if (rsp.status == 200) {
                    //$('#userModal').modal('hide')
                    this.$refs['userModal'].hide()
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
    },
    filters: {
        dateFull: function(value) {
            if (Number(value) == 0) return '-'
            //return moment(Number(value)*1000).format('D/M YYYY [kl.] H:mm:ss')
        },
        dateDM: function(value) {
            return value
            //return moment(Number(value)*1000).format('D/M')
        },
    },
    async mounted() {
        try {
                /*
            const rsp = await axios.get(window.envConfig.API_BASEURL + '/api/teams/' + this.$route.params.id,
            { withCredentials: true }
            )
            if (rsp.status == 200) {
                this.team = rsp.data.team
            }*/
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
    },
}
</script>
