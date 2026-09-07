<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Calendar, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/api/client'
import { learningSpacesApi } from '@/api/learning-spaces'
import type { LearningSpace } from '@/types/learning-space'
import { getDaysUntil } from '@/utils/date'

const loading = ref(true)
const spaces = ref<LearningSpace[]>([])
const greeting = computed(() =>
  new Date().getHours() < 12 ? '早上好' : new Date().getHours() < 18 ? '下午好' : '晚上好',
)

onMounted(async () => {
  try {
    spaces.value = (await learningSpacesApi.list()).items
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page-container">
    <header class="page-header">
      <div>
        <p class="eyebrow">{{ greeting }}</p>
        <h1>你的学习空间</h1>
        <p>选择一个目标，继续今天最重要的下一步。</p>
      </div>
      <el-button type="primary" size="large" :icon="Plus" @click="$router.push('/spaces/new')"
        >创建学习空间</el-button
      >
    </header>
    <div v-if="loading" class="space-grid">
      <el-skeleton v-for="i in 3" :key="i" animated :rows="5" />
    </div>
    <el-empty v-else-if="!spaces.length" description="还没有学习空间">
      <el-button type="primary" @click="$router.push('/spaces/new')">创建第一个空间</el-button>
    </el-empty>
    <div v-else class="space-grid">
      <article
        v-for="space in spaces"
        :key="space.id"
        class="space-card"
        @click="$router.push(`/spaces/${space.id}`)"
      >
        <div class="space-card-top">
          <span class="mode-badge">EXAM</span
          ><span class="days-left">{{ getDaysUntil(space.exam_date) }} 天后考试</span>
        </div>
        <h2>{{ space.name }}</h2>
        <p>{{ space.goal }}</p>
        <div class="space-meta">
          <span
            ><el-icon><Calendar /></el-icon>{{ space.exam_date }}</span
          ><span>{{ space.daily_minutes }} 分钟/天</span>
        </div>
        <div class="progress-track"><span class="progress-placeholder"></span></div>
        <small>等待构建学习图谱</small>
      </article>
      <button class="space-card create-card" @click="$router.push('/spaces/new')">
        <el-icon><Plus /></el-icon><span>添加新目标</span>
      </button>
    </div>
  </div>
</template>
