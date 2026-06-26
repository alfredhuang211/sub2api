import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import AgentsView from './views/AgentsView.vue'
import CustomersView from './views/CustomersView.vue'
import CommissionsView from './views/CommissionsView.vue'
import SettlementsView from './views/SettlementsView.vue'
import AuditLogsView from './views/AuditLogsView.vue'
import LoginView from './views/LoginView.vue'
import AgentOperationsView from './views/AgentOperationsView.vue'
import SettingsView from './views/SettingsView.vue'
import { hasAuthToken } from './api/auth'
import { getCurrentUser } from './api/session'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/login', component: LoginView, meta: { title: '登录', public: true } },
    { path: '/dashboard', component: DashboardView, meta: { title: '工作台', roles: ['admin', 'agent'] } },
    { path: '/agents', component: AgentsView, meta: { title: '代理管理', roles: ['admin'] } },
    { path: '/agent-operations', component: AgentOperationsView, meta: { title: '代理经营', roles: ['agent'] } },
    { path: '/customers', component: CustomersView, meta: { title: '客户归属', roles: ['admin'] } },
    { path: '/commissions', component: CommissionsView, meta: { title: '分成记录', roles: ['admin', 'agent'] } },
    { path: '/settlements', component: SettlementsView, meta: { title: '结算记录', roles: ['admin', 'agent'] } },
    { path: '/audit-logs', component: AuditLogsView, meta: { title: '审计日志', roles: ['admin'] } },
    { path: '/settings', component: SettingsView, meta: { title: '系统设置', roles: ['admin'] } }
  ]
})

router.beforeEach(async (to) => {
  if (!to.meta.public && !hasAuthToken()) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login' && hasAuthToken()) {
    return { path: '/dashboard' }
  }
  if (to.meta.public) return true

  const requiredRoles = Array.isArray(to.meta.roles) ? to.meta.roles : []
  if (!requiredRoles.length) return true

  try {
    const user = await getCurrentUser()
    const allowed =
      (requiredRoles.includes('admin') && user.is_admin) ||
      (requiredRoles.includes('agent') && user.is_agent)
    if (!allowed) {
      if (to.path !== '/dashboard') return { path: '/dashboard' }
      return true
    }
  } catch {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  return true
})

router.afterEach((to) => {
  document.title = `${String(to.meta.title || '工作台')} - Agent Admin`
})

export default router
