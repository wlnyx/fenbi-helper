<template>
  <div v-if="route.meta.auth" class="layout">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <span class="logo-badge"><el-icon :size="18"><Reading /></el-icon></span>
        <span>粉笔工作台</span>
      </div>
      <nav>
        <router-link v-for="item in items" :key="item.path" :to="item.path" class="nav-item" :class="{ active: route.path.startsWith(item.path) }">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
      <div class="sidebar-footer">
        <router-link to="/account" class="nav-item" :class="{ active: route.path.startsWith('/account') }">
          <el-icon><Setting /></el-icon>
          <span>账号设置</span>
        </router-link>
      </div>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
  <router-view v-else />
</template>

<script setup>
import { useRoute } from 'vue-router'

const route = useRoute()
const items = [
  { path: '/dashboard', label: '复盘工作台', icon: 'Odometer' },
  { path: '/history', label: '练习历史', icon: 'Document' },
  { path: '/wrong', label: '错题本', icon: 'CircleClose' },
  { path: '/collects', label: '我的收藏', icon: 'Star' },
  { path: '/review', label: '复习队列', icon: 'Calendar' }
]
</script>
