import type { AxiosAdapter } from 'axios'
import type { Component } from 'vue'
import type { LoginResponse } from './auth'

export const localDemoMode = false

export function getLocalDemoAdapter(): AxiosAdapter | undefined {
  return undefined
}

export async function loadLocalDemoLoginEntry(): Promise<Component | null> {
  return null
}

export async function getLocalDemoLoginResponse(): Promise<LoginResponse | null> {
  return null
}
