import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import DeviceDetailView from '../views/DeviceDetailView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
    },
    {
      path: '/device/:imei',
      name: 'device-detail',
      component: DeviceDetailView,
    },
  ],
})

export default router
