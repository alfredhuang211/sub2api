import { apiClient } from './client'

export interface LoginRequest {
  email: string
  password: string
  turnstile_token?: string
}

export interface LoginResponse {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  token_type?: string
  requires_2fa?: boolean
  temp_token?: string
  user_email_masked?: string
  user?: {
    id: number
    email: string
    username?: string
    role?: string
  }
}

export async function login(payload: LoginRequest) {
  const { data } = await apiClient.post<LoginResponse>('/auth/login', payload)
  return data
}

export async function login2FA(payload: { temp_token: string; totp_code: string }) {
  const { data } = await apiClient.post<LoginResponse>('/auth/login/2fa', payload)
  return data
}

export function saveAuthTokens(response: LoginResponse) {
  if (response.access_token) {
    localStorage.setItem('agent_admin_token', response.access_token)
  }
  if (response.refresh_token) {
    localStorage.setItem('agent_admin_refresh_token', response.refresh_token)
  }
}

export function clearAuthTokens() {
  localStorage.removeItem('agent_admin_token')
  localStorage.removeItem('agent_admin_refresh_token')
}

export function hasAuthToken() {
  return Boolean(localStorage.getItem('agent_admin_token') || localStorage.getItem('auth_token'))
}
