<script setup lang="ts">
import { ArrowRight, Plus } from '@element-plus/icons-vue'
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { learningSpacesApi } from '@/api/learning-spaces'
import type { LearningSpace } from '@/types/learning-space'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const spaces = ref<LearningSpace[]>([])

async function loadSpaces() {
  try { spaces.value = (await learningSpacesApi.list()).items } catch { spaces.value = [] }
}
onMounted(loadSpaces)
watch(() => route.fullPath, loadSpaces)

async function logout() {
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <div class="app-shell">
    <aside class="app-sidebar">
      <RouterLink class="brand" to="/spaces">
        <span class="brand-mark">W</span>
        <span>WhatsNext</span>
      </RouterLink>
      <nav class="main-nav">
        <div class="sidebar-nav-title"><span>学习空间</span><RouterLink to="/spaces/new" aria-label="创建学习空间"><el-icon><Plus /></el-icon></RouterLink></div>
        <RouterLink class="all-spaces-link" to="/spaces" active-class="sidebar-parent-active" exact-active-class="router-link-active">全部学习空间</RouterLink>
        <RouterLink v-for="space in spaces" :key="space.id" class="sidebar-space-link" :to="`/spaces/${space.id}`">
          <span class="space-dot" />
          <span>{{ space.name }}</span>
        </RouterLink>
        <p v-if="!spaces.length" class="sidebar-empty">还没有学习空间</p>
      </nav>
      <div class="sidebar-footer">
        <div class="avatar">{{ auth.user?.display_name.slice(0, 1) }}</div>
        <div class="user-copy">
          <strong>{{ auth.user?.display_name }}</strong>
          <span>{{ auth.user?.email }}</span>
        </div>
        <el-button text circle aria-label="退出登录" @click="logout"
          ><el-icon><ArrowRight /></el-icon
        ></el-button>
      </div>
    </aside>
    <main class="app-main"><RouterView /></main>
  </div>
</template>
