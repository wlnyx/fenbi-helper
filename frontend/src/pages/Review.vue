<template>
  <div>
    <h1 class="page-title">复习队列</h1>
    <el-skeleton :loading="loading" animated :rows="6">
      <template #default>
        <template v-if="!flagged.length && !pending.length && !mastered.length">
          <el-empty description="暂无复习任务，先去错题本给题目归类吧">
            <el-button type="primary" @click="$router.push('/wrong')">去错题本</el-button>
          </el-empty>
        </template>

        <section v-if="flagged.length" class="section">
          <h3 class="section-title"><el-icon color="#EA580C"><StarFilled /></el-icon> 重点攻坚 <el-tag type="danger" size="small" round>{{ flagged.length }}</el-tag></h3>
          <div v-for="q in flagged" :key="q.questionId" class="bento-card clickable queue-item">
            <span class="q-title" @click="$router.push('/question/' + q.questionId)">{{ q.title }}</span>
            <el-tag v-if="q.errorCategory" size="small" effect="plain" round>{{ q.errorCategory }}</el-tag>
            <el-tag v-if="q.consecutiveOK > 0" type="success" size="small" round>连对{{ q.consecutiveOK }}</el-tag>
            <div class="actions" @click.stop>
              <el-button size="small" type="success" plain @click="redo(q, 'correct')">重做做对</el-button>
              <el-button size="small" type="danger" plain @click="redo(q, 'wrong')">重做做错</el-button>
              <el-button size="small" @click="setState(q, 'mastered')">标记已掌握</el-button>
              <el-button size="small" text @click="archive(q)">归档</el-button>
            </div>
          </div>
        </section>

        <section v-if="pending.length" class="section">
          <h3 class="section-title"><el-icon><List /></el-icon> 待复习 <el-tag size="small" round>{{ pending.length }}</el-tag></h3>
          <div v-for="q in pending" :key="q.questionId" class="bento-card clickable queue-item">
            <span class="q-title" @click="$router.push('/question/' + q.questionId)">{{ q.title }}</span>
            <el-tag v-if="q.errorCategory" size="small" effect="plain" round>{{ q.errorCategory }}</el-tag>
            <el-tag v-if="q.consecutiveOK > 0" type="success" size="small" round>连对{{ q.consecutiveOK }}</el-tag>
            <div class="actions" @click.stop>
              <el-button size="small" type="success" plain @click="redo(q, 'correct')">重做做对</el-button>
              <el-button size="small" type="danger" plain @click="redo(q, 'wrong')">重做做错</el-button>
              <el-button size="small" type="warning" plain @click="setState(q, 'flagged')">标为重点</el-button>
              <el-button size="small" @click="setState(q, 'mastered')">标记已掌握</el-button>
            </div>
          </div>
        </section>

        <section v-if="mastered.length" class="section">
          <h3 class="section-title"><el-icon color="#0D9488"><CircleCheck /></el-icon> 已掌握 <el-tag type="success" size="small" round>{{ mastered.length }}</el-tag></h3>
          <div v-for="q in mastered" :key="q.questionId" class="bento-card clickable queue-item">
            <span class="q-title" @click="$router.push('/question/' + q.questionId)">{{ q.title }}</span>
            <el-tag v-if="q.errorCategory" size="small" effect="plain" round>{{ q.errorCategory }}</el-tag>
            <div class="actions" @click.stop>
              <el-button size="small" text @click="archive(q)">归档</el-button>
            </div>
          </div>
        </section>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'

const loading = ref(true)
const flagged = ref([])
const pending = ref([])
const mastered = ref([])

async function load() {
  loading.value = true
  try {
    const res = await http.get('/review/queue')
    if (res.data.code === 200) {
      flagged.value = res.data.flagged
      pending.value = res.data.pending
      mastered.value = res.data.mastered
    }
  } finally {
    loading.value = false
  }
}

async function redo(q, result) {
  const res = await http.post('/review/update', { questionId: String(q.questionId), field: 'redo', value: result })
  if (res.data.code === 200) {
    ElMessage.success(result === 'correct' ? '打卡成功' : '已记录')
    load()
  }
}

async function setState(q, state) {
  const res = await http.post('/review/update', { questionId: String(q.questionId), field: 'state', value: state })
  if (res.data.code === 200) { ElMessage.success('已更新'); load() }
}

async function archive(q) {
  const res = await http.post('/review/update', { questionId: String(q.questionId), field: 'archive', value: '1' })
  if (res.data.code === 200) { ElMessage.success('已归档'); load() }
}

onMounted(load)
</script>

<style scoped>
.section { margin-bottom: 24px; }
.section-title { font-size: 15px; font-weight: 600; margin-bottom: 12px; display: flex; align-items: center; gap: 8px; }
.queue-item { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; padding: 12px 20px; }
.q-title { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.actions { display: flex; gap: 6px; flex-shrink: 0; }
</style>
