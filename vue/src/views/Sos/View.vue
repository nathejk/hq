<template>

    <div class="container">
        <h2 v-if="sos.sosId" class="py-3 border-bottom">
            <div v-if="headline">
                <form>
                  <div class="form-row align-items-center">
                    <div class="col">
                      <input type="text" class="form-control" placeholder="Overskrift" v-model="headline">
                    </div>
                    <div class="col-auto">
                      <button @click="saveHeadline" type="button" class="btn btn-outline-success">Opdater</button>
                      <button @click="cancelHeadline" type="button" class="btn btn-outline-secondary ml-2">Anuller</button>
                    </div>
                  </div>
                </form>
            </div>
            <div v-else>
                {{sos.headline }}
                <button type="button" class="btn btn-sm btn-hint" @click="editHeadline"><i class="fas fa-pencil-alt"></i></button>
                <span v-if="false" class="text-muted">#12</span>
            </div>
        </h2>

        <div class="row py-4">
            <div class="col-8">
                <div v-if="sos.sosId" class="media summary">
                    <img src="https://st2.depositphotos.com/1001599/7396/v/450/depositphotos_73968141-stock-illustration-hiking-exercise-thin-line-icon.jpg" style="width:50px;height:50px;" class="rounded avatar mr-3">
                    <div class="media-body">
                        <div class="card">

                            <div class="card-header d-flex">
                                <small>
                                    <span class="text-uppercase pr-3">Sagen kort</span>
                                    <span v-if="sos.closed" class="badge bg-danger p-1"><i class="fas fa-check text-light"></i> Afsluttet</span>
                                    <span v-else class="badge bg-success p-1"><i class="fas fa-asterisk text-light"></i> Åben</span>
                                </small>
                                <ul class="navbar-nav flex-row ml-auto d-none d-flex">
                                    <li class="nav-item pl-3">
                                        <a class="nav-link text-dark py-0" href="#"><i class="far fa-smile"></i></a>
                                    </li>
                                    <li v-if="false" class="nav-item dropdown pl-3">
                                        <a class="nav-item nav-link text-dark py-0" data-toggle="dropdown" href="#" aria-haspopup="true" aria-expanded="true"><i class="fas fa-ellipsis-h"></i></a>
                                        <div class="dropdown-menu dropdown-menu-right">
                                            <a class="dropdown-item" href="/docs/4.1/">Latest (v4.1.x)</a>
                                            <div class="dropdown-divider"></div>
                                        </div>
                                    </li>
                                </ul>
                            </div>

                            <div class="card-body bg-light py-2">
                                    <dl class="row">
                                      <dt class="col-sm-3 small grey text-uppercase">Oprettet:</dt>
                                      <dd class="col-sm-9">{{ sos.createdAt }}</dd>
                                      <template v-if="sos.severity">
                                        <dt class="col-sm-3 small grey text-uppercase">Alvor:</dt>
                                        <dd class="col-sm-9">{{ severities[severity] | upper }}</dd>
                                      </template>
                                      <template v-if="sos.assignee">
                                        <dt class="col-sm-3 small grey text-uppercase">Tildelt:</dt>
                                        <dd class="col-sm-9">{{ assignees[assignee] }}</dd>
                                      </template>
                                      <template v-if="position">
                                        <dt class="col-sm-3 small grey text-uppercase">Position:</dt>
                                        <dd class="col-sm-9">{{ position.lat }}, {{ position.lng }}</dd>
                                      </template>
                                    </dl>
                                    <hr>
                                    <p>{{ sos.description }}</p>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-for="(activity, i) of sos.activities" :key="i">
                    <div v-if="activity.type == 'comment'" class="media mt-3">
                        <img src="https://st2.depositphotos.com/1001599/7396/v/450/depositphotos_73968141-stock-illustration-hiking-exercise-thin-line-icon.jpg" style="width:50px;height:50px;" class="rounded avatar mr-3">
                        <div class="media-body">
                            <div class="card">
                                <div class="card-header d-flex">
                                    <small>
                                        <span class="text-uppercase">{{ activity.createdAt }}</span>
                                    </small>
                                </div>
                                <div class="card-body py-2" style="white-space: pre-line;">
                                    {{ activity.comment.plainText }}
                                </div>
                            </div>
                        </div>
                    </div>
                    <div v-else-if="activity.type == 'close'">
                        <ActivityLine text="lukkede sagen" iconClass="fas fa-check" iconColorClass="text-light" iconBgColorClass="text-danger" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'reopen'">
                        <ActivityLine text="genåbnede sagen" iconClass="fas fa-asterisk" iconColorClass="text-light" iconBgColorClass="text-success" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'severity'">
                        <ActivityLine :text="'prioriterede sagen som ' + (severities[activity.value] || '').toUpperCase()" iconClass="far fa-bell" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'assign'">
                        <ActivityLine :text="'tildelte sagen til ' + (assignees[activity.value] || '').toUpperCase()" iconClass="fas fa-thumbtack" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'associate'">
                        <ActivityLine :text="'Tilknyttede patruljen ' + patrulje(activity.value).name" iconClass="fas fa-users" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'disassociate'">
                        <ActivityLine :text="'Fjernede tilknytning til patruljen ' + activity.value" iconClass="fas fa-users-slash" :timestamp="activity.createdAt" />
                    </div>
                    <div v-else-if="activity.type == 'memberstatus'">
                        <ActivityLine :text="'Person udgået ' + activity.status" iconClass="fas fa-user-slash" :timestamp="activity.createdAt">
                        <span><a href="">{{ spejder(activity.value).name }}</a> er udgået</span>
                        </ActivityLine>
                    </div>
                    <div v-else class="media">
                        <ActivityLine :text="activity.type" :timestamp="activity.createdAt" />
                    </div>
                </div>

                <div class="media">
                    <img src="https://st2.depositphotos.com/1001599/7396/v/450/depositphotos_73968141-stock-illustration-hiking-exercise-thin-line-icon.jpg" style="width:50px;height:50px;" class="rounded avatar mr-3">
                    <div class="media-body">
                        <div class="card">
                            <div class="card-header d-flex">
                                <small>
                                    <span class="text-uppercase">Ny kommentar</span>
                                </small>
                            </div>
                            <div class="card-body py-2">
                                <form novalidate="novalidate">
                                    <div v-if="!sos.sosId" class="form-group">
                                        <input type="text" placeholder="Overskrift" class="form-control" v-model="headline" />
                                        <div class="invalid-feedback"></div>
                                    </div>
                                    <div class="form-group">
                                        <textarea rows="3" placeholder="Skriv ..." class="form-control" v-model="description"></textarea>
                                        <div class="invalid-feedback"></div>
                                    </div>
                                    <div class="form-group mb-0">
                                        <div class="row">
                                            <div class="col text-right">
                                                <button @click="closeSos" v-if="sos.sosId && !sos.closed" type="button" class="btn btn-outline-info">Luk sag</button>
                                                <button @click="reopenSos" v-if="sos.sosId && sos.closed" type="button" class="btn btn-outline-info">Genåbn sag</button>
                                                <button @click="addSosComment" type="button" class="btn btn-info ml-2">Tilføj kommentar</button>
                                            </div>
                                        </div>
                                    </div>
                                </form>
                            </div>
                        </div>
                    </div>
                </div>


            </div>
            <div class="col-4">
                <div class="card mb-3">
                    <div class="card-header d-flex">
                        <small>
                            <span class="text-uppercase">Tilknyt patrulje</span>
                        </small>
                    </div>
                    <div class="card-body">
                        <select class="form-control" v-model="teamId" @change="associateTeam">
                            <option value="">Patrulje...</option>
                            <option v-for="patrulje of patruljer" :key="patrulje.id" :value="patrulje.id">{{ patrulje.name }}</option>
                        </select>

                        <div v-for="(foumd, teamId)  of sos.teamIds" :key="teamId" class="associated  list-group-flush mt-3">
                          <div class="d-flex justify-content-between align-items-center list-group-item">
                            <router-link :to="{name:'patrulje', params: { id: teamId }}" class="text-dark">
                              <strong>{{ patrulje(teamId).teamNumber }} - {{ patrulje(teamId).name }}</strong>
                            </router-link>
                            <button :id="'team-' + teamId" class="btn btn-hint hint py-0 px-1"><i class="far fa-dot-circle"></i></button>

                            <b-popover :target="'team-' + teamId" triggers="hover" placement="left">
                              <template #title><i class="fas fa-user mr-1"></i> {{ patrulje(teamId).teamNumber }} - {{ patrulje(teamId).name }}</template>
                              <div class="list-group list-group-flush">
                                <a v-for="team of mergeableTeams(teamId)" :key="team.teamId" class="list-group-item list-group-item-action" role="button" @click="mergeTeams(team.teamId, teamId)"><i class="fas fa-fw fa-link"></i> Gør til en del af {{ team.teamNumber }}</a>
                                <a class="list-group-item list-group-item-action" v-if="patrulje(teamId).parentTeamId" role="button" @click="splitTeam(teamId)"><i class="fas fa-fw fa-unlink"></i> Ophæv sammenlægning med {{ patrulje(patrulje(teamId).parentTeamId).teamNumber}}</a>
                                <a class="list-group-item list-group-item-action" role="button" @click="disassociateTeam(teamId)"><i class="fas fa-fw fa-times"></i> Fjern fra sag</a>
                              </div>
                            </b-popover>
                          </div>

                          <ul class="list-group list-group-flush text-small">
                            <li v-for="member of patrulje(teamId).members" :key="member.memberId" class="d-flex justify-content-between align-items-center list-group-item">
                              <span :class="{strike:member.status != 'active'}"><i class="far fa-user mr-1"></i> <span>{{ member.name }}</span></span>
                              <button :id="'user-' + member.memberId" class="btn btn-hint hint py-0 px-1"><i class="far fa-circle"></i></button>

                              <b-popover :target="'user-' + member.memberId" triggers="hover" placement="left">
                                <template #title><i class="fas fa-user mr-1"></i> {{ member.name }}</template>
                                <div v-for="(label, slug) of memberStatuses" :key="slug" class="form-check">
                                  <label class="form-check-label">
                                    <input class="form-check-input" :name="member.memberId" type="radio" :value="slug" v-model="member.status" @change="memberStatus(teamId, member.memberId, slug)">
                                    {{ label }}
                                  </label>
                                </div>
                              </b-popover>

                            </li>
                          </ul>
                        </div>
                    </div>
                </div>
                <div class="card mb-3">
                    <div class="card-header d-flex">
                        <small>
                            <span class="text-uppercase">Placering</span>
                        </small>
                    </div>
                    <div class="card-body">
                        <div v-if="false">
                        <label class="small mb-1">Find på kort</label>
                        <a href="#select-workshop" data-toggle="modal" data-target="#mapModal" class="btn btn-outline-form d-block text-left mb-2">
                            <span v-if="position">
                                {{ position.lat.toFixed(5) }}, {{ position.lng.toFixed(5) }}
                                <i class="fas fa-map-marked-alt float-right mt-1"></i>
                            </span>
                            <span v-else>
                                Vælg...
                                <i class="fas fa-map-marked-alt float-right mt-1"></i>
                            </span>
                        </a>
                        </div>
                        <label class="small mb-1">Find med positions-SMS</label>
                        <div class="form-row mb-3">
                            <div class="col">
                                <select class="form-control" v-model="smsSpejder">
                                    <option value="">Vælg...</option>
                                    <optgroup v-for="(found, teamId) of sos.teamIds" :key="teamId" :label="patrulje(teamId).name">
                                        <option v-for="member of patrulje(teamId).members" :key="member.memberId" :value="member.memberId">{{ member.name }}</option>
                                    </optgroup>
                                </select>
                            </div>
                            <div class="col-auto">
                                <button type="button" :disabled="smsSpejder == ''" @click="sendPositionSms" class="btn btn-outline-info">Send SMS</button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="card mb-3">
                    <div class="card-header d-flex">
                        <small>
                            <span class="text-uppercase">Handling</span>
                        </small>
                    </div>
                    <div class="card-body">
                        <label class="small mb-1">Sagens alvor</label>
                        <div class="form-row mb-3">
                            <div class="col">
                                <select class="form-control" v-model="severity" @change="setSeverity">
                                    <option disabled value="">Vælg...</option>
                                    <option v-for="(severity, slug) in severities" :key="slug" :value="slug">{{ severity }}</option>
                                </select>
                            </div>
                        </div>
                        <label class="small mb-1">Send videre til</label>
                        <div class="form-row mb-3">
                            <div class="col">
                                <select class="form-control" v-model="assignee" @change="assign">
                                    <option disabled value="">Vælg...</option>
                                    <option v-for="(assignee, slug) in assignees" :key="slug" :value="slug">{{ assignee }}</option>
                                </select>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

</template>

<style>
.strike span {
    text-decoration: line-through;
    color:#999;
}
.associated {
    font-size:0.7rem;
}

.associated    .hint {
        color: #d5d5d5;
    }

    .associated:hover .hint {
        color:#999;
    }
.associated    .hint:hover {
        color:red;
    }

.btn-hint {
    color: #d5d5d5;
    background-color: transparent;
    border-color: transparent;

    &:hover {
        color:#a2aeb2;
        background-color:#d5d5d5;
        border-color:#d5d5d5;
    }
}

.media .avatar {
    width:50px;
}
.list-group-flush {
/*    font-size:0.7rem*/
}
.list-group-item {
    padding:0.2rem
}

.btn-outline-form {
    color: #555!important;
    border-color: #ced4da!important;
}
table tr.done {
    color:#ccc;
}
.text-lightgrey {
    color: #eee;
}
.summary .card-header {
    background-color: #A1AFB2;
    border-bottom: 1px solid #445E64!important;
}
.summary .card {
    border: 1px solid #445E64!important;
}
</style>

<script>
//import MapModal from '../MapModal.vue'
import axios from 'axios';
import ActivityLine from './ActivityLine.vue'
import { BPopover } from 'bootstrap-vue'

export default {
    data: () => ({
      headline: '',
      description: '',
      position: false, //{lat:123, lng:45.3},
      smsSpejder: '',

      teamId: '',
      severity: '',
      severities: {
        green: 'Grøn',
        yellow: 'Gul',
        red: 'Rød',
      },
      assignee: '',
      assignees: {
        guide: 'Guide',
        samarit: 'Samarit',
        sam1: 'Samarit 1',
        sam2: 'Samarit 2',
        sam3: 'Samarit 3',
        sam4: 'Samarit 4',
        rover: 'Rover',
        rover1: 'Rover 1',
        rover2: 'Rover 2',
        rover3: 'Rover 3',
        rover4: 'Rover 4',
        rover5: 'Rover 5',
        logistik: 'Logistik',
        hoens: 'Hønemor',
        team: 'Team',
      },
      memberStatuses: {
        active: 'Aktiv i løbet',
        waiting: 'Afventer transport',
        transit: 'Under transport',
        emergency: 'Skadestue',
        hq: 'Hønemor',
        out: 'Afhentet',
      },
      patruljer: [],
    }),
    components: { ActivityLine, BPopover },
    filters: {
        capitalize: function (value) {
            if (!value) return ''
            value = value.toString()
            return value.charAt(0).toUpperCase() + value.slice(1)
        },
        upper(v) {
          if (!v) return ''
          return v.toUpperCase()
        },
    },
    
    computed: {
      sos() {
        const sos = this.$store.getters['dims/sos'](this.$route.params.id) || {}
              /*
        this.teamId = ''
        this.severity = sos.severity
        this.assignee = sos.assignee
   */
        return sos
      },
      patruljer_store() {
        const patruljer = [];
        for (const p of this.$store.getters['dims/patruljer']) {
            patruljer.push({id: p.teamId, number: parseInt(p.teamNumber.split('-')[0]), name: p.teamNumber + ' ' + p.name})
        }
        return patruljer.sort((a, b) => (a.number > b.number ? 1 : -1))
      },
      smsDisabled() {
              console.log(this.smsSpejder)
        if (this.smsSpejder == '') return true
        return false
      },
    },
    methods: {
      patrulje(teamId) {
        return this.$store.getters['dims/patrulje'](teamId)
      },
      spejder(memberId) {
        for (const teamId in this.sos.teamIds || []) {
          for (const member of this.patrulje(teamId).members || []) {
            if (memberId == member.memberId) {
              return member
            }
          }
        }
        return this.$store.getters['dims/spejder'](memberId)
      },
      mergeTeams(parentTeamId, teamId) {
        this.$store.dispatch("dims/sosMergeTeams", {sosId: this.sos.sosId, parentTeamId: parentTeamId, teamId: teamId})
      },
      splitTeam(teamId) {
        this.$store.dispatch("dims/sosSplitTeam", {sosId: this.sos.sosId, teamId: teamId})
      },
      async addSosComment() {
        if (this.sos.sosId) {
          this.$store.dispatch("dims/addSosComment", {sosId: this.sos.sosId, comment:this.description}).then(response => {
            if (response && response.ok) this.description = ""
          })
          return
        }
        this.$store.dispatch("dims/createSos", {headline:this.headline, description:this.description}).then(response => {
          if (response && response.ok) {
            this.headline = ""
            this.description = ""
            this.$router.push({ name: "view-sos", params: { id: response.sosId }});
          }
        }, error =>{
          console.error("Got error", error)
        })
      },
      editHeadline() {
        this.headline = this.sos ? this.sos.headline : ''
      },
      cancelHeadline() {
        this.headline = ''
      },
      saveHeadline() {
        this.$store.dispatch("dims/updateSosHeadline", {sosId: this.sos.sosId, headline:this.headline}).then(response => { if (response) this.headline = ''})
      },
      closeSos() {
        if (!this.sos) return
        this.$store.dispatch("dims/closeSos", {sosId: this.sos.sosId})
      },
      reopenSos() {
        if (!this.sos) return
        this.$store.dispatch("dims/reopenSos", {sosId: this.sos.sosId})
      },
      associateTeam() {
        this.$store.dispatch("dims/sosAssociateTeam", {sosId: this.sos.sosId, teamId:this.teamId})
      },
      disassociateTeam(teamId) {
        this.$store.dispatch("dims/sosDisassociateTeam", {sosId: this.sos.sosId, teamId:teamId})
      },
      memberStatus(teamId, memberId, status) {
        this.$store.dispatch("dims/sosMemberStatus", {sosId: this.sos.sosId, teamId:teamId, memberId: memberId, status: status})
      },
      setSeverity() {
        this.$store.dispatch("dims/setSeveritySos", {sosId: this.sos.sosId, severity:this.severity})
      },
      assign() {
        this.$store.dispatch("dims/assignSos", {sosId: this.sos.sosId, assignee:this.assignee})
      },
      sendPositionSms() {
        this.$store.dispatch("dims/sosSendPositionSms", {sosId: this.sos.sosId, memberId: this.smsSpejder})
      },
      mergeableTeams(srcTeamId) {
        const teams = []
        const srcTeam = this.patrulje(srcTeamId)
        for (const teamId in this.sos.teamIds || []) {
          const team = this.patrulje(teamId)
          if (team && !team.parentTeamId && srcTeam.parentTeamId != teamId && srcTeamId != teamId) {
            teams.push(team)
          }
        }
        return teams
      },
    },
    async mounted() {
      const response = await axios.get('/api/patruljer', { withCredentials: true });
      console.log(response)
      if (response.status != 200) {
          console.log("failed statuscode:", response.statusCode)
          return
      }
        for (const p of response.data.patruljer) {
            this.patruljer.push({id: p.id, number: p.number, name: p.number + '-' + p.memberCount + ' ' + p.name})
        }
        //return patruljer.sort((a, b) => (a.number > b.number ? 1 : -1))
      //response.data.patruljer.sort((a, b) => (a.number > b.number ? 1 : -1))
      //this.patruljer = response.data.patruljer
        console.log(this.patruljer)
    },
    beforeDestroy() {
    }
}
</script>
