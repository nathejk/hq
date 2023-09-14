<template>
    <div class="container">
            <div class="row">
  <b-card no-body class="w-100 my-3">
    <b-card-header header-tag="nav" class="">
      <b-nav card-header tabs>
        <b-nav-item v-for="(dep, i) in pending" :key="`department-${i}`" class="midnightblue" :active="i == selected" @click="select(i)">{{ dep.name }}<i v-if="dirty(dep)" class="fas fa-fw fa-circle dirty"></i></b-nav-item>
        <b-nav-form @submit.stop.prevent="add" class="px-2">
          <b-button class="btn-sm" variant="outline-secondary" type="submit"><i class="fas fa-fw fa-plus"></i></b-button>
        </b-nav-form>
      </b-nav>
    </b-card-header>

    <b-card-body class="px-0">
    <b-row class="my-1 px-3">
      <b-col sm="2">
        <label class="m-0 pt-1" for="department-name">Funtionsnavn:</label>
      </b-col>
      <b-col sm="10">
        <b-form-input id="department-name" type="text" v-model="department.name"></b-form-input>
      </b-col>
    </b-row>

    <b-row class="px-3">
      <b-col sm="2">
        <label class="m-0" for="department-name">Funtionsbeskrivelse:</label>
      </b-col>
    </b-row>
    <b-row class="px-3">
      <b-col>
        <small class="text-muted">Denne funktionsbeskrivelse bliver vist på hej.nathejk.dk</small>
      </b-col>
    </b-row>
    <b-row class="m-0 px-3">
      <b-col class="px-0">
        <markdown-editor v-model="department.desc" class="markdown" toolbar="bold italic strikethrough heading | image link | numlist bullist code quote | preview fullscreen"></markdown-editor>
      </b-col>
    </b-row>
    </b-card-body>

    <b-card-footer class="text-right">
      <a href="" class="delete" @click.prevent="del(department)">Slet funktion</a>
      <b-button :disabled="!dirty(department) || !department.id" variant="secondary ml-3" @click="reset(department)">Gendan</b-button>
      <b-button :disabled="!dirty(department)" variant="success ml-3" @click="save(department)">Gem</b-button>
    </b-card-footer>
  </b-card>
            </div>
    </div>
</template>

<style>
.midnightblue a:hover { color:#445e65; }
.midnightblue a { color:#a2aeb2; }
a.delete {color:#445e65; }
a.delete:hover {color:#900; }
.markdown .mr-5 { margin-right: 1rem !important; }
i.dirty { olor:#900; vertical-align:top; font-size:0.5rem; }
a:hover i.dirty { olor:#d00; }
</style>

<script>
import { BModal, BTable } from 'bootstrap-vue'
import 'v-markdown-editor/dist/v-markdown-editor.css';

import Vue from 'vue'
import Editor from 'v-markdown-editor'

// global register
Vue.use(Editor);


export default {
    data: () => ({
      selected: 0,
      departments: [],
      pending: [],
    }),
    components: { BModal, BTable, Editor },
    computed: {
        department: function() {
            if (this.selected < this.pending.length) {
                return this.pending[this.selected]
            }
            return {}
        },
    },
    methods: {
        select: function(i) {
            this.selected = i
        },
        add: function() {
            this.selected = this.pending.length
            this.pending.push({name:'Funktion', desc:'', id:'', initId:this.$uuid.v4()})
        },
        del: function (department) {
            if (!confirm('DANGER ZONE!!\nEr du sikker på at du vil slette denne funktion i Nathejk')) return
            this.$store.dispatch("department/del", department);
        },
        reset: function(department) {
            const clean = this.$store.getters['department/department'](department.id)
            if (!clean) return;
            for (let [key, value] of Object.entries(department)) {
                department[key] = clean[key]
            }
        },
        save: function(department) {
            this.$store.dispatch("department/update", department)
        },
        dirty: function(department) {
            const clean = this.$store.getters['department/department'](department.id)
            if (!clean || Object.keys(clean).length === 0) return true;
            for (let [key, value] of Object.entries(clean)) {
                if (value != department[key]) return true
            }
            return false
        },
    },
    watch: {
        departments (newCount, oldCount) {
            // Our fancy notification (2).
            console.log(`We have ${newCount} fruits now, yay!`)
        }
    },
    mounted: function () {
        this.departments = this.$store.getters['department/departments']
        this.pending = []
        for (const department of this.departments) {
            this.pending.push({ ...department })
        }
    },
    beforeDestroy() {
    }
}
</script>
