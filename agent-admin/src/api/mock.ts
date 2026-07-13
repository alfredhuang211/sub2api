import type { AxiosAdapter, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import type { LoginResponse } from './auth'
import type { CurrentUser } from './session'
import type { AgentAdminSettings, PublicSettings } from './settings'
import type {
  AgentAdminUser,
  AgentAuditLog,
  AgentCommission,
  AgentCustomer,
  AgentCustomerRelationChange,
  AgentOrder,
  AgentProfile,
  AgentSettlement,
  AgentSummary,
  CreateAgentRequest,
  UserOption
} from './agents'
import type { PaginatedResponse } from '@/types'

export const DEMO_TOKEN = 'agent-admin-demo-token'

const now = new Date()
const iso = (daysOffset: number) => {
  const value = new Date(now)
  value.setDate(value.getDate() + daysOffset)
  return value.toISOString()
}

const demoAgents: AgentProfile[] = [
  {
    id: 1,
    user_id: 101,
    username: 'demo_admin',
    email: 'demo-admin@sub2api.local',
    level: 1,
    parent_agent_id: null,
    parent_username: null,
    parent_email: null,
    status: 'active',
    rate_bps: 2000,
    children_count: 1,
    customers_count: 2,
    payable_amount: 54000,
    frozen_amount: 12000,
    created_at: iso(-45)
  },
  {
    id: 2,
    user_id: 102,
    username: 'east_partner',
    email: 'east-partner@sub2api.local',
    level: 2,
    parent_agent_id: 1,
    parent_username: 'demo_admin',
    parent_email: 'demo-admin@sub2api.local',
    status: 'active',
    rate_bps: 1500,
    children_count: 1,
    customers_count: 1,
    payable_amount: 18600,
    frozen_amount: 5200,
    created_at: iso(-30)
  },
  {
    id: 3,
    user_id: 103,
    username: 'studio_agent',
    email: 'studio-agent@sub2api.local',
    level: 3,
    parent_agent_id: 2,
    parent_username: 'east_partner',
    parent_email: 'east-partner@sub2api.local',
    status: 'disabled',
    rate_bps: 1000,
    children_count: 0,
    customers_count: 1,
    payable_amount: 0,
    frozen_amount: 2600,
    created_at: iso(-12)
  }
]

const assignableUsers: UserOption[] = [
  { id: 201, email: 'customer-a@sub2api.local', username: 'customer_a', role: 'user', status: 'active' },
  { id: 202, email: 'customer-b@sub2api.local', username: 'customer_b', role: 'user', status: 'active' },
  { id: 203, email: 'reseller-c@sub2api.local', username: 'reseller_c', role: 'user', status: 'active' },
  { id: 204, email: 'ops-user@sub2api.local', username: 'ops_user', role: 'user', status: 'active' }
]

const demoCustomers: AgentCustomer[] = [
  {
    id: 1,
    user_id: 201,
    email: 'customer-a@sub2api.local',
    username: 'customer_a',
    source: 'referral',
    source_referral_code: 'S2DEMO',
    agent_id: 1,
    agent_name: 'demo_admin',
    subscription_name: 'Pro 月付',
    period_end_at: iso(6),
    confirmed_revenue: 29800,
    status: 'active'
  },
  {
    id: 2,
    user_id: 202,
    email: 'customer-b@sub2api.local',
    username: 'customer_b',
    source: 'manual',
    source_referral_code: null,
    agent_id: 1,
    agent_name: 'demo_admin',
    subscription_name: 'Team 月付',
    period_end_at: iso(14),
    confirmed_revenue: 59800,
    status: 'active'
  },
  {
    id: 3,
    user_id: 203,
    email: 'reseller-c@sub2api.local',
    username: 'reseller_c',
    source: 'referral',
    source_referral_code: 'EAST15',
    agent_id: 2,
    agent_name: 'east_partner',
    subscription_name: 'Starter 月付',
    period_end_at: iso(3),
    confirmed_revenue: 9900,
    status: 'active'
  }
]

const demoInvites: AgentCustomer[] = [
  {
    id: 4,
    user_id: 204,
    email: 'ops-user@sub2api.local',
    username: 'ops_user',
    source: 'referral',
    source_referral_code: 'S2DEMO',
    agent_id: 1,
    agent_name: 'demo_admin',
    subscription_name: '未订阅',
    period_end_at: null,
    confirmed_revenue: 0,
    status: 'active'
  }
]

let settings: AgentAdminSettings = {
  turnstile_enabled: false,
  turnstile_site_key: '',
  updated_at: iso(-1)
}

let adminUsers: AgentAdminUser[] = [
  {
    id: 1,
    user_id: 101,
    email: 'demo-admin@sub2api.local',
    username: 'demo_admin',
    source: 'base',
    status: 'active',
    created_by_email: null,
    created_at: iso(-90),
    revoked_at: null
  },
  {
    id: 2,
    user_id: 104,
    email: 'finance-admin@sub2api.local',
    username: 'finance_admin',
    source: 'delegated',
    status: 'active',
    created_by_email: 'demo-admin@sub2api.local',
    created_at: iso(-20),
    revoked_at: null
  }
]

let relationChanges: AgentCustomerRelationChange[] = [
  {
    id: 1,
    customer_user_id: 202,
    customer_email: 'customer-b@sub2api.local',
    from_agent_id: null,
    from_agent_email: null,
    to_agent_id: 1,
    to_agent_email: 'demo-admin@sub2api.local',
    reason: '线下销售归属补录',
    operator_email: 'demo-admin@sub2api.local',
    effective_at: iso(-5),
    created_at: iso(-6)
  }
]

let commissions: AgentCommission[] = [
  {
    id: 1,
    customer_email: 'customer-a@sub2api.local',
    order_id: 9001,
    period_start_at: iso(-35),
    period_end_at: iso(-5),
    order_paid_amount: 29800,
    confirmed_revenue: 29800,
    rate_bps: 2000,
    commission_amount: 5960,
    reverse_amount: 0,
    reverse_reason_type: null,
    status: 'payable'
  },
  {
    id: 2,
    customer_email: 'customer-b@sub2api.local',
    order_id: 9002,
    period_start_at: iso(-32),
    period_end_at: iso(-2),
    order_paid_amount: 59800,
    confirmed_revenue: 59800,
    rate_bps: 2000,
    commission_amount: 11960,
    reverse_amount: 1200,
    reverse_reason_type: '退款冲正',
    status: 'frozen'
  },
  {
    id: 3,
    customer_email: 'reseller-c@sub2api.local',
    order_id: 9003,
    period_start_at: iso(-28),
    period_end_at: iso(2),
    order_paid_amount: 9900,
    confirmed_revenue: 9900,
    rate_bps: 1500,
    commission_amount: 1485,
    reverse_amount: 0,
    reverse_reason_type: null,
    status: 'pending'
  }
]

let settlements: AgentSettlement[] = [
  {
    id: 1,
    agent_id: 1,
    agent_email: 'demo-admin@sub2api.local',
    period_month: '2026-07',
    amount: 59600,
    reverse_amount: 5600,
    net_amount: 54000,
    status: 'payable',
    frozen_until: iso(-1),
    paid_at: null,
    payment_amount: null,
    payment_method: null,
    payment_reference: null,
    payment_remark: null,
    payment_registered_at: null,
    payment_operator_email: null
  },
  {
    id: 2,
    agent_id: 2,
    agent_email: 'east-partner@sub2api.local',
    period_month: '2026-07',
    amount: 18600,
    reverse_amount: 0,
    net_amount: 18600,
    status: 'frozen',
    frozen_until: iso(3),
    paid_at: null,
    payment_amount: null,
    payment_method: null,
    payment_reference: null,
    payment_remark: null,
    payment_registered_at: null,
    payment_operator_email: null
  }
]

let auditLogs: AgentAuditLog[] = [
  {
    id: 1,
    operator_email: 'demo-admin@sub2api.local',
    operator_role: 'admin',
    action: 'assign_customer',
    target_type: 'agent_customer_relation',
    target_id: 2,
    reason: '线下销售归属补录',
    created_at: iso(-6)
  },
  {
    id: 2,
    operator_email: 'demo-admin@sub2api.local',
    operator_role: 'admin',
    action: 'update_commission_rate',
    target_type: 'agent_profile',
    target_id: 2,
    reason: '季度合作政策调整',
    created_at: iso(-10)
  }
]

const orders: AgentOrder[] = [
  {
    id: 9001,
    order_no: 'DEMO-202607-001',
    customer_email: 'customer-a@sub2api.local',
    status: 'paid',
    pay_amount: 29800,
    paid_at: iso(-7),
    completed_at: iso(-5)
  },
  {
    id: 9002,
    order_no: 'DEMO-202607-002',
    customer_email: 'customer-b@sub2api.local',
    status: 'paid',
    pay_amount: 59800,
    paid_at: iso(-3),
    completed_at: iso(-2)
  }
]

const currentUser = (): CurrentUser => ({
  id: 101,
  email: 'demo-admin@sub2api.local',
  username: 'demo_admin',
  role: 'admin',
  status: 'active',
  is_base_admin: true,
  is_admin: true,
  is_agent: true,
  agent: demoAgents[0]
})

export function createDemoAdapter(): AxiosAdapter {
  return async (config) => demoResponse(config, routeDemoRequest(config))
}

export function demoLoginResponse(): LoginResponse {
  return {
    access_token: DEMO_TOKEN,
    refresh_token: `${DEMO_TOKEN}-refresh`,
    expires_in: 86400,
    token_type: 'Bearer',
    user: {
      id: 101,
      email: 'demo-admin@sub2api.local',
      username: 'demo_admin',
      role: 'admin'
    }
  }
}

function routeDemoRequest(config: InternalAxiosRequestConfig) {
  const method = String(config.method || 'get').toLowerCase()
  const path = normalizePath(config.url || '')
  const params = (config.params || {}) as Record<string, unknown>
  const payload = parsePayload(config.data)

  if (method === 'post' && path === '/auth/login') {
    return demoLoginResponse()
  }
  if (method === 'post' && path === '/auth/login/2fa') {
    return demoLoginResponse()
  }
  if (method === 'get' && path === '/me') return currentUser()
  if (method === 'get' && path === '/settings/public') {
    return {
      turnstile_enabled: false,
      turnstile_site_key: ''
    } satisfies PublicSettings
  }
  if (method === 'get' && path === '/admin/settings') return settings
  if (method === 'put' && path === '/admin/settings') {
    settings = { ...settings, ...payload, updated_at: new Date().toISOString() }
    return settings
  }

  if (method === 'get' && path === '/admin/agents/summary') return adminSummary()
  if (method === 'get' && path === '/admin/agents') return paginate(search(demoAgents, params), params)
  if (method === 'post' && path === '/admin/agents') return createDemoAgent(payload as CreateAgentRequest)
  if (method === 'get' && path === '/admin/users/assignable') return paginate(search(assignableUsers, params), params)
  if (method === 'get' && path === '/admin/users/search') return paginate(search(assignableUsers, params), params)
  if (method === 'get' && path === '/admin/admin-users') return paginate(search(adminUsers, params), params)
  if (method === 'get' && path === '/admin/admin-users/candidates') {
    return paginate(search(assignableUsers, params), params)
  }
  if (method === 'post' && path === '/admin/admin-users') return grantDemoAdmin(Number(payload.user_id))

  const revokeAdminMatch = path.match(/^\/admin\/admin-users\/(\d+)\/revoke$/)
  if (method === 'post' && revokeAdminMatch) return revokeDemoAdmin(Number(revokeAdminMatch[1]))

  const agentCustomersMatch = path.match(/^\/admin\/agents\/(\d+)\/customers$/)
  if (method === 'get' && agentCustomersMatch) {
    return paginate(
      demoCustomers.filter((customer) => customer.agent_id === Number(agentCustomersMatch[1])),
      params
    )
  }
  const agentChildrenMatch = path.match(/^\/admin\/agents\/(\d+)\/children$/)
  if (method === 'get' && agentChildrenMatch) {
    return paginate(
      demoAgents.filter((agent) => agent.parent_agent_id === Number(agentChildrenMatch[1])),
      params
    )
  }
  const agentRateMatch = path.match(/^\/admin\/agents\/(\d+)\/commission-rate$/)
  if (method === 'put' && agentRateMatch) return updateDemoAgent(Number(agentRateMatch[1]), payload)
  const agentForceMatch = path.match(/^\/admin\/agents\/(\d+)\/force-adjust$/)
  if (method === 'post' && agentForceMatch) {
    appendAuditLog('force_adjust_agent', 'agent_profile', Number(agentForceMatch[1]), payload.reason)
    return updateDemoAgent(Number(agentForceMatch[1]), payload)
  }
  const agentDisableMatch = path.match(/^\/admin\/agents\/(\d+)\/disable$/)
  if (method === 'post' && agentDisableMatch) return updateDemoAgent(Number(agentDisableMatch[1]), { status: 'disabled' })
  const agentRestoreMatch = path.match(/^\/admin\/agents\/(\d+)\/restore$/)
  if (method === 'post' && agentRestoreMatch) return updateDemoAgent(Number(agentRestoreMatch[1]), { status: 'active' })
  const agentMatch = path.match(/^\/admin\/agents\/(\d+)$/)
  if (agentMatch) {
    if (method === 'get') return demoAgents.find((agent) => agent.id === Number(agentMatch[1])) ?? null
    if (method === 'put') return updateDemoAgent(Number(agentMatch[1]), payload)
  }

  if (method === 'post' && path === '/admin/agent-customer-relations') {
    return assignDemoCustomer(Number(payload.customer_user_id), Number(payload.agent_id), String(payload.reason || '演示归属'))
  }
  if (method === 'get' && path === '/admin/agent-customer-relations/changes') {
    return paginate(relationChanges, params)
  }
  if (method === 'get' && path === '/admin/agent-commissions') return paginate(commissions, params)
  if (method === 'get' && path === '/admin/agent-settlements') return paginate(settlements, params)
  if (method === 'get' && path === '/admin/agent-audit-logs') return paginate(auditLogs, params)

  const settlementPaymentMatch = path.match(/^\/admin\/agent-settlements\/(\d+)\/register-payment$/)
  if (method === 'post' && settlementPaymentMatch) {
    return updateSettlement(Number(settlementPaymentMatch[1]), {
      status: 'paid',
      payment_amount: Number(payload.amount || 0),
      payment_method: payload.payment_method || 'bank_transfer',
      payment_reference: payload.payment_reference || 'DEMO-PAYMENT',
      payment_remark: payload.remark || '',
      paid_at: payload.paid_at || new Date().toISOString(),
      payment_registered_at: new Date().toISOString(),
      payment_operator_email: 'demo-admin@sub2api.local'
    })
  }
  const settlementAdjustMatch = path.match(/^\/admin\/agent-settlements\/(\d+)\/adjust$/)
  if (method === 'post' && settlementAdjustMatch) {
    appendAuditLog('adjust_settlement', 'agent_settlement', Number(settlementAdjustMatch[1]), payload.reason)
    return updateSettlement(Number(settlementAdjustMatch[1]), payload)
  }

  if (method === 'get' && path === '/agent/profile') return demoAgents[0]
  if (method === 'get' && path === '/agent/dashboard') return mySummary()
  if (method === 'get' && path === '/agent/customers') {
    return paginate(demoCustomers.filter((customer) => customer.agent_id === 1), params)
  }
  if (method === 'get' && (path === '/agent/developable-users' || path === '/agent/invites')) {
    return paginate(developableUsers(), params)
  }
  if (method === 'get' && path === '/agent/children') {
    return paginate(demoAgents.filter((agent) => agent.parent_agent_id === 1), params)
  }
  if (method === 'post' && path === '/agent/children') return createDemoAgent({ ...payload, level: 2, parent_agent_id: 1 })

  const childRateMatch = path.match(/^\/agent\/children\/(\d+)\/commission-rate$/)
  if (method === 'put' && childRateMatch) return updateDemoAgent(Number(childRateMatch[1]), payload)

  if (method === 'get' && path === '/agent/commissions') {
    return paginate(commissions.filter((commission) => commission.id !== 3), params)
  }
  if (method === 'get' && path === '/agent/settlements') {
    return paginate(settlements.filter((settlement) => settlement.agent_id === 1), params)
  }
  if (method === 'get' && path === '/agent/upline') return null
  if (method === 'get' && path === '/agent/orders') return paginate(orders, params)

  return fallbackResult(method, path, params)
}

function demoResponse<T>(config: InternalAxiosRequestConfig, data: T): AxiosResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
    request: null
  }
}

function normalizePath(url: string) {
  const parsed = new URL(url, 'http://agent-admin-demo.local')
  return parsed.pathname.replace(/^\/api\/v1/, '') || '/'
}

function parsePayload(data: unknown): Record<string, any> {
  if (!data) return {}
  if (typeof data === 'string') {
    try {
      return JSON.parse(data)
    } catch {
      return {}
    }
  }
  if (typeof data === 'object') return data as Record<string, any>
  return {}
}

function paginate<T>(items: T[], params: Record<string, unknown>): PaginatedResponse<T> {
  const page = Number(params.page || 1)
  const pageSize = Number(params.page_size || 20)
  const start = (page - 1) * pageSize
  return {
    items: items.slice(start, start + pageSize),
    total: items.length,
    page,
    page_size: pageSize
  }
}

function search<T extends object>(items: T[], params: Record<string, unknown>) {
  const keyword = String(params.search || '').trim().toLowerCase()
  if (!keyword) return items
  return items.filter((item) =>
    Object.values(item as Record<string, unknown>).some((value) =>
      String(value ?? '').toLowerCase().includes(keyword)
    )
  )
}

function adminSummary(): AgentSummary {
  return {
    total_agents: demoAgents.length,
    active_agents: demoAgents.filter((agent) => agent.status === 'active').length,
    disabled_agents: demoAgents.filter((agent) => agent.status === 'disabled').length,
    direct_customers: demoCustomers.length,
    child_agents: demoAgents.filter((agent) => agent.parent_agent_id).length,
    confirmed_revenue: demoCustomers.reduce((sum, customer) => sum + customer.confirmed_revenue, 0),
    commission_amount: commissions.reduce((sum, commission) => sum + commission.commission_amount, 0),
    payable_amount: settlements.reduce((sum, settlement) => sum + settlement.net_amount, 0),
    reversed_amount: settlements.reduce((sum, settlement) => sum + settlement.reverse_amount, 0)
  }
}

function mySummary(): AgentSummary {
  const ownSettlements = settlements.filter((settlement) => settlement.agent_id === 1)
  return {
    total_agents: demoAgents.length,
    active_agents: demoAgents.filter((agent) => agent.status === 'active').length,
    disabled_agents: demoAgents.filter((agent) => agent.status === 'disabled').length,
    direct_customers: demoCustomers.filter((customer) => customer.agent_id === 1).length,
    child_agents: demoAgents.filter((agent) => agent.parent_agent_id === 1).length,
    confirmed_revenue: demoCustomers
      .filter((customer) => customer.agent_id === 1)
      .reduce((sum, customer) => sum + customer.confirmed_revenue, 0),
    commission_amount: commissions
      .filter((commission) => commission.id !== 3)
      .reduce((sum, commission) => sum + commission.commission_amount, 0),
    payable_amount: ownSettlements.reduce((sum, settlement) => sum + settlement.net_amount, 0),
    reversed_amount: ownSettlements.reduce((sum, settlement) => sum + settlement.reverse_amount, 0)
  }
}

function developableUsers() {
  const existingAgentUserIds = new Set(demoAgents.map((agent) => agent.user_id))
  return [...demoCustomers.filter((customer) => customer.agent_id === 1), ...demoInvites].filter(
    (customer) => !existingAgentUserIds.has(customer.user_id)
  )
}

function createDemoAgent(payload: Partial<CreateAgentRequest>) {
  const user = assignableUsers.find((item) => item.id === Number(payload.user_id)) ?? assignableUsers[0]
  const level = Number(payload.level || 1) as 1 | 2 | 3
  const parent = payload.parent_agent_id
    ? demoAgents.find((agent) => agent.id === Number(payload.parent_agent_id))
    : null
  const agent: AgentProfile = {
    id: Math.max(...demoAgents.map((item) => item.id)) + 1,
    user_id: user.id,
    username: user.username,
    email: user.email,
    level,
    parent_agent_id: parent?.id ?? null,
    parent_username: parent?.username ?? null,
    parent_email: parent?.email ?? null,
    status: 'active',
    rate_bps: Number(payload.rate_bps ?? (level === 1 ? 2000 : level === 2 ? 1500 : 1000)),
    children_count: 0,
    customers_count: 0,
    payable_amount: 0,
    frozen_amount: 0,
    created_at: new Date().toISOString()
  }
  demoAgents.push(agent)
  appendAuditLog('create_agent', 'agent_profile', agent.id, '本地演示创建')
  return agent
}

function updateDemoAgent(id: number, patch: Record<string, any>) {
  const index = demoAgents.findIndex((agent) => agent.id === id)
  if (index < 0) return null
  demoAgents[index] = { ...demoAgents[index], ...patch }
  return demoAgents[index]
}

function assignDemoCustomer(customerUserId: number, agentId: number, reason: string) {
  const user = assignableUsers.find((item) => item.id === customerUserId) ?? assignableUsers[0]
  const agent = demoAgents.find((item) => item.id === agentId) ?? demoAgents[0]
  const customer: AgentCustomer = {
    id: Math.max(...demoCustomers.map((item) => item.id)) + 1,
    user_id: user.id,
    email: user.email,
    username: user.username,
    source: 'manual',
    source_referral_code: null,
    agent_id: agent.id,
    agent_name: agent.username,
    subscription_name: '演示套餐',
    period_end_at: iso(30),
    confirmed_revenue: 0,
    status: 'active'
  }
  demoCustomers.push(customer)
  relationChanges.unshift({
    id: Math.max(...relationChanges.map((item) => item.id)) + 1,
    customer_user_id: user.id,
    customer_email: user.email,
    from_agent_id: null,
    from_agent_email: null,
    to_agent_id: agent.id,
    to_agent_email: agent.email,
    reason,
    operator_email: 'demo-admin@sub2api.local',
    effective_at: new Date().toISOString(),
    created_at: new Date().toISOString()
  })
  appendAuditLog('assign_customer', 'agent_customer_relation', customer.id, reason)
  return customer
}

function grantDemoAdmin(userId: number) {
  const user = assignableUsers.find((item) => item.id === userId) ?? assignableUsers[0]
  const admin: AgentAdminUser = {
    id: Math.max(...adminUsers.map((item) => item.id)) + 1,
    user_id: user.id,
    email: user.email,
    username: user.username,
    source: 'delegated',
    status: 'active',
    created_by_email: 'demo-admin@sub2api.local',
    created_at: new Date().toISOString(),
    revoked_at: null
  }
  adminUsers.push(admin)
  return admin
}

function revokeDemoAdmin(id: number) {
  const index = adminUsers.findIndex((admin) => admin.id === id)
  if (index < 0) return null
  adminUsers[index] = { ...adminUsers[index], status: 'disabled', revoked_at: new Date().toISOString() }
  return adminUsers[index]
}

function updateSettlement(id: number, patch: Record<string, any>) {
  const index = settlements.findIndex((settlement) => settlement.id === id)
  if (index < 0) return null
  const next = { ...settlements[index], ...patch }
  if ('reverse_amount' in patch) {
    next.net_amount = next.amount - Number(patch.reverse_amount || 0)
  }
  settlements[index] = next
  return next
}

function appendAuditLog(action: string, targetType: string, targetId: number, reason?: unknown) {
  auditLogs.unshift({
    id: Math.max(...auditLogs.map((item) => item.id)) + 1,
    operator_email: 'demo-admin@sub2api.local',
    operator_role: 'admin',
    action,
    target_type: targetType,
    target_id: targetId,
    reason: reason ? String(reason) : '本地演示操作',
    created_at: new Date().toISOString()
  })
}

function fallbackResult(method: string, path: string, params: Record<string, unknown>) {
  if (method === 'get' && path.includes('/affiliates/users')) {
    return paginate(
      assignableUsers.map((user) => ({
        user_id: user.id,
        email: user.email,
        username: user.username,
        aff_code: `DEMO${user.id}`,
        aff_code_custom: false,
        aff_rebate_rate_percent: 10,
        aff_count: 2
      })),
      params
    )
  }
  if (method === 'get') return paginate([], params)
  return {}
}
