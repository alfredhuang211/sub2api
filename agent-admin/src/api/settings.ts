import { apiClient } from './client'

export interface AgentAdminSettings {
  turnstile_enabled: boolean
  turnstile_site_key: string
  updated_at?: string
}

export interface PublicSettings {
  turnstile_enabled: boolean
  turnstile_site_key: string
}

export async function getSettings() {
  const { data } = await apiClient.get<AgentAdminSettings>('/admin/settings')
  return data
}

export async function updateSettings(payload: AgentAdminSettings) {
  const { data } = await apiClient.put<AgentAdminSettings>('/admin/settings', payload)
  return data
}

export async function getPublicSettings() {
  const { data } = await apiClient.get<PublicSettings>('/settings/public')
  return data
}
