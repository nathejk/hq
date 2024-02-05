import Vue from 'vue'
import Router from 'vue-router'

const routes = [
  { path: '/',              component: () => import('@/views/Frontpage.vue'), name: 'home' },
  { path: '/kort',          component: () => import('@/views/Kort.vue'), meta: {footer:false}},
  { path: '/poster',        component: () => import('@/views/Poster.vue') },
  { path: '/organisation',  component: () => import('@/views/Organisation.vue') },
  { path: '/hej',           component: () => import('@/views/Hej.vue') },
  { path: '/ude',           component: () => import('@/views/Udgået.vue') },
  { path: '/sos',           component: () => import('@/views/Sos/List.vue') },
  { path: '/sos/new',       component: () => import('@/views/Sos/View.vue'), name: 'new-sos' },
  { path: '/sos/:id',       component: () => import('@/views/Sos/View.vue'), name: 'view-sos' },

  { path: '/patruljer',     component: () => import('@/views/Patruljer.vue'), name: 'patruljer' },
  { path: '/patrulje/:id',  component: () => import('@/views/Patrulje.vue'), name: 'patrulje' },
  { path: '/lok',           component: () => import('@/views/Lok.vue'), name: 'loks' },
//  { path: '/klan',          component: () => import('@/views/List.vue'), name: 'klan-list', props: { team: "klan" } },
  { path: '/klan/:id',      component: () => import('@/views/Team.vue'), name: 'klan-view', props: { team: "klan" } },
  { path: '/senior/:id',    component: () => import('@/views/Team.vue') },
  { path: '/years',         component: () => import('@/views/Year.vue'), name:'years' },
  // Notfound
  { path: '*',  component: () => import('@/views/NotFound.vue') },
]

Vue.use(Router)

const router = new Router({
  mode: 'history',
  routes,
})

router.setPermissions = function (userPermissions) {
  this.userPermissions = userPermissions
}

router.beforeEach((to, from, next) => {
  const route = router.options.routes.find(r => r.path === to.path)
  const pathPermissions = route && route.meta && route.meta.permissions ? route.meta.permissions : []
  const userPermissions = router.userPermissions
  if (pathPermissions.length == 0) return next()
  if (!userPermissions) return next()
  if (userPermissions.some(el => pathPermissions.includes(el))) return next()
  next({ path: '/' })
})

/*
// Make sure navbar collapses when changing route.
router.beforeEach((to, from, next) => {
  $('#navbarNav').collapse('hide')
  next()
})
*/
/*
// Make sure navbar collapses when changing route.
router.afterEach((to, from) => {
  //window.Vue.$store.commit('app/refreshCurrentUrl', to.fullPath)
  window.scrollTo({top: 0})
})
*/
export default router
