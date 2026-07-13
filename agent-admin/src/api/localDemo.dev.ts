import type { AxiosAdapter } from 'axios'
import type { Component } from 'vue'
import type { LoginResponse } from './auth'
import LocalDemoLoginEntry from '@/components/LocalDemoLoginEntry.vue'
import { createDemoAdapter, demoLoginResponse } from './mock'

export const localDemoMode =
  import.meta.env.DEV && import.meta.env.VITE_AGENT_ADMIN_DEMO === 'true'

export function getLocalDemoAdapter(): AxiosAdapter | undefined {
  return localDemoMode ? createDemoAdapter() : undefined
}

export async function loadLocalDemoLoginEntry(): Promise<Component | null> {
  return localDemoMode ? LocalDemoLoginEntry : null
}

export async function getLocalDemoLoginResponse(): Promise<LoginResponse | null> {
  return localDemoMode ? demoLoginResponse() : null
}
