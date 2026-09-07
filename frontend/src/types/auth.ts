export interface User {
  id: string
  email: string
  display_name: string
  created_at: string
}

export interface AuthResult {
  user: User
  access_token: string
  expires_in: number
}

export interface LoginInput {
  email: string
  password: string
}

export interface RegisterInput extends LoginInput {
  display_name: string
}
