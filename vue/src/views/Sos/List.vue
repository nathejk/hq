<template>
    <div class="container py-3">
        <vue-good-table ref="soses" styleClass="vgt-table condensed"
            :columns="columns"
            :rows="soses"
            @on-row-click="onRowClick"
            @on-selected-rows-change="selectionChanged"
            :search-options="{enabled: true}"
            :group-options="{enabled: true}"
            >
            <div slot="table-actions">
                <div class="btn-group" role="toolbar" >
                    <button class="btn btn-sm btn-outline-success float-right mr-2" @click="newSos"><i class="fas fa-plus"></i> ny</button>
                </div>
            </div>
            <div slot="emptystate">
                Ingen nødråb fundet
            </div>
        </vue-good-table>
    </div>
</template>

<style>
</style>

<script>
export default {
    data: () => ({
        columns: [
            {label: 'Overskrift', field: 'headline'},
            {label: 'Oprettet', field: 'createdAt', type:'date', dateInputFormat: 'yyyy-MM-dd\'T\'HH:mm:ss.SSSSSSSSSX', dateOutputFormat: 'ccc HH:mm'},
            {label: 'Sidst opdateret', field: 'lastActivityAt', type: 'date', dateInputFormat: 'yyyy-MM-dd\'T\'HH:mm:ss.SSSSSSSSSX', dateOutputFormat: 'ccc HH:mm'},
            {label: 'Prioritet', field: 'severity', tdClass: 'text-center'},
            {label: 'Tildelt', field: 'assignee'},
        ],
    }),
    filters: {
        name: function (teamId, teams) {
            return teams[teamId] ? teams[teamId] : ''
                /*
            console.log(teams, teamId)
            return 'cd';*/
        }

    },
    
    computed: {
      soses() {
        const groups = {
          open: {mode:'span', label:'Åbne sager', children:[]},
          closed: {mode:'span', label:'Lukkede sager', children:[]},
        }
        for (const sos of this.$store.getters['dims/soses']) {
          groups[sos.closed ? 'closed' : 'open'].children.push(sos)
        }
        return [groups.open, groups.closed]
      },
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
        this.$router.push({ name: "view-sos", params: { id: params.row.sosId }});
      },
      newSos() {
        this.$router.push({ name: "new-sos"});
      },
    },
    mounted: function () {
    },
    beforeDestroy() {
    }
}
</script>
