<template>
  <div>
    <h1 class="page-title">账号设置</h1>
    <el-skeleton :loading="loading" animated :rows="4">
      <template #default>
        <div class="bento-card" style="max-width:560px;">
          <div class="bento-card-title"><el-icon><User /></el-icon> 登录信息</div>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="账号 ID">{{ info.userId || '-' }}</el-descriptions-item>
            <el-descriptions-item label="设备标识">
              <span style="font-size:12px;color:var(--el-text-color-secondary);">{{ info.deviceId || '-' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="登录时间">
              {{ info.loginTime ? new Date(info.loginTime).toLocaleString('zh-CN') : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="会话 Cookie">{{ info.cookieCount ?? 0 }} 个</el-descriptions-item>
          </el-descriptions>

          <div style="margin-top:20px;display:flex;gap:12px;">
            <el-button type="primary" @click="reLogin">重新扫码登录</el-button>
            <el-button type="danger" plain @click="logout">退出登录</el-button>
          </div>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const router = useRouter()
const loading = ref(true)
const info = ref({})

onMounted(async () => {
  try {
    const res = await http.get('/session/info')
    if (res.data.code === 200) info.value = res.data
  } finally {
    loading.value = false
  }
})

function reLogin() {
  router.push('/setup')
}

async function logout() {
  try {
    await ElMessageBox.confirm('退出后将清除本地登录凭据，需要重新扫码登录。', '确认退出', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch { return }
  const res = await http.post('/session/logout')
  if (res.data.code === 200) {
    ElMessage.success('已退出登录')
    router.push('/setup')
  }
}
</script>
