import { createApp } from 'vue'
import 'element-plus/dist/index.css'
import {
  Reading, Setting, Odometer, Document, CircleClose, Star, Calendar, Timer,
  DataAnalysis, TrendCharts, ArrowRight, EditPen, StarFilled, List, CircleCheck,
  QuestionFilled, ArrowLeft, CircleCheckFilled, Edit, Loading, User,
  Check, Close, FolderDelete, Refresh
} from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import './theme.css'

const app = createApp(App)
const icons = { Reading, Setting, Odometer, Document, CircleClose, Star, Calendar, Timer, DataAnalysis, TrendCharts, ArrowRight, EditPen, StarFilled, List, CircleCheck, QuestionFilled, ArrowLeft, CircleCheckFilled, Edit, Loading, User, Check, Close, FolderDelete, Refresh }
for (const [name, comp] of Object.entries(icons)) {
  app.component(name, comp)
}
app.use(router)
app.mount('#app')
