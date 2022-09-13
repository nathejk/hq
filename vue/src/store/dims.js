import axios from 'axios';
//import Vue from 'vue';


// initial state
const state = {
  ws: null, // Our websocket
  status: '',
  cache: {},
  list: Object,
  ids: {
    users:[],
    controlgroup:[],
    klan:[],
    patrulje:[],
    spejder:[],
    sos:[],
  },
  lastModify: {
    sos: null,
    klan: null,
    patrulje: null,
    spejder: null,
  }
}

const getters = {
    soses: (state) => state.ids['sos'].map( id => state.list['sos'][id] ),
    sos: (state) => (id) => {
        if (!state.lastModify['sos']) {
            return {}
        }
        return state.list['sos'][id]
    },
    websocketStatus(state) { 
        if (!state.ws) return 'uninitialized'
        return state.status
    },
    users: (state) => state.ids['users'].map( id => state.list['users'][id] ),
    controlGroups: (state) => state.ids['controlgroup'].map( id => state.list['controlgroup'][id] ),
    klans: (state) => state.ids['klan'].map( id => state.list['klan'][id] ),
    klan: (state) => (id) => {
        if (!state.lastModify['klan']) {
            return {}
        }
        return state.list['klan'][id]
    },
    patruljer: (state) => state.ids['patrulje'].map( id => state.list['patrulje'][id] ),
    patrulje: (state) => (id) => {
        if (!state.lastModify['patrulje']) {
            return {}
        }
        return state.list['patrulje'][id] ||{}
    },
    spejdere: (state) => state.ids['spejder'].map( id => state.list['spejder'][id] ),
    spejder: (state) => (id) => {
        if (!state.lastModify['spejder']) {
            return {}
        }
        return state.list['spejder'][id] ||{}
    },
    //controlGroups: (state) => state.controlgroups
}

/** template:
  closeSos({ commit, state }, sos) {
    return axios
      .post(window.envConfig.API_BASEURL + '/api/sos/close', sos, { withCredentials: true })
      .then((response) => { return response.data })
      .catch((error) => {})
  },
*/
// actions
const actions = {
  async createSos({commit}, sos) {
    return axios
        .put('/api/sos', sos, { withCredentials: true })
        .then((response) => {
          //commit('logUserIn', response.data);
            return response.data
        })
  },
  async addSosComment({commit}, sos) {
    return axios
      .put('/api/sos/comment', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  updateSosHeadline({commit}, sos) {
      console.log('updateSosHeadline(sos)', sos)
    return axios
      .post('/api/sos/headline', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  closeSos({commit}, sos) {
    return axios
      .post('/api/sos/close', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  reopenSos({commit}, sos) {
    return axios
      .post('/api/sos/reopen', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  sosAssociateTeam({commit}, sos) {
    return axios
      .post('/api/sos/team', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  sosDisassociateTeam({commit}, sos) {
    return axios
      .delete('/api/sos/team', { withCredentials: true, data: sos })
      .then((response) => { return response.data })
  },
  sosMergeTeams({commit}, sos) {
    return axios
      .post('/api/sos/merge', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  sosSplitTeam({commit}, sos) {
    return axios
      .post('/api/sos/split', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  sosMemberStatus({commit}, sos) {
    return axios
      .post('/api/sos/member', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  setSeveritySos({commit}, sos) {
    return axios
      .post('/api/sos/severity', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  assignSos({commit}, sos) {
    return axios
      .post('/api/sos/assign', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  sosSendPositionSms({commit}, sos) {
    return axios
      .post('/api/sos/sms', sos, { withCredentials: true })
      .then((response) => { return response.data })
  },
  async updateControlGroup({ commit }, ctrlgrp) {
    commit("UPDATE_CONTROL_GROUP", ctrlgrp);
  },
  async deleteControlGroup({ commit }, id) {
    commit("DELETE_CONTROL_GROUP", id);
  },
}

// mutations
const mutations = {
  CREATE_SOS(state, sos) {
    console.log('SOS', sos)
    //state.user.name = name;
  },
  async UPDATE_CONTROL_GROUP(state, ctrlgrp) {
    try {
      var rsp;
      if (ctrlgrp.controlGroupId) {
        rsp = await axios.post('/api/controlgroup', ctrlgrp, { withCredentials: true })
      } else {
        rsp = await axios.put('/api/controlgroup', ctrlgrp, { withCredentials: true })
      }

      if (rsp.status == 200) {
          console.log('ok')
      }
      console.log('rsp', rsp)
    } catch(error) {
      console.log('about to throw an error', error)
      throw new Error(error.response.data)
    }
    console.log('UPDATE_CONTROL_GROUP', ctrlgrp)
    //state.user.name = name;
  },
  DELETE_CONTROL_GROUP(state, id) {
    console.log('DELETE_CONTROL_GROUP', id)
    //state.user.name = name;
  },

  initialize(state) {
    const host = 'api.hq.dev.nathejk.dk' // window.location.host
    console.log("connecting to websocket: ws://" + location.host + "/ws (" + state.timeout + "ms)")
    try {
      const protocol = location.protocol == 'https:' ? 'wss:' : 'ws:'
      state.ws = new WebSocket(protocol + '//' + location.host + '/ws');
    } catch(e) {
        console.log(e)
    }
      console.log(state)
    const that = this
    state.ws.addEventListener('open', function() {
        console.log("ws connected")
      state.status = 'open'
      state.timeout = 250 // reset reconnect timeout
      // subscribe to relevant views
      state.ws.send(JSON.stringify({View:'users'}))
      state.ws.send(JSON.stringify({View:'controlgroup'}))
      state.ws.send(JSON.stringify({View:'klan'}))
      state.ws.send(JSON.stringify({View:'patrulje'}))
      state.ws.send(JSON.stringify({View:'spejder'}))
      state.ws.send(JSON.stringify({View:'sos'}))
    });
      console.log('readystate', state.ws.readyState)
    if (state.ws.readyState == 1) {
        console.log("ws connected")
      state.status = 'open'
      state.timeout = 250 // reset reconnect timeout
      // subscribe to relevant views
      state.ws.send(JSON.stringify({View:'users'}))
      state.ws.send(JSON.stringify({View:'controlgroup'}))
      state.ws.send(JSON.stringify({View:'klan'}))
      state.ws.send(JSON.stringify({View:'patrulje'}))
      state.ws.send(JSON.stringify({View:'spejder'}))
      state.ws.send(JSON.stringify({View:'sos'}))
    }
    state.ws.addEventListener('close', function(e) {
        console.log("ws close", e)
      state.status = 'closed'
      state.timeout = Math.min(10000, state.timeout*2)
      setTimeout (function(){that.commit('dims/initialize')}, state.timeout)
    });
    state.ws.addEventListener('error', function(err) {
        console.log("ws error", err)
      state.status = 'errored'
    //  console.error('Socket encountered error: ', err, 'Closing socket');
      //state.ws.close();
    })
    state.ws.addEventListener('message', function(e) {
        var msg = JSON.parse(e.data);
        //console.log(msg)
        if (!state.list[msg.view]) {
            state.list[msg.view] = []
        }
        state.list[msg.view][msg.key] = msg.body
        if (msg.body == null) {
            delete state.list[msg.view][msg.key]
        }
        state.ids[msg.view] = Object.keys(state.list[msg.view])
        state.lastModify[msg.view] = Date.now()
        //console.log(state)
    });
  }
}

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}
