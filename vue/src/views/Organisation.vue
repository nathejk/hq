<template>
    <div class="container p-3">
        <vue-good-table ref="teamlist" styleClass="vgt-table condensed"
            :columns="columns"
            :rows="users"
            @on-row-click="showUser"
            :search-options="{enabled: true}"
            :group-options="{enabled: true}"
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
  <b-modal ref="userModal" size="lg" header-class="hazyblue bg-midnightblue">
    <div slot="modal-title">
        <i class="fas fa-fw fa-phone"></i> Telefonnummer
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
          <select class="form-control form-control-sm" v-model="user.group">
            <optgroup v-for="group in groupOptions" :label="group.label" :key="group.label">
              <option v-for="option in group.options" :value="option.slug" :key="option.slug">{{ option.text }}</option>
            </optgroup>
          </select>
        </div>
      </div>
      <div v-if="false" class="form-group row">
        <label class="col-sm-2 col-form-label" for="modalFunction">Scanner som</label>
        <div class="col-sm-10">

          <div v-for="(control, i) in user.controls" class="form-group d-flex mb-1" :key="i">
            <select class="form-control form-control-sm col mr-3" v-model="control.slug">
              <optgroup v-for="(group, name) in controlOptions" :label="name" :key="name">
                <option v-for="option in group" :value="option.value" :key="option.value">{{ option.text }}</option>
              </optgroup>
            </select>
            <select class="form-control form-control-sm col-1 mr-3" v-model="control.day">
              <option>fre</option>
              <option>lør</option>
              <option>søn</option>
            </select>
            <span>
              <!-- vue-timepicker placeholder="Starttid" v-model="control.start"></vue-timepicker> <i class="fa fa-arrow-right"></i> <vue-timepicker placeholder="Sluttid" v-model="control.end"></vue-timepicker -->
              <button type="button" @click="deleteControl(i)" class="btn btn-sm btn-outline-danger ml-3"><i class="far fa-trash-alt text-dange"></i></button>
            </span>
          </div>

          <div class="form-group d-flex justify-content-end">
              <button type="button" @click="addControl" class="btn btn-sm btn-outline-success ml-3"><i class="fa fa-plus"></i></button>
          </div>

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
</template>

<style>
.vgt-global-search {
    border:0;
    background:none;
}
</style>

<script>
import axios from 'axios';
//import VueTimepicker from 'vue2-timepicker/src/vue-timepicker.vue'
import { BModal } from 'bootstrap-vue'
//import { corps } from '@constants'
//Vue.use(ModalPlugin)


export default {
    components: {
        //Modal: () => import('@/components/Modal'),
        BModal,
    },
    data: () => ({
      date:{},
      title: 'Nathejk 2019',
      team: {},
      columns: [
        {label: 'Navn', field: 'name'},
        {label: 'Telefon', field: 'phone'},
        {label: 'E-mail', field: 'email'},
        {label: 'Korps', field: 'corps'},
        {label: 'medlemsnr.', field: 'medlemnr'},
        {label: 'Hold', field: 'group'},
        //{label: 'Antal scanninger', field: 'scanCount', type:'number'},
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
      controlOptions: {
        Start: [
          { text: 'Startpost',  value: 'start' },
        ],
        'Postlinie 1': [
          { text: 'Post 1A',  value: 'post1a' },
          { text: 'Post 1B',  value: 'post1b' },
          { text: 'Post 1C',  value: 'post1c' },
        ],
        'Postlinie 2': [
          { text: 'Post 2A',  value: 'post2a' },
          { text: 'Post 2B',  value: 'post2b' },
          { text: 'Post 2C',  value: 'post2c' },
        ],
        'Postlinie 3': [
          { text: 'Post 3A',  value: 'post3a' },
          { text: 'Post 3B',  value: 'post3b' },
          { text: 'Post 3C',  value: 'post3c' },
        ],
        'Postlinie 4': [
          { text: 'Post 4A',  value: 'post4a' },
          { text: 'Post 4B',  value: 'post4b' },
          { text: 'Post 4C',  value: 'post4c' },
        ],
        Oplevelse: [
          { text: 'Oplevelsespost',  value: 'oplev' },
        ],
        'Mål': [
          { text: 'Målpost',  value: 'slut' },
        ],
      },
      user: { name:'', controls:[] },
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
            //return this.$store.getters['dims/users']
            const users = {}
            for (const user of this.$store.getters['dims/users']) {
                const label = this.groupSlugs[user.group] || 'Andet'
                if (!users[label]) {
                    users[label] = []
                }
                users[label].push(user)

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
        addControl() {
            this.user.controls.push({})
        },
        deleteControl(index) {
            this.user.controls.splice(index, 1)
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
                const rsp = await axios.post(window.envConfig.API_BASEURL + '/api/user', this.user, { withCredentials: true })
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
                const rsp = await axios.delete(window.envConfig.API_BASEURL + '/api/user', { withCredentials: true, data: this.user })
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
