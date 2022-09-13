import Vue from 'vue'
import App from './App.vue'
import router from './router'
import "./filters"
import { store, initializeStores } from "./store";

import { BootstrapVue } from 'bootstrap-vue'
Vue.use(BootstrapVue)
// Import Bootstrap an BootstrapVue CSS files (order is important)
import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue/dist/bootstrap-vue.css'


Vue.config.productionTip = false

import VueMoment from 'vue-moment'
Vue.use(VueMoment)

import VueGoodTablePlugin from 'vue-good-table';
import 'vue-good-table/dist/vue-good-table.css'
Vue.use(VueGoodTablePlugin);

new Vue({
  render: h => h(App),
  router,
  store,
  beforeCreate() {
    initializeStores.apply(this)
  },
}).$mount('#app')
