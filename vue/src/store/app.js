const state = {
  show: false,
};

const getters = {}

const actions = {
  start({ commit }) {
    commit('setState', true)
  },
  async finish({ commit }, callback = null) {
    if (!state.show) {
      if (callback) callback()
    }
    else {
      commit('setState', false)
      setTimeout(function () {
        if (callback) callback()
      }, 1000)
    }
  },
  async login({ commit }) {
    const loginUrl = window.envConfig.AUTH_BASEURL + "/customer?goto=" + window.location
    await actions.finish({ commit }, function () {
      window.location = loginUrl
    })
  },
  async logout({ commit }) {
    const logoutUrl = window.envConfig.AUTH_BASEURL + "/logout?goto=" + window.location.protocol + "//" + window.location.host
    await actions.finish({ commit }, function () {
      window.location = logoutUrl
    })
  }
};

const mutations = {
  setState(state, show) {
    state.show = show
  }
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}