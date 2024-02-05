<template>
<div>
  <div class="container p-3">
    <h1>Oversigt over Nathejk</h1>

    <b-card v-for="year in years" class="mb-2" :title="year.theme" header-tag="header" footer-tag="footer">
        <template #header><div class="d-flex justify-content-between">
          <h6 class="mb-0"><span class="text-uppercase">{{ year.name }}</span><span class="ml-3 text-muted">{{ year.cityDeparture }} - {{ year.cityDestination }}</span></h6>
          <div><button type="button" @click="showModal(year.slug)" class="btn btn-sm btn-outline-secondary p-0 b-0"><i class="fas fa-fw fa-pencil-alt"></i></button>
</div>
            </div>
      </template>
      <b-card-text>{{ year.story }}</b-card-text>
    </b-card>
  </div>

  <b-modal ref="modal" size="lg" header-class="hazyblue bg-midnightblue">
    <div slot="modal-title">
      <i class="fas fa-fw fa-map-marker-alt"></i> Nathejk
    </div>
<form>
  <div class="form-row">
    <div class="form-group col-md-8">
      <label for="inputEmail4">Udgave</label>
      <input type="text" class="form-control" id="inputEmail4" placeholder="Udgave" v-model="year.name">
    </div>
    <div class="form-group col-md-4">
      <label for="inputPassword4">Slug</label>
      <input type="text" class="form-control" id="inputPassword4" placeholder="Slug" v-model="year.slug">
    </div>
  </div>
  <div class="form-row">
    <div class="form-group col-md-6">
      <label for="inputEmail4">Startby</label>
      <input type="text" class="form-control" id="inputEmail4" placeholder="Startby" v-model="year.cityDeparture">
    </div>
    <div class="form-group col-md-6">
      <label for="inputPassword4">Målby</label>
      <input type="text" class="form-control" id="inputPassword4" placeholder="Målby" v-model="year.cityDestination">
    </div>
  </div>
  <div class="form-group">
    <label for="inputAddress">Historieramme</label>
    <input type="text" class="form-control" id="inputAddress" placeholder="Overskrift">
  </div>
  <div class="form-group">
    <textarea class="form-control" id="exampleFormControlTextarea1" rows="3" placeholder="Beskrivelse"></textarea>
  </div>
</form>
    <div slot="modal-footer">
      <button type="button" class="btn btn-sm btn-outline-secondary" @click="closeModal">Luk</button>
    </div>
  </b-modal>
</div>
</template>

<style>
</style>

<script>
import axios from 'axios';

export default {
    data: () => ({
      date:{},
      title: 'Nathejk 2019',
      team: {},
      user: { name:'', controls:[] },
      years: [],
      year: {},
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
    },
    methods: {
      async load () {
        try {
            const rsp = await axios.get('/api/years', { withCredentials: true } )
            if (rsp.status == 200) {
                this.years = rsp.data.years
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
      },
      async save () {
        try {
            const rsp = await axios.get('/api/years', { withCredentials: true } )
            if (rsp.status == 200) {
                this.years = rsp.data.years
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
      },
      getYear(slug) {
        for (const year of this.years) {
          if (year.slug == slug) {
            return year
          }
        }
        return null
      }
      showModal(slug) {
        this.year = JSON.parse(JSON.stringify(this.getYear(slug)))
        this.$refs['modal'].show()
      },
      closeModal() {
        this.$refs['modal'].hide()
      },
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
    },
    async mounted() {
      this.load()
    },
}
</script>
