<template>
  <div>
    <h1 class="page-title">练习详情</h1>
    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <!-- 统计 -->
        <div style="display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:20px;">
          <div v-for="k in kpis" :key="k.label" class="bento-card" style="text-align:center;padding:18px 12px;">
            <div class="kpi-num">{{ k.value }}</div>
            <div class="kpi-label">{{ k.label }}</div>
          </div>
        </div>

        <!-- 可视化图表：耗时分布 + 难度分析 -->
        <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:20px;">
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><Timer /></el-icon> 单题耗时分布</div>
            <div ref="costChart" style="height:260px;"></div>
          </div>
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><TrendCharts /></el-icon> 题目难度 vs 正确率</div>
            <div ref="diffChart" style="height:260px;"></div>
          </div>
        </div>

        <!-- 错题清单 -->
        <div class="bento-card" style="margin-bottom:20px;">
          <div class="bento-card-title"><el-icon><CircleClose /></el-icon> 本次作答（{{ items.length }} 题）</div>
          <el-empty v-if="!items.length" description="暂无作答记录" :image-size="60" />
          <div v-for="it in items" :key="it.questionId" class="ex-item"
               :class="{ wrong: !it.correct }" @click="$router.push('/question/' + it.questionId)">
            <span class="ex-idx">{{ it.idx }}</span>
            <span class="ex-title">{{ it.title || ('题目 ' + it.questionId) }}</span>
            <el-tag v-if="it.cost" size="small" effect="plain" round>{{ it.cost }}s</el-tag>
            <el-tag size="small" :type="it.correct ? 'success' : 'danger'" round>
              {{ it.correct ? '✓' : '✗' }}
            </el-tag>
            <el-tag v-if="it.difficulty" size="small" effect="plain" round>难度{{ it.difficulty }}</el-tag>
          </div>
        </div>

        <!-- 宏观总结 ⑤ -->
        <div class="bento-card" style="max-width:720px;">
          <div class="bento-card-title"><el-icon><EditPen /></el-icon> 整套宏观总结</div>
          <el-form label-position="top">
            <el-form-item label="占比最高的错误类型">
              <el-input v-model="form.topErrorType" type="textarea" :rows="2" placeholder="如：计算蠢（截位失误、抄错数字）" />
            </el-form-item>
            <el-form-item label="耗时最长的材料以及原因">
              <el-input v-model="form.slowestMaterial" type="textarea" :rows="2" placeholder="如：第二篇资料（26分钟），原因：增长率比较题反复验算" />
            </el-form-item>
            <el-form-item label="本次最容易踩的陷阱">
              <el-input v-model="form.trap" type="textarea" :rows="2" placeholder="如：混淆「多几倍」与「是几倍」、同比环比看错" />
            </el-form-item>
            <el-form-item label="可执行做题规范（例：读题强制圈时间、主体、单位）">
              <el-input v-model="form.rules" type="textarea" :rows="3" placeholder="下次做题必须执行：…" />
            </el-form-item>
            <el-button type="primary" :loading="saving" @click="save">保存总结</el-button>
          </el-form>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import { getExercise, reviewSummary } from '../api'

const route = useRoute()
const loading = ref(true)
const saving = ref(false)
const report = ref({})
const items = ref([])
const form = ref({ topErrorType: '', slowestMaterial: '', trap: '', rules: '' })
const costChart = ref(null)
const diffChart = ref(null)
let costChartInst = null
let diffChartInst = null

const kpis = computed(() => {
  const r = report.value
  const rate = r.answerCount > 0 ? (r.correctCount / r.answerCount * 100).toFixed(1) : '-'
  return [
    { label: '做题数', value: r.answerCount ?? '-' },
    { label: '正确数', value: r.correctCount ?? '-' },
    { label: '正确率', value: r.answerCount ? rate + '%' : '-' },
    { label: '总耗时', value: r.elapsedTime != null ? r.elapsedTime + 's' : '-' }
  ]
})

function renderCharts() {
  const data = items.value
  if (!data.length) return

  // 图1：单题耗时柱状（对绿错红 + 80分位线 + 累计耗时）
  const costs = data.map(d => d.cost || 0)
  const sorted = [...costs].sort((a, b) => a - b)
  const p80 = sorted[Math.floor(sorted.length * 0.8)] || 0
  const cumulative = []
  let acc = 0
  costs.forEach(c => { acc += c; cumulative.push(acc) })

  costChartInst = echarts.init(costChart.value)
  costChartInst.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['单题耗时', '累计耗时'], top: 0 },
    grid: [{ left: 40, right: 50, top: 30, height: '62%' }, { left: 40, right: 50, top: '76%', height: '18%' }],
    xAxis: [{ type: 'category', data: data.map(d => d.idx), gridIndex: 0 },
            { type: 'category', data: data.map(d => d.idx), gridIndex: 1 }],
    yAxis: [{ type: 'value', name: '耗时(s)', gridIndex: 0 },
            { type: 'value', name: '累计(s)', gridIndex: 1, inverse: true }],
    series: [
      {
        name: '单题耗时', type: 'bar', data: data.map(d => ({
          value: d.cost,
          itemStyle: { color: d.correct ? '#0D9488' : '#DC2626', borderRadius: [3, 3, 0, 0] }
        })),
        markLine: { symbol: 'none', data: [{ yAxis: p80, label: { formatter: '80分位 ' + p80 + 's' } }],
                    lineStyle: { color: '#EA580C', type: 'dashed' } }
      },
      { name: '累计耗时', type: 'line', xAxisIndex: 1, yAxisIndex: 1, data: cumulative,
        lineStyle: { color: '#6D28D9', width: 1.5 }, itemStyle: { color: '#6D28D9' } }
    ]
  })

  // 图2：难度柱状 + 正确率线（双轴）
  diffChartInst = echarts.init(diffChart.value)
  diffChartInst.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['难度', '正确率'], top: 0 },
    grid: { left: 40, right: 50, top: 30, bottom: 30 },
    xAxis: { type: 'category', data: data.map(d => d.idx) },
    yAxis: [
      { type: 'value', name: '难度', min: 0, max: 10 },
      { type: 'value', name: '正确率%', min: 0, max: 100 }
    ],
    series: [
      { name: '难度', type: 'bar', data: data.map(d => d.difficulty || 0),
        itemStyle: { color: '#0D9488', borderRadius: [3, 3, 0, 0] } },
      { name: '正确率', type: 'line', yAxisIndex: 1, data: data.map(() => null),
        lineStyle: { color: '#EA580C', width: 2 }, itemStyle: { color: '#EA580C' } }
    ]
  })
}

async function save() {
  saving.value = true
  try {
    const res = await reviewSummary({ exerciseId: route.params.id, summary: form.value })
    if (res.code === 200) ElMessage.success('已保存')
    else ElMessage.error('保存失败：' + (res.msg || ''))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const res = await getExercise(route.params.id)
    if (res.code === 200) {
      report.value = res.report || {}
      items.value = res.items || []
      if (res.macro) form.value = { ...res.macro }
    }
  } finally {
    loading.value = false
  }
  await nextTick()
  renderCharts()
})
</script>

<style scoped>
.ex-item {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 8px;
  border-bottom: 1px dashed #F0F0F4;
  cursor: pointer;
  border-radius: 6px;
  transition: background 150ms ease;
}
.ex-item:hover { background: #FAFAFC; }
.ex-item:last-child { border-bottom: none; }
.ex-item.wrong { background: #FFFDFB; }
.ex-idx { width: 26px; height: 26px; border-radius: 50%; background: var(--el-color-primary-light-9);
         color: var(--el-color-primary); display: flex; align-items: center; justify-content: center;
         font-size: 12px; font-weight: 600; flex-shrink: 0; }
.ex-item.wrong .ex-idx { background: #FEF2F2; color: var(--el-color-danger); }
.ex-title { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
