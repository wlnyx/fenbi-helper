import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 60000 })

// 简单内存缓存（GET 数据 5 分钟；题目 1 小时）
const cacheMap = new Map()

function cachedGet(key, fn, ttl = 300000) {
  const hit = cacheMap.get(key)
  if (hit && Date.now() - hit.time < ttl) {
    return Promise.resolve(hit.data)
  }
  return fn().then(data => {
    cacheMap.set(key, { data, time: Date.now() })
    return data
  })
}

export function forceRefresh(prefix) {
  for (const key of [...cacheMap.keys()]) {
    if (key.startsWith(prefix)) cacheMap.delete(key)
  }
}

export const qrStart = () => http.post('/qr/start')
export const qrStatus = () => http.get('/qr/status')
export const qrFinish = () => http.post('/qr/finish')
export const sessionApply = () => http.post('/session/apply')

export const getDashboard = (force) => cachedGet('dashboard',
  () => http.get('/dashboard').then(r => r.data), 300000)
export const getHistory = (force) => cachedGet('history',
  () => http.get('/history').then(r => r.data), 300000)
export const getWrong = (params) => cachedGet('wrong:' + (params?.module || '') + '|' + (params?.sub || '') + '|' + (params?.range || ''),
  () => http.get('/wrong', { params }).then(r => r.data), 300000)
export const getCollects = (params) => cachedGet('collects:' + (params?.module || '') + '|' + (params?.sub || '') + '|' + (params?.range || ''),
  () => http.get('/collects', { params }).then(r => r.data), 300000)
export const getQuestion = (id) => cachedGet('question:' + id,
  () => http.get(`/question/${id}`).then(r => r.data), 3600000)
export const getExercise = (id) => http.get(`/exercise/${id}`).then(r => r.data)

export const reviewUpdate = (payload) => http.post('/review/update', payload).then(r => {
  if (r.data.code === 200) {
    for (const p of ['wrong:', 'collects:', 'dashboard', 'history', 'review:']) forceRefresh(p)
  }
  return r.data
})
export const reviewNote4 = (payload) => http.post('/review/note4', payload).then(r => r.data)
export const reviewSummary = (payload) => http.post('/review/summary', payload).then(r => r.data)

export default http
