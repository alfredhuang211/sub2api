import { apiClient } from './client'
import type { AgentProfile } from './agents'

export interface CurrentUser {
  id: number
  email: string
  username: string
  role: string
  status: string
  is_base_admin: boolean
  is_admin: boolean
  is_agent: boolean
  agent?: AgentProfile | null
}

let currentUserPromise: Promise<CurrentUser> | null = null
let currentUserCache: CurrentUser | null = null

export async function getCurrentUser(force = false) {
  if (!force && currentUserCache) return currentUserCache
  if (!force && currentUserPromise) return currentUserPromise

  currentUserPromise = apiClient
    .get<CurrentUser>('/me')
    .then(({ data }) => {
      currentUserCache = data
      return data
    })
    .finally(() => {
      currentUserPromise = null
    })

  return currentUserPromise
}

export function clearCurrentUser() {
  currentUserCache = null
  currentUserPromise = null
}
