import axios from 'axios';
import Vue from 'vue';

function cacheKey(position) {
  return position.latitude + ':' + position.longitude
}

// initial state
const state = {
  cache: {},
}

const getters = {}

// actions
const actions = {
  async reverse({ commit, state }, position) {
    if (state.cache[cacheKey(position)]) {
      return Object.assign({}, state.cache[cacheKey(position)])
    }
    // https://nominatim.openstreetmap.org/reverse?format=json&lat=55.661904&lon=12.560676
    const params = {
      format: 'json',
      lat: position.latitude,
      lon: position.longitude,
    }
    try {
      const response = await axios.get('https://nominatim.openstreetmap.org/reverse', { params });
      commit('cache', {
        position,
        result: response.data,
      })
      return response.data
    }
    catch (error) {
      console.log(error)
      return null
    }
  }
}

// mutations
const mutations = {
  cache(state, { position, result }) {
    Vue.set(state.cache, cacheKey(position), result)
    Vue.localStorage.set('nominatimCache', state.cache)
  },

  initialize(state) {
    const nominatimCache = Vue.localStorage.get('nominatimCache', {})
    state.cache = nominatimCache
  }
}

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}
