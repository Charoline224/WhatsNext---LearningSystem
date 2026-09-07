<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AuthLayout from '@/layouts/AuthLayout.vue'
import { getApiErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({ email: '', password: '' })
const rules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  password: [{ required: true, min: 8, message: '密码至少 8 位', trigger: 'blur' }],
}
const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

async function submit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  loading.value = true
  try {
    await auth.login(form)
    await router.push(typeof route.query.redirect === 'string' ? route.query.redirect : '/spaces')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <p class="eyebrow">欢迎回来</p>
    <h2>登录 WhatsNext</h2>
    <p class="form-intro">继续你的学习路径。</p>
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      @submit.prevent="submit"
    >
      <el-form-item label="邮箱" prop="email"
        ><el-input v-model="form.email" size="large"
      /></el-form-item>
      <el-form-item label="密码" prop="password"
        ><el-input v-model="form.password" type="password" show-password size="large"
      /></el-form-item>
      <el-button
        class="full-button"
        type="primary"
        size="large"
        :loading="loading"
        native-type="submit"
        >登录</el-button
      >
    </el-form>
    <p class="auth-switch">还没有账号？<RouterLink to="/register">创建账号</RouterLink></p>
  </AuthLayout>
</template>
