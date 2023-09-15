<template>
    <div class="container p-3">
        <vue-good-table ref="teamlist" styleClass="vgt-table condensed"
            :columns="columns"
            :rows="Members"
            @on-row-click="onRowClick"
            @on-selected-rows-change="selectionChanged"
            :search-options="{enabled: true}"
            :group-options="{enabled: true}"
            >
            <div slot="emptystate">
                No records found
            </div>
        </vue-good-table>
    </div>
</template>

<style>
td, th {
    font-size:0.8rem;
}
.btn.disabled, .btn:disabled {
    opacity: 0.3;
}
.vgt-global-search {
    border:0;
    background:none;
}

</style>

<script>
//import axios from 'axios';
//import moment from 'moment'
//import 'vue-good-table/dist/vue-good-table.css'
//import { VueGoodTable } from 'vue-good-table';

export default {
    data: () => ({
        columns: [
            {label: 'Spejder', field: 'name'},
            {label: 'Patrulje', field: 'patrulje.name'},
            {label: 'Siden', field: 'since'},
        ],
        teams: [],
        members: [],
        selectedCount: 0,
        groupings: [
            { label:'Tilmeldingsstatus', groupby:'signupStatusTypeName', grouporder:[{key:'PAY', label:'Afventer betaling'},{key:'PAID', label:'Tilmeldt'}], defaultcolumns:[] },
        ],
    }),
    components: {
  //      VueGoodTable,
    },
    computed: {
        grouping() {
            return this.groupings[0]
        },
        Members() {
            const groups = {}
            for (const spejder of this.$store.getters['dims/spejdere']) {
                if (spejder.status == 'active') {
                    continue
                }
                if (!groups[spejder.status]) {
                    groups[spejder.status] = []
                }
                spejder.patrulje = this.$store.getters['dims/patrulje'](spejder.teamId)
                groups[spejder.status].push(spejder)
            }
/*
            if (!this.members.length) return []

            const groups = {
                afhentet:[],
                hq:[],
                out:[],
            }
            for (const member of this.members) {
                if (member.discontinuedUts > 0) {
                    member.since = moment(Number(member.discontinuedUts)*1000).format('ddd [kl.] H:mm:ss')
                    member.team = this.team(member.teamId)
                    groups.afhentet.push(member)
                } else if(member.inhqUts > 0) {
                    member.since = member.inhqUts
                    member.since = moment(Number(member.inhqUts)*1000).format('ddd [kl.] H:mm:ss')
                    member.team = this.team(member.teamId)

                    groups.hq.push(member)
                } else if (member.pausedUts > 0) {
                    member.since = member.pausedUts
                    member.since = moment(Number(member.pausedUts)*1000).format('ddd [kl.] H:mm:ss')
                    member.team = this.team(member.teamId)
                        groups.out.push(member)
                }
            }*/
            const members = [
                {mode:'span', label:'Afventer transport', children:groups.waiting || []},
                {mode:'span', label:'Under transport', children:groups.transit || []},
                {mode:'span', label:'Skadestue', children:groups.emergency || []},
                {mode:'span', label:'Hønemor', children:groups.hq || []},
                {mode:'span', label:'Afhentet', children:groups.out || []},
            ]
                /*
            for (const order of this.grouping.grouporder) {
                const group = groups[order.key]
                group.label = order.label + ' (' + group.children.length + ')'
                teams.push(group)
            }*/
            return members
        },
        Teams() {
            if (!this.teams.length) return []

            const groups = {}
            for (const team of this.teams.filter(team => team.signupStatusTypeName != 'NEW')) {
                if (!groups[team[this.grouping.groupby]]) {
                    groups[team[this.grouping.groupby]] = {mode:'span', label: team[this.grouping.groupby], children:[]}
                }
                groups[team[this.grouping.groupby]].children.push(team)
            }
            const teams = []
            for (const order of this.grouping.grouporder) {
                const group = groups[order.key]
                group.label = order.label + ' (' + group.children.length + ')'
                teams.push(group)
            }
            return teams
        },
        selectedTeams() {
            if (!this.$refs['teamlist']) return []
            return this.$refs['teamlist'].selectedRows
        }
    },
    methods: {
      selectionChanged(params) {
          console.log('selectionChanged', params)
          this.selectedCount = this.$refs.teamlist.selectedRows.length
      },
        openMember(member) {
            this.member = member
           // $('#memberModal').modal({})
        },
      onRowClick(params) {

        // If click-event originates from the checkbox column then ignore
        for (let el = params.event.target; el && el.nodeName != 'TR'; el = el.parentNode) {
          for (const className of el.classList) {
            if (className == 'vgt-checkbox-col') return
          }
        }
        this.$router.push({ name: "patrulje", params: { id: params.row.teamId }});
      },
      toggleColumn( index, event ){
          event.preventDefault()
          event.stopPropagation()
        // Set hidden to inverse of what it currently is
        this.$set( this.columns[ index ], 'hidden', ! this.columns[ index ].hidden );
      },
      team(teamId) {
        for (const team of this.teams) {
          if (team.id == teamId) {
            return team
          }
        }
        return {}
      },
    },
        /*
    async mounted() {
        try {
            const rsp = await axios.get('/api/teams',
            { withCredentials: true }
            )
            if (rsp.status == 200) {
                this.teams = rsp.data.teams
                if (rsp.data.members) {
                    this.members = rsp.data.members
                }
            }
        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
    },
    /*
    mounted() {
        axios.get('/api/teams',
            { withCredentials: true }
        ).then((rsp) => {
            var teams = {}
            for (const team of rsp.data.teams) {
                teams[team.id] = team
            }
            var rows = []
            for (const member of rsp.data.members) {
                rows.push(Object.assign({}, member, {
                    'name': member.title,
                    'team': teams[member.teamId] ? teams[member.teamId].title : '',
                }))
            }
            this.rows = rows
            //this.rows = [{ id:6, name:"John Måwensen", age: 20, createdAt: '2011-10-31', score: 0.03343 }]
        });
    },//*/
}
</script>
