import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import type { ApiErrorBody, ApiEnvelope } from '@/types/api'
import type { AuthResult } from '@/types/auth'

let accessToken: string | null = null
let refreshPromise: Promise<string> | null = null
let unauthorizedHandler: (() => void) | null = null
const apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const apiClient = axios.create({
  baseURL: apiBaseURL,
  timeout: 15_000,
  withCredentials: true,
})

const refreshClient = axios.create({
  baseURL: apiBaseURL,
  timeout: 15_000,
  withCredentials: true,
})

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function setUnauthorizedHandler(handler: (() => void) | null) {
  unauthorizedHandler = handler
}

apiClient.interceptors.request.use((config) => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  config.headers['X-Request-ID'] = crypto.randomUUID()
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableRequestConfig | undefined
    const isAuthRequest = config?.url?.includes('/auth/')
    if (error.response?.status !== 401 || !config || config._retry || isAuthRequest) {
      return Promise.reject(error)
    }

    config._retry = true
    refreshPromise ??= refreshClient
      .post<ApiEnvelope<AuthResult>>('/auth/refresh')
      .then((response) => {
        const token = response.data.data.access_token
        setAccessToken(token)
        return token
      })
      .finally(() => {
        refreshPromise = null
      })

    try {
      const token = await refreshPromise
      config.headers.Authorization = `Bearer ${token}`
      return apiClient(config)
    } catch (refreshError) {
      setAccessToken(null)
      unauthorizedHandler?.()
      return Promise.reject(refreshError)
    }
  },
)

export async function unwrap<T>(request: Promise<{ data: ApiEnvelope<T> }>): Promise<T> {
  const response = await request
  return response.data.data
}

export function getApiErrorMessage(error: unknown): string {
  if (error instanceof AxiosError) {
    const body = error.response?.data as ApiErrorBody | undefined
    return body?.details?.[0]?.reason || body?.message || '请求失败，请稍后重试'
  }
  return error instanceof Error ? error.message : '未知错误'
}
