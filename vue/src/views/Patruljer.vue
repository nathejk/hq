<template>
    <div class="p-3">
        <vue-good-table ref="teamlist" styleClass="vgt-table condensed"
            :columns="columns"
            :rows="Teams"
            @on-row-click="onRowClick"
            @on-selected-rows-change="selectionChanged"
            :search-options="{enabled: true}"
            :select-options="{enabled: true, selectOnCheckboxOnly: true, disableSelectInfo: true}"
            :group-options="{enabled: true}"
            >
            <div slot="table-actions">
                <div class="btn-group" role="toolbar" >
                    <button type="button" class="btn btn-sm btn-outline-dark" :disabled="!selectedCount"><i style="color: rgb(221, 27, 22);" class="fa fa-fw fa-trash-alt"></i> slet</button>
                    <button type="button" class="btn btn-sm btn-outline-dark" :disabled="!selectedCount"><i class="fa fa-fw fa-at"></i> e-mail</button>
                    <button type="button" class="btn btn-sm btn-outline-dark" :disabled="!selectedCount"><i style="color: rgb(65, 184, 131);" class="fa fa-fw fa-money-bill-wave"></i> indbetaling</button>
                    <button type="button" class="btn btn-sm btn-outline-dark"><i style="color: rgb(65, 184, 131);" class="fa fa-fw fa-file-excel"></i> eksport</button>
                    <button type="button" class="btn btn-sm btn-outline-dark dropdown-toggle" data-toggle="dropdown"><i class="fa fa-fw fa-cog" aria-hidden="true"></i> <span class="caret"></span></button>
                    <ul class="dropdown-menu" style="top: auto; left: auto;">
                        <li v-for="(column, index) in columns" :key="index">
                            <a href="#" class="dropdown-item" tabIndex="-1" @click.prevent="toggleColumn( index, $event )"><input :checked="!column.hidden" type="checkbox"/>&nbsp;{{column.label}}</a>
                        </li>
                    </ul>
                </div>
            </div>
            <div slot="emptystate">
                No records found
            </div>
        </vue-good-table>
    </div>
</template>

<style>

.vgt-table td, .vgt-table th {
    font-size:0.8rem;
}
.btn.disabled, .btn:disabled {
    opacity: 0.3;
}
.vgt-global-search {
    border:0;
    background:none;
}

.vgt-row-header > span {
    padding-left:35px;
}
</style>

<script>
//import 'vue-good-table/dist/vue-good-table.css'
//import { VueGoodTable } from 'vue-good-table';

export default {
    data: () => ({
        columns: [
            {label: 'ID', field: 'id'},
            {label: 'Nr', field: 'teamNumber'},
            {label: 'Patrulje', field: 'name'},
            {label: 'Gruppe', field: 'groupName'},
            {label: 'Korps', field: 'korps'},
            {label: 'Antal', field: 'memberCount', sortable: false},
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
        Teams() {

        const groups = {
          active: {mode:'span', label:'Aktive patruljer', children:[]},
          merged: {mode:'span', label:'Sammenlagte', children:[]},
          stopped: {mode:'span', label:'Udgåede patruljer', children:[]},
        }
        for (const patrulje of this.$store.getters['dims/patruljer']) {
                console.log(patrulje)
          groups[patrulje.activeMembers.length == 0 ? 'stopped' : (patrulje.parentTeamId ? 'merged' : 'active')].children.push(patrulje)
        }
        return [groups.active, groups.merged, groups.stopped]
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
      onRowClick(params) {

        // If click-event originates from the checkbox column then ignore
        for (let el = params.event.target; el && el.nodeName != 'TR'; el = el.parentNode) {
          for (const className of el.classList) {
            if (className == 'vgt-checkbox-col') return
          }
        }
              console.log('row', params.row)
        this.$router.push({ name: "patrulje", params: { id: params.row.teamId }});
      },
      toggleColumn( index, event ){
          event.preventDefault()
          event.stopPropagation()
        // Set hidden to inverse of what it currently is
        this.$set( this.columns[ index ], 'hidden', ! this.columns[ index ].hidden );
      }
    },
    /*
    async mounted() {
        try {
            const rsp = await axios.get(window.envConfig.API_BASEURL + '/api/teams',
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
    mounted() {
        axios.get(window.envConfig.API_BASEURL + '/api/teams',
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
