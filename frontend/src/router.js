import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/setup', component: () => import('./pages/Setup.vue') },
  { path: '/dashboard', component: () => import('./pages/Dashboard.vue'), meta: { auth: true } },
  { path: '/history', component: () => import('./pages/History.vue'), meta: { auth: true } },
  { path: '/exercise/:id', component: () => import('./pages/Exercise.vue'), meta: { auth: true } },
  { path: '/wrong', component: () => import('./pages/Wrong.vue'), meta: { auth: true } },
  { path: '/collects', component: () => import('./pages/Collects.vue'), meta: { auth: true } },
  { path: '/review', component: () => import('./pages/Review.vue'), meta: { auth: true } },
  { path: '/question/:id', component: () => import('./pages/Question.vue'), meta: { auth: true } },
  { path: '/account', component: () => import('./pages/Account.vue'), meta: { auth: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
