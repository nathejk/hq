import axios from 'axios';
//import Vue from 'vue';


// initial state
const state = {
    departments: [
          {id:'12131434', name:'Guides', dirty:false, desc:'#Hej Guide'},
          {id:'435542f5', name:'Banditter', dirty:false, desc:'# Hej Bandit'},
          {id:'4326f436', name:'Teknisk tjeneste', dirty:false, desc:''},
          {id:'54216146', name:'Logistik', dirty:false, desc:''},
          {id:'24526642', name:'Postmandskab', dirty:false, desc:''},
          {id:'52632466', name:'PR', dirty:false, desc:''},
    ],
}

const getters = {
    departments: (state) => state.departments,
    department: (state) => (id) => {
        for (const d of state.departments) {
            if (d.id == id) return d
        }
        return {}
    },
    //soses: (state) => state.ids['sos'].map( id => state.list['sos'][id] ),
}

// actions
const actions = {
  create({commit}, payload) {
    return axios
      .put('/api/department', payload, { withCredentials: true })
      .then((response) => { return response.data })
  },
  del({commit}, payload) {
    //commit("DELETE_DEPARTMENT", payload.id);
    return axios
      .delete('/api/department', { withCredentials: true, data: payload })
      .then((response) => { return response.data })
  },
  update({commit}, payload) {
    //commit("UPDATE_DEPARTMENT", payload);
    return axios
      .post('/api/department', payload, { withCredentials: true })
      .then((response) => { return response.data })
  },
}

// mutations
const mutations = {
  UPDATE_DEPARTMENT(state, department) {
    console.log('UPDATE_DEPARTMENT', department)
    for (const i in state.departments) {
        if (state.departments[i].id == department.id) {
            for (let [key, value] of Object.entries(department)) {
                state.departments[i][key] = value
            }
            return
        }
    }
    // new
    const dep = {}
            for (let [key, value] of Object.entries(department)) {
                dep[key] = value
            }
      dep.id = ''+Math.random()
        state.departments.push(dep)
  },
  DELETE_DEPARTMENT(state, id) {
    console.log('DELETE_DEPARTMENT', id)
    for (const i in state.departments) {
        if (state.departments[i].id == id) {
            state.departments.splice(i, 1)
        }
    }
  },

}

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}
