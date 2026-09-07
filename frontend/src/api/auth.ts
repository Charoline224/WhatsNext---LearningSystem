import { apiClient, unwrap } from './client'
import type { AuthResult, LoginInput, RegisterInput, User } from '@/types/auth'

export const authApi = {
  register: (input: RegisterInput) => unwrap<AuthResult>(apiClient.post('/auth/register', input)),
  login: (input: LoginInput) => unwrap<AuthResult>(apiClient.post('/auth/login', input)),
  refresh: () => unwrap<AuthResult>(apiClient.post('/auth/refresh')),
  logout: () => apiClient.post('/auth/logout'),
  me: () => unwrap<User>(apiClient.get('/users/me')),
}
