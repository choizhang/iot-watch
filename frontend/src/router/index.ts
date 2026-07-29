import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import MoreView from '../views/MoreView.vue'
import HealthView from '../views/HealthView.vue'
import PhonebookView from '../views/PhonebookView.vue'
import IntervalView from '../views/IntervalView.vue'
import ReminderView from '../views/ReminderView.vue'
import ParamsView from '../views/ParamsView.vue'
import FindView from '../views/FindView.vue'
import HeartRateView from '../views/HeartRateView.vue'
import GeofenceView from '../views/GeofenceView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/more', name: 'more', component: MoreView },
    { path: '/health', name: 'health', component: HealthView },
    { path: '/phonebook', name: 'phonebook', component: PhonebookView },
    { path: '/interval', name: 'interval', component: IntervalView },
    { path: '/reminder', name: 'reminder', component: ReminderView },
    { path: '/params', name: 'params', component: ParamsView },
    { path: '/find', name: 'find', component: FindView },
    { path: '/heart-rate', name: 'heart-rate', component: HeartRateView },
    { path: '/geofence', name: 'geofence', component: GeofenceView },
  ]
})

export default router
