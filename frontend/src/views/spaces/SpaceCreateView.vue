<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRouter } from 'vue-router'
import { learningSpacesApi } from '@/api/learning-spaces'
import { getApiErrorMessage } from '@/api/client'
import type { CreateSpaceInput } from '@/types/learning-space'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive<CreateSpaceInput>({
  name: '',
  mode: 'exam',
  goal: '',
  exam_date: '',
  daily_minutes: 120,
})
const rules: FormRules = {
  name: [{ required: true, message: '请输入空间名称', trigger: 'blur' }],
  goal: [{ required: true, message: '请输入考试目标', trigger: 'blur' }],
  exam_date: [{ required: true, message: '请选择考试日期', trigger: 'change' }],
  daily_minutes: [
    {
      required: true,
      type: 'number',
      min: 15,
      max: 720,
      message: '请输入 15–720 分钟',
      trigger: 'blur',
    },
  ],
}
const disabledDate = (date: Date) => date.getTime() < new Date().setHours(0, 0, 0, 0)

async function submit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  loading.value = true
  try {
    const space = await learningSpacesApi.create(form)
    ElMessage.success('学习空间已创建')
    await router.push(`/spaces/${space.id}`)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-container narrow-page">
    <el-button text :icon="ArrowLeft" @click="$router.back()">返回</el-button>
    <header class="form-page-header">
      <p class="eyebrow">NEW LEARNING SPACE</p>
      <h1>设定你的考试目标</h1>
      <p>时间和目标越清晰，规划就越有效。</p>
    </header>
    <section class="form-panel">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="submit"
      >
        <el-form-item label="学习空间名称" prop="name"
          ><el-input v-model="form.name" size="large" placeholder="例如：计算机网络期末复习"
        /></el-form-item>
        <el-form-item label="我的目标" prop="goal"
          ><el-input
            v-model="form.goal"
            type="textarea"
            :rows="3"
            placeholder="例如：期末考试达到 80 分"
        /></el-form-item>
        <div class="form-row">
          <el-form-item label="考试日期" prop="exam_date"
            ><el-date-picker
              v-model="form.exam_date"
              value-format="YYYY-MM-DD"
              :disabled-date="disabledDate"
              size="large"
          /></el-form-item>
          <el-form-item label="每日可用时间" prop="daily_minutes"
            ><el-input-number
              v-model="form.daily_minutes"
              :min="15"
              :max="720"
              :step="15"
              size="large"
            /><span class="unit">分钟</span></el-form-item
          >
        </div>
        <div class="form-actions">
          <el-button size="large" @click="$router.back()">取消</el-button
          ><el-button type="primary" size="large" :loading="loading" native-type="submit"
            >创建并继续</el-button
          >
        </div>
      </el-form>
    </section>
  </div>
</template>
