<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AuthLayout from '@/layouts/AuthLayout.vue'
import { getApiErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({ display_name: '', email: '', password: '' })
const rules: FormRules = {
  display_name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  email: [{ required: true, type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  password: [{ required: true, min: 8, message: '密码至少 8 位', trigger: 'blur' }],
}
const auth = useAuthStore()
const router = useRouter()
async function submit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  loading.value = true
  try {
    await auth.register(form)
    await router.push('/spaces')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <p class="eyebrow">开始学习</p>
    <h2>创建你的账号</h2>
    <p class="form-intro">先设定一个目标，其余的交给路径。</p>
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      @submit.prevent="submit"
    >
      <el-form-item label="昵称" prop="display_name"
        ><el-input v-model="form.display_name" size="large"
      /></el-form-item>
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
        >创建账号</el-button
      >
    </el-form>
    <p class="auth-switch">已有账号？<RouterLink to="/login">直接登录</RouterLink></p>
  </AuthLayout>
</template>
