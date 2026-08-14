<template>
  <div>
    <el-skeleton :loading="loading" animated :rows="12">
      <template #default>
        <div style="display:flex;gap:8px;align-items:center;margin-bottom:16px;flex-wrap:wrap;">
          <el-button text @click="$router.back()"><el-icon><ArrowLeft /></el-icon> 返回</el-button>
          <el-tag size="small" round>{{ q.type }}</el-tag>
          <el-tag size="small" round type="warning">难度 {{ q.difficulty }}/10</el-tag>
          <el-tag v-if="review.reviewState === 'mastered'" size="small" round type="success">已掌握</el-tag>
          <el-tag v-if="review.reviewState === 'flagged'" size="small" round type="danger">重点</el-tag>
          <el-tag v-if="review.doubt" size="small" round type="warning">存疑</el-tag>
        </div>

        <!-- 题目卡 -->
        <div class="bento-card" style="margin-bottom:20px;">
          <div v-if="q.material" class="q-material" v-html="q.material"></div>
          <div class="q-content" v-html="q.content"></div>

          <div v-if="!answered" style="margin-top:16px;">
            <div v-for="(opt, i) in options" :key="i" class="opt" @click="answer(i)">
              <span class="opt-letter">{{ letter(i) }}.</span>
              <span v-html="opt"></span>
            </div>
          </div>

          <template v-else>
            <el-alert :type="isCorrect ? 'success' : 'error'" :closable="false" show-icon
                      :title="isCorrect ? '回答正确' : '回答错误'" style="margin-bottom:14px;" />
            <div v-for="(opt, i) in options" :key="i" class="opt"
                 :class="{ correct: isRightChoice(i), wrong: !isRightChoice(i) && i === myChoice }">
              <span class="opt-letter">{{ letter(i) }}.</span>
              <span v-html="opt"></span>
              <el-icon v-if="isRightChoice(i)" color="#0D9488"><CircleCheckFilled /></el-icon>
            </div>
          </template>

          <el-collapse v-if="answered" style="margin-top:14px;">
            <el-collapse-item title="查看解析" name="sol">
              <div class="sol-content" v-html="sol.solution"></div>
            </el-collapse-item>
          </el-collapse>
        </div>

        <!-- 六步复盘卡 -->
        <div class="bento-card">
          <div class="bento-card-title"><el-icon><Edit /></el-icon> 六步复盘</div>

          <!-- ③ 错误归类 -->
          <div class="rp-section">
            <div class="rp-label">③ 错误归类（可输入自定义标签）</div>
            <el-select v-model="category" filterable allow-create clearable default-first-option
                       placeholder="选择或输入自定义标签" style="width:320px;" @change="saveCategory">
              <el-option v-for="c in categoryOptions" :key="c" :label="c" :value="c" />
            </el-select>
          </div>

          <!-- ④ 四句话 -->
          <div class="rp-section">
            <div class="rp-label">④ 四句话记录</div>
            <el-form label-position="top" size="default">
              <el-form-item label="① 考点是什么">
                <el-input v-model="note4.point" placeholder="本题考查的知识点" />
              </el-form-item>
              <el-form-item label="② 错在哪里">
                <el-input v-model="note4.mistake" placeholder="看错主体/公式记混/计算失误…" />
              </el-form-item>
              <el-form-item label="③ 正确计算思路">
                <el-input v-model="note4.approach" type="textarea" :rows="2" placeholder="正确的解题路径" />
              </el-form-item>
              <el-form-item label="④ 长效提醒 / 同类题举一反三">
                <el-input v-model="note4.reminder" type="textarea" :rows="2" placeholder="下次遇到同类题要注意什么" />
              </el-form-item>
              <div>
                <el-button type="primary" :loading="savingNote" @click="saveNote4(false)">保存四句话</el-button>
                <el-button :loading="savingNote" @click="saveNote4(true)">保存并同步粉笔笔记</el-button>
              </div>
            </el-form>
          </div>

          <!-- ②⑥ 重做与状态 -->
          <div class="rp-section">
            <div class="rp-label">② 重做打卡 · ⑥ 复习状态</div>
            <div style="display:flex;gap:8px;flex-wrap:wrap;">
              <el-button type="success" plain @click="reviewAct('redo', 'correct')">重做做对</el-button>
              <el-button type="danger" plain @click="reviewAct('redo', 'wrong')">重做做错</el-button>
              <el-button type="warning" plain @click="reviewAct('state', 'flagged')">标为重点</el-button>
              <el-button type="success" plain @click="reviewAct('state', 'mastered')">标记已掌握</el-button>
              <el-button plain @click="reviewAct('doubt', review.doubt ? '0' : '1')">
                {{ review.doubt ? '取消存疑' : '标记存疑' }}
              </el-button>
              <el-button text @click="reviewAct('archive', '1')">归档</el-button>
            </div>
            <div v-if="review.redoHistory?.length" class="redo-history">
              重做历史：
              <el-tag v-for="(h, i) in review.redoHistory" :key="i" size="small"
                      :type="h.result === 'correct' ? 'success' : 'danger'" effect="plain" round>
                {{ h.date }} {{ h.result === 'correct' ? '✓' : '✗' }}
              </el-tag>
            </div>
          </div>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api'

const route = useRoute()
const qid = route.params.id
const loading = ref(true)
const q = ref({})
const sol = ref({})
const review = ref({ redoHistory: [] })
const categories = ref([])
const usedCategories = ref([])
const categoryOptions = computed(() => [...new Set([...categories.value, ...usedCategories.value])])
const options = computed(() => (q.value.accessories?.[0]?.options) || [])

const answered = ref(false)
const myChoice = ref(-1)
const isCorrect = ref(false)
const correctChoice = computed(() => {
  // fenbi choice 为 0-based（0=A, 1=B, ...）
  const c = sol.value.correctAnswer?.choice
  return c !== undefined && c !== null && c !== '' ? parseInt(c, 10) : -1
})
const category = ref('')
const note4 = ref({ point: '', mistake: '', approach: '', reminder: '' })
const savingNote = ref(false)

const letter = (i) => String.fromCharCode(65 + i)
const isRightChoice = (i) => i === correctChoice.value

function answer(i) {
  if (answered.value) return
  answered.value = true
  myChoice.value = i
  isCorrect.value = i === correctChoice.value
}

async function reviewAct(field, value) {
  const res = await http.post('/review/update', { questionId: qid, field, value })
  if (res.data.code === 200) { ElMessage.success('已更新'); load() }
}

async function saveCategory(val) {
  const res = await http.post('/review/update', { questionId: qid, field: 'categorize', value: val })
  if (res.data.code === 200) ElMessage.success('已归类')
}

async function saveNote4(sync) {
  savingNote.value = true
  try {
    const res = await http.post('/review/note4', { questionId: qid, syncFenbi: sync, note4: note4.value })
    if (res.data.code === 200) ElMessage.success(sync ? '已保存并同步粉笔' : '已保存')
  } finally {
    savingNote.value = false
  }
}

async function load() {
  loading.value = true
  try {
    const res = await http.get(`/question/${qid}`)
    if (res.data.code === 200) {
      q.value = res.data.question
      sol.value = res.data.solution || {}
      review.value = res.data.review || { redoHistory: [] }
      categories.value = res.data.categories || []
      usedCategories.value = res.data.usedCategories || []
      category.value = review.value.errorCategory || ''
      note4.value = review.value.note4 || { point: '', mistake: '', approach: '', reminder: '' }
    }
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.q-material {
  background: #FAFAFC;
  border-left: 3px solid var(--el-color-primary);
  padding: 12px 16px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.7;
  white-space: pre-wrap;
}
.q-material :deep(img), .q-content :deep(img), .opt :deep(img) { max-width: 100%; vertical-align: middle; }
.q-material :deep(p), .opt :deep(p) { margin: 0; }
.q-content { line-height: 1.8; font-size: 15px; }
.q-content :deep(p) { margin-bottom: 8px; }
.opt {
  display: flex; align-items: flex-start; gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  margin-top: 10px;
  cursor: pointer;
  transition: all 150ms ease;
  font-size: 14px; line-height: 1.6;
}
.opt:hover { border-color: var(--el-color-primary); }
.opt.correct { border-color: var(--el-color-success); background: #F0FDF9; }
.opt.wrong { border-color: var(--el-color-danger); background: #FEF2F2; }
.opt-letter { font-weight: 600; color: var(--el-color-primary); flex-shrink: 0; }
.opt.correct .opt-letter { color: var(--el-color-success); }
.opt.wrong .opt-letter { color: var(--el-color-danger); }
.sol-content { line-height: 1.8; font-size: 14px; }
.sol-content :deep(p) { margin-bottom: 8px; }
.rp-section { padding: 16px 0; border-top: 1px solid var(--el-border-color-lighter); }
.rp-label { font-size: 13px; font-weight: 600; color: var(--el-text-color-regular); margin-bottom: 10px; }
.redo-history { margin-top: 12px; font-size: 12px; color: var(--el-text-color-secondary); display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
</style>
