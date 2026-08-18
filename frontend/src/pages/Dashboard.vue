<template>
  <div>
    <h1 class="page-title" style="display:flex;align-items:center;gap:12px;">
      复盘工作台
      <el-tooltip content="强制刷新" placement="top">
        <el-button size="small" circle text @click="reload">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </el-tooltip>
    </h1>
    <el-skeleton :loading="loading" animated :rows="12">
      <template #default>
        <!-- KPI 统计行 -->
        <div class="kpi-grid">
          <div v-for="k in kpis" :key="k.label" class="bento-card" style="text-align:center;padding:18px 12px;">
            <div class="kpi-num" :style="{ color: k.color }">{{ k.value }}</div>
            <div class="kpi-label">{{ k.label }}</div>
          </div>
        </div>

        <!-- 图表区：趋势 + Donut -->
        <div class="grid-2">
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><TrendCharts /></el-icon> 近30天练习趋势</div>
            <div ref="trendChart" class="chart-box"></div>
          </div>
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><PieChart /></el-icon> 错误类型分布</div>
            <div v-if="totalCat === 0" class="empty-hint">还没有归类记录</div>
            <div v-else ref="donutChart" class="chart-box"></div>
          </div>
        </div>

        <!-- 热力图 + Waffle -->
        <div class="grid-2">
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><Calendar /></el-icon> 年度刷题热力图</div>
            <div v-if="!hasHeatmap" class="empty-hint">暂无练习数据</div>
            <div v-else ref="heatmapChart" class="chart-box heatmap-box"></div>
          </div>
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><Odometer /></el-icon> 复习状态占比</div>
            <div v-if="stateTotal === 0" class="empty-hint">暂无复盘数据</div>
            <div v-else ref="waffleChart" class="waffle-box"></div>
          </div>
        </div>

        <!-- 雷达 + 今日复习 -->
        <div class="grid-2">
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><Aim /></el-icon> 模块能力雷达</div>
            <div v-if="!moduleStats.length" class="empty-hint">暂无错题数据</div>
            <div v-else ref="radarChart" class="chart-box"></div>
          </div>
          <div class="bento-card">
            <div class="bento-card-title"><el-icon><List /></el-icon> 今日复习</div>
            <el-empty v-if="!data.queue.length" description="暂无待复习题目" :image-size="50">
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
        </div>

        <!-- 近期练习 -->
        <div class="bento-card">
          <div class="bento-card-title"><el-icon><Document /></el-icon> 近期练习</div>
          <el-empty v-if="!data.recentHistory.length" description="暂无练习记录" :image-size="50" />
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { getDashboard, forceRefresh } from '../api'

const loading = ref(true)
const data = ref({ queue: [], recentHistory: [], categoryCount: {}, trend: [], heatmap: {}, moduleStats: [] })
const categories = ['找数瞎', '列式乱', '计算蠢', '掉坑快', '死磕傻', '读不懂']
const catColors = ['#0D9488', '#6D28D9', '#EA580C', '#DC2626', '#92400E', '#0891B2']
const stateColors = { 待复习: '#86868B', 重点: '#EA580C', 已掌握: '#0D9488', 未复盘: '#B45309' }

const chartRefs = {}
const trendChart = ref(null); chartRefs.trend = trendChart
const donutChart = ref(null); chartRefs.donut = donutChart
const heatmapChart = ref(null); chartRefs.heatmap = heatmapChart
const waffleChart = ref(null); chartRefs.waffle = waffleChart
const radarChart = ref(null); chartRefs.radar = radarChart
let instances = []

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
const hasHeatmap = computed(() => Object.keys(data.value.heatmap || {}).length > 0)
const moduleStats = computed(() => data.value.moduleStats || [])

const pctCat = (n) => (totalCat.value ? Math.round(n / totalCat.value * 100) : 0)
const catColor = (c) => catColors[categories.indexOf(c)]

const TOOLTIP_STYLE = {
  backgroundColor: 'rgba(29,29,31,.92)', borderWidth: 0,
  textStyle: { color: '#fff', fontSize: 12 }
}

function makeInstance(refName) {
  const el = chartRefs[refName].value
  if (!el) return null
  const inst = echarts.getInstanceByDom(el) || echarts.init(el)
  instances.push(inst)
  return inst
}

function renderTrend() {
  const trend = data.value.trend || []
  if (!trend.length) return
  const inst = makeInstance('trend')
  inst.setOption({
    animationDuration: 800,
    tooltip: { ...TOOLTIP_STYLE, trigger: 'axis' },
    legend: { data: ['练习量', '正确率'], top: 0, icon: 'circle', itemWidth: 8, textStyle: { fontSize: 12 } },
    grid: { left: 44, right: 50, top: 32, bottom: 24 },
    xAxis: { type: 'category', data: trend.map(t => t.date), axisLine: { lineStyle: { color: '#E8E8ED' } }, axisTick: { show: false }, axisLabel: { fontSize: 10, color: '#86868B', interval: 4 } },
    yAxis: [
      { type: 'value', name: '题数', nameTextStyle: { fontSize: 10, color: '#86868B' }, axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { color: '#F0F0F4' } } },
      { type: 'value', name: '正确率%', min: 0, max: 100, nameTextStyle: { fontSize: 10, color: '#86868B' }, axisLabel: { fontSize: 10 }, splitLine: { show: false } }
    ],
    series: [
      { name: '练习量', type: 'bar', data: trend.map(t => t.count), barWidth: '55%',
        itemStyle: { color: 'rgba(13,148,136,.85)', borderRadius: [3, 3, 0, 0] } },
      { name: '正确率', type: 'line', yAxisIndex: 1, data: trend.map(t => t.correctRate),
        smooth: true, symbol: 'circle', symbolSize: 4,
        lineStyle: { color: '#EA580C', width: 2 }, itemStyle: { color: '#EA580C' } }
    ]
  })
}

function renderDonut() {
  const inst = makeInstance('donut')
  const data2 = categories.map((c, i) => ({ name: c, value: catCount.value[c] || 0 })).filter(d => d.value > 0)
  inst.setOption({
    animationDuration: 800,
    tooltip: { ...TOOLTIP_STYLE, trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0, icon: 'circle', itemWidth: 8, textStyle: { fontSize: 11 }, itemGap: 8 },
    series: [{
      type: 'pie', radius: ['52%', '72%'], center: ['50%', '44%'],
      data: data2,
      itemStyle: { borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 13, fontWeight: 600 } },
      labelLine: { show: false }
    }]
  })
  // 中心总数（graphic 的 left 支持数字/百分比，'center' 非法）
  inst.setOption({
    graphic: [{
      type: 'text', left: '50%', top: '38%',
      style: { text: String(totalCat.value), textAlign: 'center', fill: '#1D1D1F', fontSize: 26, fontWeight: 700 }
    }, {
      type: 'text', left: '50%', top: '50%',
      style: { text: '已归类', textAlign: 'center', fill: '#86868B', fontSize: 11 }
    }]
  })
}

function renderHeatmap() {
  const inst = makeInstance('heatmap')
  const hm = data.value.heatmap || {}
  // 当前年份的日历热力图（range 动态，不硬编码年份）
  const year = new Date().getFullYear()
  const max = Math.max(...Object.values(hm), 1)
  const cells = []
  for (let m = 0; m < 12; m++) {
    const daysInMonth = new Date(year, m + 1, 0).getDate()
    for (let d = 1; d <= daysInMonth; d++) {
      const key = `${year}-${String(m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`
      const v = hm[key] || 0
      if (v > 0) cells.push([key, v])
    }
  }
  inst.setOption({
    animationDuration: 600,
    tooltip: {
      ...TOOLTIP_STYLE,
      formatter: (p) => `${p.data[0]}<br/>刷题：${p.data[1]} 题`
    },
    calendar: {
      top: 24, left: 32, right: 12, bottom: 34,
      range: year,
      cellSize: ['auto', 11],
      itemStyle: { borderWidth: 2, borderColor: '#fff', borderRadius: 2 },
      splitLine: { lineStyle: { color: '#F0F0F4' } },
      dayLabel: { fontSize: 8, color: '#86868B' },
      monthLabel: { fontSize: 9, color: '#86868B' },
      yearLabel: { show: false }
    },
    visualMap: {
      min: 0, max, calculable: false, orient: 'horizontal', left: 'center', bottom: 0,
      itemWidth: 10, itemHeight: 60,
      inRange: { color: ['#F0FDFA', '#A3DCD6', '#0D9488', '#065F56'] },
      text: ['多', '少'], textStyle: { fontSize: 9, color: '#86868B' }
    },
    series: [{
      type: 'heatmap', coordinateSystem: 'calendar',
      data: cells,
      itemStyle: { borderColor: '#fff', borderWidth: 1, borderRadius: 2 }
    }]
  })
}

function renderWaffle() {
  const inst = makeInstance('waffle')
  // 100 格 Waffle，按状态占比填充
  const cells = []
  const statesData = states.value.filter(s => s.value > 0)
  const total = stateTotal.value || 1
  let acc = 0
  statesData.forEach(s => {
    const n = Math.round(s.value / total * 100)
    for (let i = 0; i < n; i++) {
      cells.push({ value: s.value, itemStyle: { color: s.color } })
      acc++
    }
  })
  while (cells.length < 100) cells.push({ value: 0, itemStyle: { color: '#F0F0F4' } })
  inst.setOption({
    animationDuration: 500,
    tooltip: {
      ...TOOLTIP_STYLE,
      formatter: (p) => {
        if (!p.data.value) return '未复盘空间'
        return `状态：${statesData.find(s => s.value === p.data.value)?.label || ''}<br/>数量：${p.data.value}`
      }
    },
    grid: { left: 8, right: 8, top: 8, bottom: 30 },
    xAxis: { type: 'category', data: Array.from({ length: 10 }, (_, i) => i + 1), splitArea: { show: false }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { show: false } },
    yAxis: { type: 'category', data: Array.from({ length: 10 }, (_, i) => i + 1), splitArea: { show: false }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { show: false } },
    visualMap: { show: false },
    series: [{
      type: 'heatmap',
      data: cells.map((c, i) => [i % 10, Math.floor(i / 10), c.value]),
      itemStyle: { borderColor: '#fff', borderWidth: 1.5, borderRadius: 2 }
    }]
  })
  // 图例（按容器宽度动态分布，避免窄卡片溢出）
  const w = inst.getWidth() || 400
  const gap = Math.min(96, Math.floor((w - 16) / Math.max(statesData.length, 1)))
  inst.setOption({
    graphic: statesData.map((s, i) => ({
      type: 'group', left: 8 + i * gap, bottom: 4,
      children: [
        { type: 'rect', shape: { x: 0, y: 0, width: 10, height: 10 }, style: { fill: s.color } },
        { type: 'text', left: 14, top: -2, style: { text: `${s.label} ${s.value}`, fill: '#86868B', fontSize: 10 } }
      ]
    }))
  })
}

function renderRadar() {
  const inst = makeInstance('radar')
  const ms = moduleStats.value
  const names = ms.map(m => m.module)
  const maxCount = Math.max(...ms.map(m => m.count), 1)
  // 统一量纲到 0-100：错题数按峰值归一，难度按 10 分制归一
  const countPct = ms.map(m => Math.round(m.count / maxCount * 100))
  const diffPct = ms.map(m => Math.min(100, Math.round(m.avgDiff / 10 * 100)))
  const byName = {}
  ms.forEach((m, i) => { byName[m.module] = { m, countPct: countPct[i], diffPct: diffPct[i] } })
  inst.setOption({
    animationDuration: 800,
    tooltip: {
      ...TOOLTIP_STYLE, trigger: 'item',
      formatter: (p) => {
        // radar 的 p.name 是指标轴名（模块名），单数据项时 dataIndex 恒为 0，须用 name 匹配
        const hit = byName[p.name]
        if (!hit) return p.name
        return `${p.name}<br/>错题数：${hit.m.count}<br/>平均难度：${hit.m.avgDiff}/10<br/>占比：${hit.countPct}%`
      }
    },
    legend: { data: ['错题数占比', '难度占比'], top: 0, icon: 'circle', itemWidth: 8, textStyle: { fontSize: 11 } },
    radar: {
      indicator: names.map(n => ({ name: n, max: 100 })),
      radius: '60%', center: ['50%', '55%'],
      axisName: { fontSize: 10, color: '#3a3a3c' },
      splitArea: { areaStyle: { color: ['#fff', '#FAFAFC'] } },
      splitLine: { lineStyle: { color: '#E8E8ED' } },
      axisLine: { lineStyle: { color: '#E8E8ED' } }
    },
    series: [
      { type: 'radar', name: '错题数占比', data: [{ value: countPct, itemStyle: { color: '#0D9488' }, areaStyle: { color: 'rgba(13,148,136,.18)' } }] },
      { type: 'radar', name: '难度占比', data: [{ value: diffPct, itemStyle: { color: '#EA580C' }, areaStyle: { color: 'rgba(234,88,12,.15)' } }] }
    ]
  })
}

function renderAll() {
  // 复用实例，notMerge 全量替换配置（避免 dispose 重建闪烁）
  instances = []
  renderTrend()
  renderDonut()
  renderHeatmap()
  renderWaffle()
  renderRadar()
}

let resizeTimer = null
function onResize() {
  clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    instances.forEach(i => i.resize())
  }, 150)
}

async function load() {
  loading.value = true
  try {
    const res = await getDashboard()
    if (res.code === 200) data.value = res.data
  } finally {
    loading.value = false
    setTimeout(renderAll, 50)
  }
}

function reload() {
  forceRefresh('dashboard')
  load()
}

onMounted(() => {
  load()
  window.addEventListener('resize', onResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  instances.forEach(i => i.dispose())
})
</script>

<style scoped>
.kpi-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px; }
.grid-2 { display: grid; grid-template-columns: 1.4fr 1fr; gap: 16px; margin-bottom: 20px; }
@media (max-width: 1024px) {
  .grid-2 { grid-template-columns: 1fr; }
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
}
.chart-box { height: 260px; }
.heatmap-box { height: 190px; }
.waffle-box { height: 240px; display: flex; align-items: center; }
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
.empty-hint { color: var(--el-text-color-secondary); padding: 40px 0; text-align: center; font-size: 13px; }
</style>
