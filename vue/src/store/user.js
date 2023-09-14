import axios from 'axios'
import Vue from 'vue'

// initial state
const state = {
  permissions: [],
  expires: null,
  claims: {},
  profile: {},
  profiles: {},
  users: [],
  invitations: {},
  customer: null,
  isLoggedIn: false
}

// getters
const getters = {
  userId() {
    return state.claims.user_id
  },

  profile: (state) => (userId) => {
    if (!userId) return null
    if (!state.profiles[userId]) {
      Vue.set(state.profiles, userId, null)
    }
    return state.profiles[userId]
  },

  users: (state) => {
    return Object.values(state.users)
      .filter(item => item)
  },

  invitations: (state) => {
    return Object.values(state.invitations)
      .filter(item => item)
  }
}

// actions
const actions = {
  async loadUser({ commit }) {
    const response = await axios.get(window.envConfig.AUTH_BASEURL + '/token', { withCredentials: true });
      console.log(response)
      if (response.statusCode != 200) {
          //location.href = window.envConfig.AUTH_BASEURL
      }
    commit('setPermissions', response.data.permissions)
    commit('setExpires', response.data.expires)
    commit('setClaims', response.data.claims)
    commit('setProfile', response.data.profile)
    commit('setIsLoggedIn', true)
    return response.data
  },

  async loadUsers({ commit }) {
    try {
      const response = await axios.get(window.envConfig.API_BASEURL + '/api/users', { withCredentials: true });
      commit('setUsers', response.data.users)
      commit('setInvitations', response.data.invitations)
    }
    catch (error) {
      console.log(error)
    }
  },

  async loadProfile({ commit }, userId) {
    try {
      const response = await axios.get(window.envConfig.API_BASEURL + '/api/user/' + userId + '/profile', { withCredentials: true });
      commit('setProfileData', response.data.profile)
      return response.data.profile
    }
    catch (error) {
      console.log(error)
      return null
    }
  },

  async saveProfile({ commit }, { userId, profileData }) {
    try {
      const response = await axios.put(window.envConfig.API_BASEURL + '/api/user/' + userId, { profileData }, { withCredentials: true });
      commit('setProfile', Object.assign({}, profileData))
      commit('setProfileData', Object.assign({userId}, profileData))
      console.log(response)
      return true
    }
    catch (error) {
      console.log(error)
      return false
    }
  },
  async updatePassword({ userId, password, newPassword }) {
    return axios.post(window.envConfig.API_BASEURL + '/api/user/' + userId + '/password', { password, newPassword }, { withCredentials: true });
  },
  autoLogout(callback) {
    const timeout = state.expires - Math.floor(Date.now() / 1000)
    // Auto logout after session expires.
    setTimeout(function () {
      if (state.expires <= Math.floor(Date.now() / 1000)) {
        console.log("Access token expired")
        callback()
      }
    }, timeout * 1000)

    // Auto logout after session expires through polling.
    // The above handler might not be triggered, due to browser sleep etc.
    //
    // @todo will this polling method suffice? Can we skip the above timeout
    //       handler?
    setInterval(function () {
      // console.log("Checking: " + Math.floor(Date.now() / 1000))
      if (state.expires <= Math.floor(Date.now() / 1000)) {
        console.log("Access token expired")
        callback()
      }
    }, 5 * 1000)
  },

  async uploadAvatar({ userId, file }) {
    const data = new FormData();
    data.append('avatar', file);
    const config = {
      withCredentials: true,
      onUploadProgress: function (progressEvent) {
        file.progress.current = Math.round((progressEvent.loaded * 100) / progressEvent.total)
      }
    }
    try {
      const response = await axios.post(window.envConfig.API_BASEURL + '/api/user/' + userId + '/avatar', data, config);
      file.progress.uploading = false
      file.progress.error = false
      file.progress.id = response.data
      return response.data
    }
    catch (error) {
      file.progress.uploading = false
      file.progress.error = error.toString()
      console.log("error", error)
      return null
    }
  },
  async invite({ commit }, { name, email }) {
    const response = await axios.post(window.envConfig.API_BASEURL + '/api/users/invite', { name, email }, { withCredentials: true });
    commit('setInvitations', [
      {
        invitationId: response.data,
        name,
        email,
      }
    ])
  },
  async revokeAccess({ commit }, userId) {
    await axios.post(window.envConfig.API_BASEURL + '/api/user/' + userId + '/revoke', {}, { withCredentials: true });
    commit('removeUser', userId)
  },
  async recallInvitation({ commit }, invitationId) {
    await axios.post(window.envConfig.API_BASEURL + '/api/invitation/' + invitationId + '/recall', {}, { withCredentials: true });
    commit('removeInvitation', invitationId)
  }
}

// mutations
const mutations = {
  setPermissions(state, permissions) {
    state.permissions = permissions
  },

  setExpires(state, expires) {
    state.expires = expires
  },

  setTimeoutHandler(state, timeoutHandler) {
    state.timeoutHandler = timeoutHandler
  },

  setCustomer(state, customer) {
    state.customer = customer
  },

  setClaims(state, claims) {
    state.claims = claims
  },

  setProfile(state, profile) {
    if (!profile) {
      return
    }

    profile.firstname = profile.name ? profile.name.replace(/ .*/, '') : ''
    state.profile = Object.assign({}, state.profile, profile)
  },

  setProfileData(state, profile) {
    const current = state.profiles[profile.userId] ? state.profiles[profile.userId] : {}
    Vue.set(state.profiles, profile.userId, Object.assign({}, current, profile))
  },

  setIsLoggedIn(state, newState) {
    state.isLoggedIn = newState
  },

  setUsers(state, users) {
    for (let user of users) {
      Vue.set(state.users, user.userId, user)
    }
  },

  removeUser(state, userId) {
    Vue.delete(state.users, userId)
  },

  setInvitations(state, invitations) {
    for (let invitation of invitations) {
      Vue.set(state.invitations, invitation.invitationId, invitation)
    }
  },

  removeInvitation(state, invitationId) {
    Vue.delete(state.invitations, invitationId)
  },
}

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations
}
