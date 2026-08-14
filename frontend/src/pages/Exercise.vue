<template>
  <div>
    <h1 class="page-title">练习详情</h1>
    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <!-- 统计 -->
        <div style="display:grid;grid-template-columns:repeat(5,1fr);gap:16px;margin-bottom:20px;">
          <div v-for="k in kpis" :key="k.label" class="bento-card" style="text-align:center;padding:18px 12px;">
            <div class="kpi-num" :style="{ fontSize: k.big ? '28px' : '22px' }">{{ k.value }}</div>
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
  const n = r.answerCount || 0
  const rate = n > 0 ? (r.correctCount / n * 100).toFixed(1) : '-'
  const avg = n > 0 && r.elapsedTime ? (r.elapsedTime / n).toFixed(0) : '-'
  return [
    { label: '做题数', value: n, big: true },
    { label: '正确数', value: r.correctCount ?? '-', big: false },
    { label: '正确率', value: n ? rate + '%' : '-', big: false },
    { label: '平均耗时', value: avg != '-' ? avg + 's/题' : '-', big: false },
    { label: '累计耗时', value: r.elapsedTime != null ? fmtDur(r.elapsedTime) : '-', big: false }
  ]
})

const fmtDur = (sec) => {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return m > 0 ? `${m}分${s}秒` : `${s}秒`
}

function renderCharts() {
  const data = items.value
  if (!data.length) return

  const C_OK = '#0D9488'
  const C_BAD = '#DC2626'
  const C_ACCENT = '#EA580C'
  const C_PURPLE = '#6D28D9'

  // 图1：单题耗时横向条形（每行一题，对错着色 + 平均/80分位参考线）
  const costs = data.map(d => d.cost || 0)
  const sorted = [...costs].sort((a, b) => a - b)
  const p80 = sorted[Math.floor(sorted.length * 0.8)] || 0
  const avg = Math.round(costs.reduce((a, b) => a + b, 0) / costs.length) || 0

  costChartInst = echarts.init(costChart.value)
  costChartInst.setOption({
    animationDuration: 800,
    animationEasing: 'cubicOut',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: 'rgba(29,29,31,.92)',
      borderWidth: 0,
      textStyle: { color: '#fff', fontSize: 12 },
      formatter: (params) => {
        const d = data[params[0].dataIndex]
        return `<b>第 ${d.idx} 题</b><br/>耗时：${d.cost}s<br/>结果：${d.correct ? '✓ 正确' : '✗ 错误'}<br/>难度：${d.difficulty}/10`
      }
    },
    legend: {
      data: [{ name: '正确', itemStyle: { color: C_OK } }, { name: '错误', itemStyle: { color: C_BAD } }],
      top: 0, icon: 'circle', itemWidth: 8, textStyle: { fontSize: 12 }
    },
    grid: { left: 52, right: 64, top: 30, bottom: 24 },
    xAxis: {
      type: 'value', name: '耗时(s)', nameTextStyle: { fontSize: 10, color: '#86868B' },
      axisLabel: { fontSize: 10, color: '#86868B' }, splitLine: { lineStyle: { color: '#F0F0F4' } }
    },
    yAxis: {
      type: 'category', inverse: true,
      data: data.map(d => '第' + d.idx + '题'),
      axisLine: { show: false }, axisTick: { show: false },
      axisLabel: { fontSize: 11, color: '#3a3a3c' }
    },
    series: [{
      name: '正确', type: 'bar', stack: 't', barWidth: 14,
      data: data.map(d => d.correct ? d.cost : null),
      itemStyle: { color: C_OK, borderRadius: [0, 4, 4, 0] },
      label: { show: true, position: 'right', fontSize: 10, color: '#3a3a3c', formatter: (p) => p.value != null ? p.value + 's' : '' },
      markLine: {
        symbol: 'none', silent: true,
        label: { fontSize: 10, color: '#3a3a3c' },
        data: [
          { xAxis: avg, label: { formatter: '平均 ' + avg + 's', position: 'insideEndTop', color: C_OK },
            lineStyle: { color: C_OK, type: 'dashed', width: 1 } },
          { xAxis: p80, label: { formatter: '80分位 ' + p80 + 's', position: 'insideEndTop', color: C_ACCENT },
            lineStyle: { color: C_ACCENT, type: 'dashed', width: 1 } }
        ]
      }
    }, {
      name: '错误', type: 'bar', stack: 't', barWidth: 14,
      data: data.map(d => !d.correct ? d.cost : null),
      itemStyle: { color: C_BAD, borderRadius: [0, 4, 4, 0] },
      label: { show: true, position: 'right', fontSize: 10, color: '#3a3a3c', formatter: (p) => p.value != null ? p.value + 's' : '' }
    }]
  })

  // 图2：难度柱状 + 正确率线（双轴）
  const hasRatio = data.some(d => d.correctRatio > 0)
  diffChartInst = echarts.init(diffChart.value)
  diffChartInst.setOption({
    animationDuration: 800,
    animationEasing: 'cubicOut',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(29,29,31,.92)',
      borderWidth: 0,
      textStyle: { color: '#fff', fontSize: 12 },
      formatter: (params) => {
        const d = data[params[0].dataIndex]
        return `<b>第 ${d.idx} 题</b><br/>难度：${d.difficulty}/10<br/>${hasRatio ? '全网正确率：' + d.correctRatio.toFixed(1) + '%' : ''}`
      }
    },
    legend: { data: ['难度', '全网正确率'], top: 0, icon: 'circle', itemWidth: 8, textStyle: { fontSize: 12 } },
    grid: { left: 44, right: 56, top: 34, bottom: 30 },
    xAxis: {
      type: 'category', data: data.map(d => d.idx),
      axisLine: { lineStyle: { color: '#E8E8ED' } }, axisTick: { show: false },
      axisLabel: { fontSize: 10, color: '#86868B' }
    },
    yAxis: [
      { type: 'value', name: '难度', min: 0, max: 10, nameTextStyle: { fontSize: 10, color: '#86868B' },
        axisLabel: { fontSize: 10, color: '#86868B' }, splitLine: { lineStyle: { color: '#F0F0F4' } } },
      { type: 'value', name: '正确率%', min: 0, max: 100,
        nameTextStyle: { fontSize: 10, color: '#86868B' }, axisLabel: { fontSize: 10, color: '#86868B' },
        splitLine: { show: false } }
    ],
    series: [
      {
        name: '难度', type: 'bar', barWidth: '45%',
        data: data.map(d => ({
          value: d.difficulty || 0,
          itemStyle: {
            color: d.difficulty >= 7 ? 'rgba(220,38,38,.75)' : (d.difficulty <= 4 ? 'rgba(13,148,136,.85)' : 'rgba(180,83,9,.75)'),
            borderRadius: [4, 4, 0, 0]
          }
        }))
      },
      ...(hasRatio ? [{
        name: '全网正确率', type: 'line', yAxisIndex: 1,
        data: data.map(d => d.correctRatio ? +d.correctRatio.toFixed(1) : null),
        smooth: true, symbol: 'circle', symbolSize: 5,
        lineStyle: { color: C_ACCENT, width: 2 }, itemStyle: { color: C_ACCENT }
      }] : [])
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
