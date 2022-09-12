<template>
    <div class="p-3">
        <vue-good-table styleClass="vgt-table condensed"
            :columns="columns"
            :rows="rows"
            :search-options="{
                enabled: true
            }"/>
    </div>
</template>

<style>
</style>

<script>
import axios from 'axios';
//import 'vue-good-table/dist/vue-good-table.css'
//import { VueGoodTable } from 'vue-good-table';

export default {
    data: () => ({
        title: 'Nathejk 2019',
        feed: null,
        columns: [
        {
          label: 'Name',
          field: 'name',
        },
        {
          label: 'E-mail',
          field: 'mail',
        },
        {
          label: 'Telefon',
          field: 'phone',
        },
        {
          label: 'Hold',
          field: 'team',
        },
        {
          label: 'holdid',
          field: 'teamId',
        },
      ],
      rows: [
        { id:1, name:"John", age: 20, createdAt: '',score: 0.03343 },
        { id:2, name:"Jane", age: 24, createdAt: '2011-10-31', score: 0.03343 },
        { id:3, name:"Susan", age: 16, createdAt: '2011-10-30', score: 0.03343 },
        { id:4, name:"Chris", age: 55, createdAt: '2011-10-11', score: 0.03343 },
        { id:5, name:"Dan", age: 40, createdAt: '2011-10-21', score: 0.03343 },
        { id:6, name:"John", age: 20, createdAt: '2011-10-31', score: 0.03343 },
      ],
    }),
    components: {
  //      VueGoodTable,
    },
    computed: {
        icolumns() { return [
            {label: 'Navn', field: 'title'},
            {label: 'Email', field: 'mail'},
            {label: 'Telefon', field: 'phone'},
        ]}, 
        irows() { return this.feed.members },
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
    },
}
</script>
