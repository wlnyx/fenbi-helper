<template>
  <div>
    <h1 class="page-title">练习历史</h1>
    <el-skeleton :loading="loading" animated :rows="8">
      <template #default>
        <el-empty v-if="!history.length" description="暂无练习记录" />
        <div v-for="h in history" :key="h.id" class="bento-card clickable his-item"
             @click="$router.push('/exercise/' + h.id)">
          <span class="his-name">{{ h.sheet.name || '未命名练习' }}</span>
          <el-tag size="small" effect="plain" round>做了{{ h.answerCount }}题</el-tag>
          <el-tag v-if="h.elapsedTime" size="small" effect="plain" round>{{ fmtTime(h.elapsedTime) }}</el-tag>
          <el-tag v-if="h.correctRate > 0" :type="h.correctRate >= 60 ? 'success' : 'warning'" size="small" round>
            {{ h.correctRate.toFixed(1) }}%
          </el-tag>
          <span class="his-meta">{{ h.client }}</span>
          <el-icon class="his-arrow"><ArrowRight /></el-icon>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getHistory } from '../api'

const loading = ref(true)
const history = ref([])

const fmtTime = (sec) => {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return m > 0 ? `${m}分${s}秒` : `${s}秒`
}

onMounted(async () => {
  try {
    const res = await getHistory()
    if (res.code === 200) history.value = res.history
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.his-item {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 12px;
  padding: 14px 20px;
}
.his-name { flex: 1; font-size: 14px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.his-meta { font-size: 12px; color: var(--el-text-color-secondary); white-space: nowrap; }
.his-arrow { color: var(--el-text-color-secondary); }
</style>
