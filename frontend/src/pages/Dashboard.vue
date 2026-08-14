<template>
  <div>
    <h1 class="page-title">复盘工作台</h1>
    <el-skeleton :loading="loading" animated :rows="8">
      <template #default>
        <!-- KPI 统计行 -->
        <div style="display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:20px;">
          <div v-for="k in kpis" :key="k.label" class="bento-card" style="text-align:center;padding:18px 12px;">
            <div class="kpi-num" :style="{ color: k.color }">{{ k.value }}</div>
            <div class="kpi-label">{{ k.label }}</div>
          </div>
        </div>

        <div style="display:grid;grid-template-columns:2fr 1fr 1fr;gap:16px;margin-bottom:20px;">
          <!-- 今日复习 2x2 -->
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><Calendar /></el-icon> 今日复习</div>
            <el-empty v-if="!data.queue.length" description="暂无待复习题目" :image-size="60">
              <el-button type="primary" size="small" @click="$router.push('/wrong')">去错题本</el-button>
            </el-empty>
            <div v-else>
              <div v-for="q in data.queue.slice(0, 6)" :key="q.questionId"
                   class="queue-item" @click="$router.push('/question/' + q.questionId)">
                <span class="queue-title">{{ q.title }}</span>
                <el-tag v-if="q.errorCategory" size="small" effect="plain" round>{{ q.errorCategory }}</el-tag>
                <el-tag v-if="q.reviewState === 'flagged'" type="danger" size="small" round>重点</el-tag>
                <el-tag v-if="q.consecutiveOK > 0" type="success" size="small" round>连对{{ q.consecutiveOK }}</el-tag>
              </div>
              <el-button link type="primary" size="small" @click="$router.push('/review')">查看全部 →</el-button>
            </div>
          </div>

          <!-- 错误类型分布 1x1 -->
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><DataAnalysis /></el-icon> 错误类型分布</div>
            <el-empty v-if="!totalCat" description="还没有归类记录" :image-size="50" />
            <div v-else v-for="c in categories" :key="c" class="cat-row">
              <span class="cat-name">{{ c }}</span>
              <div class="cat-bar">
                <div class="cat-bar-inner" :style="{ width: pctCat(catCount[c] || 0) + '%', background: catColor(c) }"></div>
              </div>
              <span class="cat-num">{{ catCount[c] || 0 }}</span>
            </div>
          </div>

          <!-- 难度分布 1x1 -->
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><TrendCharts /></el-icon> 复习状态</div>
            <el-empty v-if="!stateTotal" description="暂无数据" :image-size="50" />
            <div v-else v-for="s in states" :key="s.label" class="cat-row">
              <span class="cat-name">{{ s.label }}</span>
              <div class="cat-bar">
                <div class="cat-bar-inner" :style="{ width: pctState(s.value) + '%', background: s.color }"></div>
              </div>
              <span class="cat-num">{{ s.value }}</span>
            </div>
          </div>
        </div>

        <!-- 近期练习 全宽 -->
        <div class="bento-card">
          <div class="bento-card-title"><el-icon><Document /></el-icon> 近期练习</div>
          <el-empty v-if="!data.recentHistory.length" description="暂无练习记录" :image-size="60" />
          <div style="display:grid;grid-template-columns:repeat(2,1fr);gap:0 24px;">
            <div v-for="h in data.recentHistory" :key="h.id" class="queue-item" @click="$router.push('/exercise/' + h.id)">
              <span class="queue-title">{{ h.name }}</span>
              <span style="color:var(--el-text-color-secondary);font-size:12px;">{{ h.answerCount }}题</span>
              <span style="color:var(--el-color-success);font-size:12px;">{{ h.correctRate.toFixed(1) }}%</span>
            </div>
          </div>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getDashboard } from '../api'

const loading = ref(true)
const data = ref({ queue: [], recentHistory: [], categoryCount: {} })
const categories = ['找数瞎', '列式乱', '计算蠢', '掉坑快', '死磕傻', '读不懂']
const catColors = ['#0D9488', '#6D28D9', '#EA580C', '#DC2626', '#92400E', '#0891B2']

const kpis = computed(() => [
  { label: '待复习', value: data.value.totalReview ?? 0, color: '#1D1D1F' },
  { label: '重点攻坚', value: data.value.flagged ?? 0, color: '#EA580C' },
  { label: '已掌握', value: data.value.mastered ?? 0, color: '#0D9488' },
  { label: '未复盘', value: data.value.unreviewed ?? 0, color: '#B45309' }
])
const catCount = computed(() => data.value.categoryCount || {})
const totalCat = computed(() => categories.reduce((a, c) => a + (catCount.value[c] || 0), 0))
const stateTotal = computed(() => (data.value.totalReview ?? 0) + (data.value.flagged ?? 0) + (data.value.mastered ?? 0) + (data.value.unreviewed ?? 0))
const states = computed(() => [
  { label: '待复习', value: data.value.totalReview ?? 0, color: '#86868B' },
  { label: '重点', value: data.value.flagged ?? 0, color: '#EA580C' },
  { label: '已掌握', value: data.value.mastered ?? 0, color: '#0D9488' },
  { label: '未复盘', value: data.value.unreviewed ?? 0, color: '#B45309' }
])

const pctCat = (n) => (totalCat.value ? Math.round(n / totalCat.value * 100) : 0)
const pctState = (n) => (stateTotal.value ? Math.round(n / stateTotal.value * 100) : 0)
const catColor = (c) => catColors[categories.indexOf(c)]

onMounted(async () => {
  try {
    const res = await getDashboard()
    if (res.code === 200) data.value = res.data
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.queue-item {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 6px;
  border-bottom: 1px dashed #F0F0F4;
  cursor: pointer;
  border-radius: 6px;
  transition: background 150ms ease;
}
.queue-item:hover { background: #FAFAFC; }
.queue-item:last-child { border-bottom: none; }
.queue-title { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cat-row { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; font-size: 13px; }
.cat-bar { height: 10px; border-radius: 5px; background: #F0F0F4; overflow: hidden; flex: 1; }
.cat-bar-inner { height: 100%; border-radius: 5px; transition: width 300ms ease; }
.cat-name { width: 48px; color: var(--el-text-color-regular); }
.cat-num { width: 28px; text-align: right; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
</style>
