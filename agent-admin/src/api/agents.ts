import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export type AgentLevel = 1 | 2 | 3
export type AgentStatus = 'active' | 'disabled'
export type SettlementStatus = 'pending' | 'frozen' | 'payable' | 'paid' | 'reversed'

export interface ListParams {
  page?: number
  page_size?: number
  search?: string
}

export interface AgentProfile {
  id: number
  user_id: number
  username: string
  email: string
  level: AgentLevel
  parent_agent_id?: number | null
  parent_username?: string | null
  parent_email?: string | null
  status: AgentStatus
  rate_bps: number
  children_count: number
  customers_count: number
  payable_amount: number
  frozen_amount: number
  created_at: string
}

export interface AgentSummary {
  total_agents: number
  active_agents: number
  disabled_agents: number
  direct_customers: number
  child_agents: number
  confirmed_revenue: number
  commission_amount: number
  payable_amount: number
  reversed_amount: number
}

export interface AgentCustomer {
  id: number
  user_id: number
  email: string
  username: string
  source: 'referral' | 'manual'
  source_referral_code?: string | null
  agent_id: number
  agent_name: string
  subscription_name?: string | null
  period_end_at?: string | null
  confirmed_revenue: number
  status: string
}

export interface AgentCustomerRelationChange {
  id: number
  customer_user_id: number
  customer_email: string
  from_agent_id?: number | null
  from_agent_email?: string | null
  to_agent_id?: number | null
  to_agent_email?: string | null
  reason: string
  operator_email: string
  effective_at: string
  created_at: string
}

export interface AgentCommission {
  id: number
  customer_email: string
  order_id?: number | null
  period_start_at: string
  period_end_at: string
  order_paid_amount: number
  confirmed_revenue: number
  rate_bps: number
  commission_amount: number
  reverse_amount: number
  reverse_reason_type?: string | null
  status: SettlementStatus
}

export interface AgentSettlement {
  id: number
  agent_id: number
  agent_email: string
  period_month: string
  amount: number
  reverse_amount: number
  net_amount: number
  status: SettlementStatus
  frozen_until?: string | null
  paid_at?: string | null
  payment_amount?: number | null
  payment_method?: string | null
  payment_reference?: string | null
  payment_remark?: string | null
  payment_registered_at?: string | null
  payment_operator_email?: string | null
}

export interface AgentAuditLog {
  id: number
  operator_email: string
  operator_role: 'admin' | 'agent'
  action: string
  target_type: string
  target_id: number
  reason?: string | null
  created_at: string
}

export interface AgentOrder {
  id: number
  order_no: string
  customer_email: string
  status: string
  pay_amount: number
  paid_at?: string | null
  completed_at?: string | null
}

export interface UserOption {
  id: number
  email: string
  username: string
  role: string
  status: string
}

export interface AgentAdminUser {
  id: number
  user_id: number
  email: string
  username: string
  source: 'base' | 'delegated'
  status: 'active' | 'disabled'
  created_by_email?: string | null
  created_at: string
  revoked_at?: string | null
}

export interface CreateAgentRequest {
  user_id: number
  level: AgentLevel
  parent_agent_id?: number | null
  rate_bps?: number
}

export interface UpdateAgentRequest {
  level?: AgentLevel
  parent_agent_id?: number | null
  rate_bps?: number
}

export interface ForceAdjustAgentRequest extends UpdateAgentRequest {
  reason: string
}

export interface AssignCustomerRequest {
  customer_user_id: number
  agent_id: number
  reason: string
}

export interface AdjustSettlementRequest {
  amount?: number
  reverse_amount?: number
  net_amount?: number
  status?: SettlementStatus
  payment_reference?: string
  reason: string
}

export interface RegisterSettlementPaymentRequest {
  amount: number
  payment_method?: string
  payment_reference?: string
  paid_at?: string
  remark?: string
}

export async function getAdminAgentSummary() {
  const { data } = await apiClient.get<AgentSummary>('/admin/agents/summary')
  return data
}

export async function listAgents(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentProfile>>('/admin/agents', {
    params: withDefaults(params)
  })
  return data
}

export async function searchAssignableUsers(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<UserOption>>('/admin/users/assignable', {
    params: withDefaults(params)
  })
  return data
}

export async function searchUsers(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<UserOption>>('/admin/users/search', {
    params: withDefaults(params)
  })
  return data
}

export async function listAgentAdminUsers(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentAdminUser>>('/admin/admin-users', {
    params: withDefaults(params)
  })
  return data
}

export async function searchAgentAdminCandidates(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<UserOption>>(
    '/admin/admin-users/candidates',
    { params: withDefaults(params) }
  )
  return data
}

export async function grantAgentAdmin(user_id: number) {
  const { data } = await apiClient.post<AgentAdminUser>('/admin/admin-users', { user_id })
  return data
}

export async function revokeAgentAdmin(id: number) {
  const { data } = await apiClient.post<AgentAdminUser>(`/admin/admin-users/${id}/revoke`)
  return data
}

export async function getAgent(id: number) {
  const { data } = await apiClient.get<AgentProfile>(`/admin/agents/${id}`)
  return data
}

export async function createAgent(payload: CreateAgentRequest) {
  const { data } = await apiClient.post<AgentProfile>('/admin/agents', payload)
  return data
}

export async function updateAgent(id: number, payload: UpdateAgentRequest) {
  const { data } = await apiClient.put<AgentProfile>(`/admin/agents/${id}`, payload)
  return data
}

export async function forceAdjustAgent(id: number, payload: ForceAdjustAgentRequest) {
  const { data } = await apiClient.post<AgentProfile>(`/admin/agents/${id}/force-adjust`, payload)
  return data
}

export async function disableAgent(id: number) {
  const { data } = await apiClient.post<AgentProfile>(`/admin/agents/${id}/disable`)
  return data
}

export async function restoreAgent(id: number) {
  const { data } = await apiClient.post<AgentProfile>(`/admin/agents/${id}/restore`)
  return data
}

export async function listAgentCustomers(agentId: number, params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCustomer>>(
    `/admin/agents/${agentId}/customers`,
    { params: withDefaults(params) }
  )
  return data
}

export async function listAgentChildren(agentId: number, params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentProfile>>(
    `/admin/agents/${agentId}/children`,
    { params: withDefaults(params) }
  )
  return data
}

export async function updateAgentCommissionRate(id: number, rate_bps: number) {
  const { data } = await apiClient.put<AgentProfile>(`/admin/agents/${id}/commission-rate`, {
    rate_bps
  })
  return data
}

export async function assignCustomer(payload: AssignCustomerRequest) {
  const { data } = await apiClient.post<AgentCustomer>('/admin/agent-customer-relations', payload)
  return data
}

export async function listCustomerRelationChanges(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCustomerRelationChange>>(
    '/admin/agent-customer-relations/changes',
    { params: withDefaults(params) }
  )
  return data
}

export async function listCommissions(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCommission>>(
    '/admin/agent-commissions',
    { params: withDefaults(params) }
  )
  return data
}

export async function listSettlements(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentSettlement>>(
    '/admin/agent-settlements',
    { params: withDefaults(params) }
  )
  return data
}

export async function registerSettlementPayment(
  id: number,
  payload: RegisterSettlementPaymentRequest
) {
  const { data } = await apiClient.post<AgentSettlement>(
    `/admin/agent-settlements/${id}/register-payment`,
    payload
  )
  return data
}

export async function adjustSettlement(id: number, payload: AdjustSettlementRequest) {
  const { data } = await apiClient.post<AgentSettlement>(
    `/admin/agent-settlements/${id}/adjust`,
    payload
  )
  return data
}

export async function listAuditLogs(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentAuditLog>>(
    '/admin/agent-audit-logs',
    { params: withDefaults(params) }
  )
  return data
}

export async function getMyAgentProfile() {
  const { data } = await apiClient.get<AgentProfile>('/agent/profile')
  return data
}

export async function getMyAgentDashboard() {
  const { data } = await apiClient.get<AgentSummary>('/agent/dashboard')
  return data
}

export async function listMyCustomers(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCustomer>>('/agent/customers', {
    params: withDefaults(params)
  })
  return data
}

export async function listMyInvites(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCustomer>>('/agent/invites', {
    params: withDefaults(params)
  })
  return data
}

export async function listMyDevelopableUsers(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCustomer>>(
    '/agent/developable-users',
    {
      params: withDefaults(params)
    }
  )
  return data
}

export async function listMyChildren(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentProfile>>('/agent/children', {
    params: withDefaults(params)
  })
  return data
}

export async function createMyChildAgent(payload: { user_id: number; rate_bps?: number }) {
  const { data } = await apiClient.post<AgentProfile>('/agent/children', payload)
  return data
}

export async function updateMyChildRate(id: number, rate_bps: number) {
  const { data } = await apiClient.put<AgentProfile>(`/agent/children/${id}/commission-rate`, {
    rate_bps
  })
  return data
}

export async function listMyCommissions(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentCommission>>('/agent/commissions', {
    params: withDefaults(params)
  })
  return data
}

export async function listMySettlements(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentSettlement>>('/agent/settlements', {
    params: withDefaults(params)
  })
  return data
}

export async function getMyUpline() {
  const { data } = await apiClient.get<AgentProfile | null>('/agent/upline')
  return data
}

export async function listMyOrders(params: ListParams = {}) {
  const { data } = await apiClient.get<PaginatedResponse<AgentOrder>>('/agent/orders', {
    params: withDefaults(params)
  })
  return data
}

function withDefaults(params: ListParams) {
  return {
    page: params.page ?? 1,
    page_size: params.page_size ?? 20,
    search: params.search ?? '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
  }
}
