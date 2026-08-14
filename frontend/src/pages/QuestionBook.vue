<template>
  <div>
    <h1 class="page-title" style="display:flex;align-items:center;gap:12px;">
      {{ title }}
      <el-tooltip content="强制刷新（清除缓存）" placement="top">
        <el-button size="small" circle text @click="load(true)">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </el-tooltip>
    </h1>

    <!-- 筛选卡 -->
    <div class="bento-card" style="margin-bottom:20px;">
      <div style="display:flex;gap:10px;flex-wrap:wrap;">
        <el-tag v-for="m in modules" :key="m" :effect="module === m ? 'dark' : 'plain'"
                round size="large" style="cursor:pointer;"
                @click="switchModule(m)">{{ m }}</el-tag>
      </div>
      <template v-if="module && subs.length > 1">
        <el-divider style="margin:12px 0;" />
        <div style="display:flex;gap:10px;flex-wrap:wrap;">
          <el-tag v-for="s in subs" :key="s" :effect="sub === s ? 'dark' : 'plain'"
                  round size="default" style="cursor:pointer;"
                  @click="switchSub(s)">{{ s }}</el-tag>
        </div>
      </template>
      <el-divider style="margin:12px 0;" />
      <div style="display:flex;gap:10px;flex-wrap:wrap;">
        <el-radio-group v-model="range" @change="load">
          <el-radio-button value="">近90天</el-radio-button>
          <el-radio-button value="30">近30天</el-radio-button>
          <el-radio-button value="180">近180天</el-radio-button>
          <el-radio-button value="all">全部</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <el-empty v-if="!groups.length" description="暂无数据" />
        <div v-for="g in groups" :key="g.id" class="bento-card group-card">
          <div class="group-title">
            <span>{{ g.name }}</span>
            <span class="count">共 {{ g.count }} 题</span>
          </div>
          <div v-for="q in g.items" :key="q.id" class="q-row">
            <div class="q-item" :class="{ expanded: expandedId === q.id }" @click="toggleExpand(q)">
              <span class="diff-dot" :class="diffClass(q.difficulty)"></span>
              <span class="q-title">{{ q.title }}</span>
              <el-tag v-if="q.errorCategory" size="small" effect="plain" round :color="catColor(q.errorCategory)">{{ q.errorCategory }}</el-tag>
              <el-tag v-if="!q.errorCategory" size="small" type="warning" effect="plain" round>未复盘</el-tag>
              <el-tag v-if="q.reviewState === 'mastered'" size="small" type="success" effect="light" round>已掌握</el-tag>
              <el-tag v-if="q.reviewState === 'flagged'" size="small" type="danger" effect="light" round>重点</el-tag>
              <span class="q-diff">难度{{ q.difficulty }}</span>
              <div class="q-actions" @click.stop>
                <el-select :model-value="q.errorCategory" size="small" filterable allow-create clearable
                           placeholder="归类" style="width:100px;" @change="(v) => catChange(q, v)">
                  <el-option v-for="c in catOptions" :key="c" :label="c" :value="c" />
                </el-select>
                <el-tooltip content="重做做对" placement="top">
                  <el-button size="small" circle type="success" plain @click="quickAct(q, 'redo', 'correct')">
                    <el-icon><Check /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="重做做错" placement="top">
                  <el-button size="small" circle type="danger" plain @click="quickAct(q, 'redo', 'wrong')">
                    <el-icon><Close /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip :content="q.reviewState === 'flagged' ? '取消重点' : '标为重点'" placement="top">
                  <el-button size="small" circle :type="q.reviewState === 'flagged' ? 'warning' : 'default'" plain
                             @click="quickAct(q, 'state', q.reviewState === 'flagged' ? 'pending' : 'flagged')">
                    <el-icon><StarFilled /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="已掌握" placement="top">
                  <el-button size="small" circle type="success" plain
                             @click="quickAct(q, 'state', 'mastered')">
                    <el-icon><CircleCheck /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="归档" placement="top">
                  <el-button size="small" circle text @click="quickAct(q, 'archive', '1')">
                    <el-icon><FolderDelete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </div>

            <!-- 就地展开：题干 + 选项作答 + 解析 -->
            <div v-if="expandedId === q.id" class="q-detail">
              <el-skeleton v-if="detailLoading" :rows="4" animated />
              <template v-else-if="detail">
                <div v-if="detail.question.material" class="dq-material">{{ detail.question.material }}</div>
                <div class="dq-content" v-html="detail.question.content"></div>

                <el-alert v-if="detail.answered" :type="detail.isCorrect ? 'success' : 'error'" :closable="false"
                          show-icon :title="detail.isCorrect ? '回答正确' : '回答错误'" style="margin:12px 0;" />

                <div v-for="(opt, i) in detail.options" :key="i" class="dq-opt"
                     :class="{
                       correct: detail.answered && isRightChoice(i),
                       wrong: detail.answered && !isRightChoice(i) && i === detail.myChoice,
                       clickable: !detail.answered
                     }"
                     @click="!detail.answered && answer(i)">
                  <span class="opt-letter">{{ letter(i) }}.</span>
                  <span>{{ opt }}</span>
                  <el-icon v-if="detail.answered && isRightChoice(i)" color="#0D9488"><CircleCheckFilled /></el-icon>
                  <el-icon v-if="detail.answered && !isRightChoice(i) && i === detail.myChoice" color="#DC2626"><CircleCloseFilled /></el-icon>
                </div>

                <el-collapse v-if="detail.answered" style="margin-top:10px;">
                  <el-collapse-item title="查看解析" name="sol">
                    <div class="dq-sol" v-html="detail.solution.solution"></div>
                  </el-collapse-item>
                </el-collapse>

                <div style="margin-top:10px;display:flex;gap:8px;">
                  <el-button size="small" type="primary" plain @click="$router.push('/question/' + q.id)">进入完整复盘页</el-button>
                </div>
              </template>
            </div>
          </div>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'
import { getWrong, getCollects, forceRefresh } from '../api'

const props = defineProps({ kind: { type: String, default: 'wrong' } })
const title = props.kind === 'wrong' ? '错题本' : '我的收藏'

const loading = ref(true)
const groups = ref([])
const modules = ref([])
const module = ref('')
const sub = ref('')
const subs = ref([])
const range = ref('')

const PRESET_CATS = ['找数瞎', '列式乱', '计算蠢', '掉坑快', '死磕傻', '读不懂']
const catColors = { '找数瞎': '#0D9488', '列式乱': '#6D28D9', '计算蠢': '#EA580C', '掉坑快': '#DC2626', '死磕傻': '#92400E', '读不懂': '#0891B2' }

// 分类选项 = 预设 + 当前列表已用标签
const catOptions = computed(() => {
  const used = new Set()
  groups.value.forEach(g => g.items.forEach(q => { if (q.errorCategory) used.add(q.errorCategory) }))
  return [...new Set([...PRESET_CATS, ...used])]
})

const diffClass = (d) => d <= 4 ? 'easy' : (d >= 7 ? 'hard' : 'mid')
const catColor = (c) => catColors[c]

async function catChange(q, val) {
  if (!val) return
  const res = await http.post('/review/update', { questionId: String(q.id), field: 'categorize', value: val })
  if (res.data.code === 200) {
    q.errorCategory = val
    ElMessage.success('已归类')
  }
}

// 就地展开
const expandedId = ref(null)
const detailLoading = ref(false)
const detail = ref(null)

async function toggleExpand(q) {
  if (expandedId.value === q.id) {
    expandedId.value = null
    detail.value = null
    return
  }
  expandedId.value = q.id
  detail.value = null
  detailLoading.value = true
  try {
    const res = await http.get(`/question/${q.id}`)
    if (res.data.code === 200) {
      const qd = res.data.question
      detail.value = {
        question: qd,
        solution: res.data.solution || {},
        options: qd.accessories?.[0]?.options || [],
        answered: false,
        myChoice: -1,
        isCorrect: false
      }
    }
  } finally {
    detailLoading.value = false
  }
}

const letter = (i) => String.fromCharCode(65 + i)
const correctChoice = () => {
  // fenbi choice 为 0-based（0=A, 1=B, ...）
  const c = detail.value?.solution?.correctAnswer?.choice
  return c !== undefined && c !== null && c !== '' ? parseInt(c, 10) : -1
}
const isRightChoice = (i) => i === correctChoice()

function answer(i) {
  if (!detail.value || detail.value.answered) return
  detail.value.answered = true
  detail.value.myChoice = i
  detail.value.isCorrect = i === correctChoice()
}

async function quickAct(q, field, value) {
  const res = await http.post('/review/update', { questionId: String(q.id), field, value })
  if (res.data.code === 200) {
    if (field === 'redo') {
      q.redoCount = (q.redoCount || 0) + 1
      ElMessage.success(value === 'correct' ? '打卡成功' : '已记录')
    } else if (field === 'state') {
      q.reviewState = value
      ElMessage.success('已更新')
    } else if (field === 'archive') {
      ElMessage.success('已归档')
      load()
    }
  }
}

async function load(force) {
  loading.value = true
  try {
    const fn = props.kind === 'wrong' ? getWrong : getCollects
    if (force) {
      forceRefresh(props.kind === 'wrong' ? 'wrong:' : 'collects:')
    }
    const params = { module: module.value || undefined, sub: sub.value || undefined, range: range.value || undefined }
    const res = await fn(params)
    if (res.code === 200) {
      groups.value = res.groups
      if (!module.value) {
        modules.value = ['全部', ...new Set(res.groups.map(g => g.module))]
        subs.value = []
      } else {
        modules.value = ['全部', ...new Set(res.groups.map(g => g.module))]
        const all = await fn({ module: module.value, range: range.value || undefined })
        subs.value = ['全部', ...new Set(all.groups.map(g => g.sub))]
      }
    }
  } finally {
    loading.value = false
  }
}

function switchModule(m) {
  module.value = m === '全部' ? '' : m
  sub.value = ''
  load()
}

function switchSub(s) {
  sub.value = s === '全部' ? '' : s
  load()
}

onMounted(load)
</script>

<style scoped>
.group-card { margin-bottom: 20px; padding: 0; overflow: hidden; }
.group-title {
  display: flex; align-items: center; gap: 8px;
  padding: 14px 24px;
  font-size: 15px; font-weight: 600;
  border-bottom: 1px solid #F0F0F4;
}
.group-title .count { font-size: 12px; color: var(--el-text-color-secondary); font-weight: 400; }
.q-item {
  display: flex; align-items: center; gap: 10px;
  padding: 11px 24px;
  border-bottom: 1px dashed #F0F0F4;
  cursor: pointer;
  transition: background 150ms ease;
}
.q-item:hover { background: #FAFAFC; }
.q-item:last-child { border-bottom: none; }
.q-title { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.q-diff { font-size: 12px; color: var(--el-text-color-secondary); white-space: nowrap; }
.q-actions { display: flex; align-items: center; gap: 2px; flex-shrink: 0; }
.q-actions .el-button { margin-left: 0; }
.q-item.expanded { background: #FAFAFC; box-shadow: inset 0 0 0 1px var(--el-color-primary-light-7); }
.q-detail {
  padding: 14px 24px 18px;
  background: #FCFCFD;
  border-bottom: 1px dashed #F0F0F4;
  border-top: 1px solid var(--el-color-primary-light-8);
  line-height: 1.8;
  font-size: 14px;
}
.dq-material {
  background: #FAFAFC;
  border-left: 3px solid var(--el-color-primary);
  padding: 10px 14px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
}
.dq-content :deep(p) { margin-bottom: 8px; }
.dq-opt {
  display: flex; align-items: flex-start; gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  margin-top: 8px;
  font-size: 13px;
  border: 1px solid transparent;
}
.dq-opt.clickable { cursor: pointer; }
.dq-opt.clickable:hover { border-color: var(--el-color-primary-light-7); background: #FAFAFC; }
.dq-opt.correct { background: #F0FDF9; color: var(--el-color-success); border-color: #A7F3D0; }
.dq-opt.wrong { background: #FEF2F2; color: var(--el-color-danger); border-color: #FECACA; }
.opt-letter { font-weight: 600; flex-shrink: 0; }
.dq-sol :deep(p) { margin-bottom: 8px; }
</style>
