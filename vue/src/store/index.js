import Vue from 'vue'
import Vuex from 'vuex'
/*
import {
  SOCKET_ONOPEN,
  SOCKET_ONCLOSE,
  SOCKET_ONERROR,
  SOCKET_ONMESSAGE,
  SOCKET_RECONNECT,
  SOCKET_RECONNECT_ERROR
} from './websocket-mutations'
*/
import app from './app'
import osm from './osm'
import user from './user'
import dims from './dims'

Vue.use(Vuex)

export const store = new Vuex.Store({
  modules: {
    app,
    osm,
    user,
    dims,
  },
  strict: false,
  plugins: [],
  state: {
    socket: {
      isConnected: false,
      message: '',
      reconnectError: false,
    }
  },
  mutations: {
      /*
    SOCKET_ONOPEN (state, event)  {
      Vue.prototype.$socket = event.currentTarget
      state.socket.isConnected = true
    },
    SOCKET_ONCLOSE (state)  {
      state.socket.isConnected = false
    },
    SOCKET_ONERROR (state, event)  {
      console.error(state, event)
    },
    // default handler called for all methods
    SOCKET_ONMESSAGE (state, message)  {
      state.socket.message = message
      console.log('SOCKET_ONMESSAGE', message)
    },
    // mutations for reconnect methods
    SOCKET_RECONNECT(state, count) {
      console.info("SOCKET_RECONNECT", count)
    },
    SOCKET_RECONNECT_ERROR(state) {
      state.socket.reconnectError = true;
    },

    [SOCKET_ONOPEN](state)  {
      state.socket.isConnected = true
    },
    [SOCKET_ONCLOSE](state)  {
      state.socket.isConnected = false
    },
    [SOCKET_ONERROR](state, event)  {
      console.error(state, event)
    },
    // default handler called for all methods
    [SOCKET_ONMESSAGE](state, message)  {
      state.socket.message = message
    },
    // mutations for reconnect methods
    [SOCKET_RECONNECT](state, count) {
      console.info(state, count)
    },
    [SOCKET_RECONNECT_ERROR](state) {
      state.socket.reconnectError = true;
    }
    */
  },
})

export const initializeStores = function() {
  // Initialize stores
  //this.$store.commit('osm/initialize')
  this.$store.commit('dims/initialize')
}
