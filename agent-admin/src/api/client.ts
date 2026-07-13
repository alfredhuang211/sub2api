import axios, { type AxiosError, type AxiosResponse } from 'axios'
import type { ApiEnvelope } from '@/types'
import { getLocalDemoAdapter } from '@/api/localDemo'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  timeout: 30000,
  adapter: getLocalDemoAdapter(),
  headers: {
    'Content-Type': 'application/json'
  }
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('agent_admin_token') || localStorage.getItem('auth_token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    const envelope = response.data as ApiEnvelope<unknown>
    if (envelope && typeof envelope === 'object' && 'code' in envelope) {
      if (envelope.code === 0) {
        response.data = envelope.data
        return response
      }

      return Promise.reject({
        status: response.status,
        code: envelope.code,
        message: envelope.message || '接口请求失败'
      })
    }

    return response
  },
  (error: AxiosError<ApiEnvelope<unknown>>) => {
    if (error.response?.data?.message) {
      return Promise.reject({
        status: error.response.status,
        code: error.response.data.code,
        message: error.response.data.message
      })
    }

    return Promise.reject({
      status: error.response?.status,
      message: error.message || '网络请求失败'
    })
  }
)
