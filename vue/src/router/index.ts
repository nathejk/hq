import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/year',
      name: 'year',
      component: () => import('@/views/YearView.vue')
    },
    {
      path: '/klan',
      name: 'klaner',
      component: () => import('@/views/KlanListView.vue')
    },
    {
      path: '/patrulje',
      name: 'patruljer',
      component: () => import('@/views/PatruljeListView.vue')
    },
    {
      path: '/patrulje/:teamId',
      name: 'patrulje',
      component: () => import('@/views/PatruljeView.vue'),
      props: true
    },
    {
      path: '/badut',
      name: 'badutter',
      component: () => import('@/views/BadutListView.vue')
    },
    {
      path: '/poster',
      name: 'poster',
      component: () => import('@/views/PostList.vue')
    },
    {
      path: '/mail',
      name: 'mail',
      component: () => import('@/views/MailView.vue')
    },
    {
      path: '/mail/:page',
      component: () => import('@/views/MailView.vue')
    },
    {
      path: '/kort',
      name: 'kort',
      component: () => import('@/views/KortView.vue')
    },
    {
      path: '/about',
      name: 'about',
      // route level code-splitting
      // this generates a separate chunk (About.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: () => import('@/views/AboutView.vue')
    }
  ]
})

export default router
