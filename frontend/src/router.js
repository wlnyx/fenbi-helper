import { createRouter, createWebHistory } from 'vue-router'
import http from './api'

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

// 路由守卫：未登录时访问受保护页面跳转登录页（服务端校验，HttpOnly cookie 前端不可读）
router.beforeEach(async (to) => {
  if (to.meta.auth) {
    try {
      const res = await http.get('/session/info')
      if (res.data.code !== 200 || !res.data.userId) {
        return { path: '/setup', query: { redirect: to.fullPath } }
      }
    } catch {
      return { path: '/setup', query: { redirect: to.fullPath } }
    }
  }
  return true
})

export default router
