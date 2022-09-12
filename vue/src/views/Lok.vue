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
    </div>
</template>

<style>
.vgt-global-search {
    border:0;
    background:none;
}
</style>

<script>
export default {
    data: () => ({
      date:{},
      title: 'Nathejk 2019',
      team: {},
      columns: [
        {label: 'Navn', field: 'name'},
        {label: 'Telefon', field: 'phone'},
        {label: 'Hold', field: 'group'},
        {label: 'Antal scanninger', field: 'scanCount', type:'number'},
      ],
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
        loks() {
            //return this.$store.getters['dims/users']
            const loks = {}
                /*
            for (const klan of this.$store.getters['dims/klan']) {
                const label = this.groupSlugs[user.group] || 'Andet'
                if (!users[label]) {
                    users[label] = []
                }
                users[label].push(user)

            }
          */      
            
            const groups = []
            for (const group of this.groupOptions) {
                if (loks[group.label]) {
                    groups.push({mode:'span', label:group.label, children: loks[group.label]})
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
