<template>
    <div class="container p-3">
        <h1 class="mb-2 clearfix">
            <span class="h2 mb-0"><span class="badge bg-secondary text-uppercase easy-select">{{ team.teamNumber || '&times;&times;-&times;' }}</span></span>
            <span class="h3 mb-0 ps-3 d-block d-md-inline">
                <small class="text-uppercase text-muted pl-3">{{ team.name }}<small class="ms-1"></small></small>
            </span>
        </h1>
        <hr />
        <div class="row"><div class="col">
        <div class="card">
          <div class="card-header"><i class="fas fa-fw fa-users darkblue"></i> Patrulje</div>
          <div class="card-body">
            <div class="small border rounded  bg-light pt-2 mb-3">
            <button class="btn btn-sm btn-outline-danger float-right mr-2" data-toggle="modal" data-target="#teamModal"><i class="fas fa-pencil-alt"></i> ret</button>
            <dl class="row pb-0 mb-0">
              <dt class="col-2 small text-uppercase grey">Patrulje</dt>
              <dd class="col-10">{{ team.name }}</dd>

              <dt class="col-2 small text-uppercase grey">Gruppe / Division</dt>
              <dd class="col-10">{{ team.groupName }}</dd>

              <dt class="col-2 small text-uppercase grey">Korps</dt>
              <dd class="col-10">{{ team.korps | korps }}</dd>

              <dt class="col-2 small text-uppercase grey">Adv.spejd liga ID</dt>
              <dd class="col-10">{{ team.ligaNumber || '-' }}</dd>

              <dt class="col-2 small text-uppercase grey">Status</dt>
              <dd class="col-10">{{ team.signupStatusTypeName }}</dd>
            </dl>
            </div>
  <Modal id="teamModal" iconClass="fas fa-users" title="Ret patruljeoplysninger">
    <form>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamName">Navn</label>
        <div class="col-sm-10"><input type="text" class="form-control" id="teamName" :value="team.title"></div>
      </div>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamGruppe">Gruppe / divi</label>
        <div class="col-sm-10"><input type="text" class="form-control" id="teamGruppe" :value="team.gruppe"></div>
      </div>
    </form>
  </Modal>
            <table class="table table-sm" style="font-size:0.8rem">
              <thead>
                <tr>
                  <th scope="col">Medlem</th>
                  <th scope="col">Adresse</th>
                  <th scope="col">Mail</th>
                  <th scope="col">Telefon</th>
                  <th scope="col">Pårørende Telefon</th>
                  <th scope="col">Alder ved start</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(member, i) in team.members" :key="i" @click="openMember(member)">
                  <td>{{ member.name }}</td>
                  <td>{{ member.address }}, {{ member.postalCode }}</td>
                  <td>{{ member.mail }}</td>
                  <td>{{ member.phone }}</td>
                  <td>{{ member.phoneParent }}</td>
                  <td>{{ member.birthDate }}</td>
                </tr>
              </tbody>
            </table>
  <Modal id="memberModal" iconClass="fas fa-user" title="Patruljemedlem">
    <form>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamName">Navn</label>
        <div class="col-sm-10"><input type="text" class="form-control" id="teamName" :value="member.title"></div>
      </div>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamGruppe">Adresse</label>
        <div class="col-sm-10"><input type="text" class="form-control" id="teamGruppe" :value="member.address"></div>
      </div>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamGruppe">Postnr. / By</label>
        <div class="col-sm-10"><input type="text" class="form-control" id="teamGruppe" :value="member.address"></div>
      </div>
      <div class="form-group row">
        <label class="col-sm-2 col-form-label" for="teamGruppe">Status</label>
        <div class="col-sm-10"><select class="form-control form-control-sm" id="modalFunction">
          <optgroup v-for="(group, name) in memberStati" :label="name" :key="name">
            <option v-for="(option, i) in group" :key="i" :value="option.value" :selected="member.role==option.value">{{ option.text }}</option>
          </optgroup>
        </select></div>
      </div>
    </form>
  </Modal>
          </div>
        </div>
        </div></div>

        <div class="row mt-3">
          <div class="col">
            <div class="card">
              <div class="card-header d-flex justify-content-between align-items-center">
                <span><i class="fas fa-fw fa-envelope-open-text text-info"></i> Mails</span>
              </div>
              <div class="card-body"  style="font-size:0.8rem">
                <ul class="list-group list-group-flush">
                  <li v-for="mail in team.mails" @click="showMail(mail)" role="button" class="px-0 py-1 list-group-item list-group-item-action d-flex justify-content-between align-items-center" :key="mail.id">
                    {{ mail.subject }}
                    <span class="text-nowrap">
                      <span class="small grey" :title="mail.sendUts | dateFull">{{ mail.sendUts | dateDM }}</span>
                      <i class="fa fa-chevron-right lightgrey ml-2"></i>
                    </span>
                  </li>
                  <li class="px-0 py-1 list-group-item"><a href="">ny mail...</a></li>
                </ul>
              </div>
            </div>
          </div>
  <Modal id="mailModal" iconClass="fas fa-envelope-open" title="E-mail">
    <small class="float-right">{{ mail.sendUts | dateFull }}</small>
    <dl class="row">
      <dt class="col-2">Fra</dt>
      <dd class="col-10">{{ mail.mailFrom }}</dd>

      <dt class="col-2">Til</dt>
      <dd class="col-10">
        <p vfor="rcpt in mail.rcptTo">{{ mail.rcptTo }}</p>
      </dd>

      <dt class="col-2">Emne:</dt>
      <dd class="col-10">{{ mail.subject }}</dd>
    </dl>
    <pre class="border border-dark rounded bg-light small text-muted p-2">{{ mail.body }}</pre>
  </Modal>
          <div class="col">
            <div class="card">
              <div class="card-header">
                <span><i class="fas fa-fw fa-money-bill-alt text-success"></i> Betalinger</span>
                <span class="fa-stack fa-xs">
  <i class="fa fa-circle fa-stack-2x icon-background text-secondary"></i>
  <i class="fas fa-plus fa-stack-1x text-light"></i>
</span>
              </div>
              <div class="card-body" style="font-size:0.8rem">
                <ul class="list-group list-group-flush">
                  <li v-if="team.payments" class="px-0 py-1 list-group-item">
                    <div v-for="(payment, i) in team.payments" :key="i" class="d-flex justify-content-between align-items-center">
                    DKK {{ payment.amount }},-
                    <span class="" :title="payment.uts | dateFull">{{ payment.uts | dateDM }}</span>
                    </div>
                  </li>
                  <li class="px-0 py-1 list-group-item d-flex justify-content-between align-items-center">
                      DKK {{ team.paidAmount }},- <span>Betalt i alt</span>
                  </li>
                  <li class="px-0 py-1 list-group-item d-flex justify-content-between align-items-center">
                      DKK {{ team.teamPrice }},- <span>Pris for hold</span>
                  </li>
                  <li class="px-0 py-1 list-group-item d-flex justify-content-between align-items-center">
                      <span :class="{'text-danger':team.unpaidAmount>0}" >DKK {{ team.unpaidAmount }},-</span><span>At betale</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          <div class="col">
            <div class="card">
              <div class="card-header"><i class="fas fa-fw fa-user-tie text-secondary"></i> Kontaktperson</div>
              <div class="card-body" style="font-size:0.8rem">
                  <p>
                    <i class="fas fa-fw fa-user"></i> {{ team.contactRole }}<br>
                    <i class="fas fa-fw"></i> {{ team.contactTitle }}<br>
                    <i class="fas fa-fw"></i> {{ team.contactAddress }}<br>
                    <i class="fas fa-fw"></i> {{ team.contactPostalCode }}<br>
                  </p>
                  <p>
                    <i class="fas fa-fw fa-at"></i> {{ team.contactMail }}<br>
                    <i class="fas fa-fw fa-mobile-alt"></i> {{ team.contactPhone }}<br>
                  </p>
              </div>
            </div>
          </div>

          <div class="col">
            <div class="card">
              <div class="card-header"><i class="fas fa-fw fa-info text-warning"></i> Info</div>
              <div class="card-body"  style="font-size:0.8rem">
                <div class="d-flex justify-content-between pb-2">
                  <span>Tilmeldt</span>
                  <span>{{ team.finishedUts | dateFull }}</span>
                </div>

                <div class="d-flex justify-content-between pb-2">
                  <span>Afmeldt</span>
                  <span>{{ team.startUts | dateFull }}</span>
                </div>
                <div class="d-flex justify-content-between pb-2">
                  <span>Startet</span>
                  <span>{{ team.startUts | dateFull }}</span>
                </div>
                <div class="d-flex justify-content-between pb-2">
                  <span>Udgået</span>
                  <span>{{ team.startUts | dateFull }}</span>
                </div>
                <div class="d-flex justify-content-between pb-2">
                  <span>Sluttet</span>
                  <span>{{ team.startUts | dateFull }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
        <form enctype="multipart/form-data" action="" method="post">

        </form>
    </div>
</template>

<style lang="scss">
.badge {
    font-weight:400;
}
</style>

<script>
import Modal from '@/components/Modal.vue'
//import moment from 'moment'


export default {
    components: { Modal },
    data: () => ({
        selected: 'dds',
            memberStati: {
                    Tilmeldt:[
                {text:'Oprettet', value:''},
                {text:'Startet', value:''},
                {text:'Gennemført', value:''},
                ],
            'Udgået': [
                {text:'Afventer transport', value:''},
                {text:'Under transport', value:''},
                {text:'Skadestue', value:''},
                {text:'Hønemor', value:''},
                {text:'Afhentet', value:''},
                ]},
        korps: [
          { text: 'Det Danske Spejderkorps', value: 'dds' },
          { text: 'KFUM-Spejderne', value: 'kfum' },
          { text: 'De grønne pigespejdere', value: 'kfuk' },
          { text: 'Danske Baptisters Spejderkorps', value: 'dbs' },
          { text: 'De Gule Spejdere', value: 'dgs' },
          { text: 'Dansk Spejderkorps Sydslesvig', value: 'dss' },
          { text: 'FDF / FPF', value: 'fdf' },
          { text: 'Andet', value: 'andet' },
        ],
        //team: {},
        mail: {},
        member: {},
    }),
    computed: {
        paidUts() {
            let paid = 0
            return paid
        },
        team() {
            return this.$store.getters['dims/patrulje'](this.$route.params.id) || {}
        },
    },
    methods: {
        showMail(mail) {
            this.mail = mail
 //           $('#mailModal').modal({})
        },
        openMember(member) {
            this.member = member
 //           $('#memberModal').modal({})
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
        /*
    async mounted() {
        try {
            const rsp = await axios.get(window.envConfig.API_BASEURL + '/api/teams/' + this.$route.params.id,
            { withCredentials: true }
            )
            if (rsp.status == 200) {
                this.team = rsp.data.team
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
    },*/
}
</script>
