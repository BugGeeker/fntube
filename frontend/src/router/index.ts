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
    path: '/movie-search',
    name: 'MovieSearch',
    component: () => import('@/views/MovieSearchView.vue'),
  },
  {
    path: '/media-inconsistent',
    name: 'MediaInconsistent',
    component: () => import('@/views/MediaInconsistentView.vue'),
  },
  {
    path: '/system-test',
    name: 'SystemTest',
    component: () => import('@/views/SystemTestView.vue'),
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/SettingsView.vue'),
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
  {
    path: '/guangya-config',
    name: 'GuangyaConfig',
    component: () => import('@/views/GuangyaConfigView.vue'),
  },
  {
    path: '/smartstrm-config',
    name: 'SmartStrmConfig',
    component: () => import('@/views/SmartStrmConfigView.vue'),
  },
  {
    path: '/storage-config',
    name: 'StorageConfig',
    component: () => import('@/views/StorageConfigView.vue'),
  },
  {
    path: '/guangya-cloud-jobs',
    name: 'GuangyaCloudJobs',
    component: () => import('@/views/GuangyaCloudJobsView.vue'),
  },
  {
    path: '/scrape-log',
    name: 'ScrapeLog',
    component: () => import('@/views/ScrapeLogView.vue'),
  },
  {
    path: '/scrape-task',
    name: 'ScrapeTask',
    component: () => import('@/views/ScrapeTaskView.vue'),
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
