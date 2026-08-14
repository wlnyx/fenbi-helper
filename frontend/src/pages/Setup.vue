<template>
  <div class="login-wrap">
    <div class="bento-card login-card">
      <div class="logo-badge"><el-icon :size="26"><Reading /></el-icon></div>
      <h1 class="login-title">粉笔刷题助手</h1>
      <p class="login-sub">扫码登录 · 六步复盘工作台</p>
      <div class="qr-box" :class="{ ready: qrReady }">
        <el-icon v-if="!qrReady" class="is-loading" :size="28" style="color:var(--el-color-primary);"><Loading /></el-icon>
        <img v-else :src="qrSrc" alt="二维码" />
      </div>
      <p class="qr-msg">{{ msg }}</p>
      <el-button v-if="qrFailed" type="primary" round @click="start">二维码已失效，点击重新生成</el-button>
      <p class="qr-tip">打开粉笔 App → 右上角「扫一扫」</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { qrStart, qrStatus, qrFinish, sessionApply } from '../api'
import http from '../api'

const router = useRouter()
const qrSrc = ref('')
const qrReady = ref(false)
const qrFailed = ref(false)
const msg = ref('正在连接粉笔...')
let timer = null

onMounted(async () => {
  // 已登录则直接进入工作台
  try {
    const res = await http.get('/session/info')
    if (res.data.code === 200 && res.data.userId) {
      router.push('/dashboard')
      return
    }
  } catch { /* 未登录 */ }
  start()
})

async function start() {
  qrFailed.value = false
  qrReady.value = false
  msg.value = '正在连接粉笔...'
  try {
    const res = await qrStart()
    if (res.data.code !== 200) {
      msg.value = '获取二维码失败：' + (res.data.msg || '')
      qrFailed.value = true
      return
    }
    qrSrc.value = '/api/qr.png?t=' + Date.now()
    qrReady.value = true
    msg.value = '请用粉笔App扫描二维码'
    poll()
  } catch {
    msg.value = '请求失败，请重试'
    qrFailed.value = true
  }
}

async function poll() {
  clearInterval(timer)
  timer = setInterval(async () => {
    try {
      const res = await qrStatus()
      const s = res.data
      if (s.state === 'waiting') {
        msg.value = '请用粉笔App扫描二维码'
      } else if (s.state === 'scanned') {
        msg.value = '已扫码，请在手机上确认'
      } else if (s.state === 'authorized') {
        clearInterval(timer)
        msg.value = '登录成功，正在写入 Cookie...'
        const f = await qrFinish()
        if (f.data.code === 200) {
          const a = await sessionApply()
          if (a.data.code === 200) router.push(a.data.redirectPath || '/dashboard')
        } else {
          msg.value = '登录失败：' + (f.data.msg || '')
          qrFailed.value = true
        }
      } else if (s.state === 'expired' || s.state === 'canceled') {
        clearInterval(timer)
        msg.value = s.msg + '，正在重新生成...'
        start()
      }
    } catch { /* 网络抖动忽略 */ }
  }, 2000)
}

onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: var(--page-bg);
}
.login-card { width: 360px; text-align: center; padding: 36px 32px; }
.logo-badge {
  width: 56px; height: 56px;
  border-radius: 14px;
  background: var(--el-color-primary);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 14px;
}
.login-title { font-size: 20px; font-weight: 700; margin-bottom: 4px; }
.login-sub { font-size: 13px; color: var(--el-text-color-secondary); margin-bottom: 22px; }
.qr-box {
  width: 240px; height: 240px;
  margin: 0 auto 16px;
  border: 2px dashed #A3DCD6;
  border-radius: 16px;
  display: flex; align-items: center; justify-content: center;
  background: #FCFCFD;
}
.qr-box.ready { border-style: solid; border-color: #C2E8E4; }
.qr-box img { width: 220px; height: 220px; }
.qr-msg { font-size: 14px; color: var(--el-text-color-regular); margin-bottom: 14px; min-height: 20px; }
.qr-tip { margin-top: 18px; font-size: 12px; color: var(--el-text-color-secondary); }
</style>
