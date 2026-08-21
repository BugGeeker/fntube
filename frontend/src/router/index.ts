import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/HomeView.vue'),
  },
  {
    path: '/media',
    name: 'Media',
    component: () => import('@/views/MediaView.vue'),
  },
  {
    path: '/media/:id',
    name: 'MediaItems',
    component: () => import('@/views/MediaItemsView.vue'),
  },
  {
    path: '/config',
    name: 'Config',
    component: () => import('@/views/ConfigView.vue'),
  },
  {
    path: '/metatube-config',
    name: 'MetaTubeConfig',
    component: () => import('@/views/MetaTubeConfigView.vue'),
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
