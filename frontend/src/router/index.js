import { createRouter, createWebHistory } from 'vue-router'
import Browser from '../views/Browser.vue'

const routes = [
  { path: '/', redirect: '/browser' },
  { path: '/browser', component: Browser, name: 'browser' },
  { path: '/browser/:bucket', component: Browser, name: 'bucket' },
  { path: '/browser/:bucket/:pathMatch(.*)*', component: Browser, name: 'folder' }
]

export default createRouter({
  history: createWebHistory(),
  routes
})
