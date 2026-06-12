import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import AgentsView from './views/AgentsView.vue'
import CustomersView from './views/CustomersView.vue'
import CommissionsView from './views/CommissionsView.vue'
import SettlementsView from './views/SettlementsView.vue'
import AuditLogsView from './views/AuditLogsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', component: DashboardView, meta: { title: '工作台' } },
    { path: '/agents', component: AgentsView, meta: { title: '代理管理' } },
    { path: '/customers', component: CustomersView, meta: { title: '客户归属' } },
    { path: '/commissions', component: CommissionsView, meta: { title: '分成记录' } },
    { path: '/settlements', component: SettlementsView, meta: { title: '结算记录' } },
    { path: '/audit-logs', component: AuditLogsView, meta: { title: '审计日志' } }
  ]
})

router.afterEach((to) => {
  document.title = `${String(to.meta.title || '工作台')} - Agent Admin`
})

export default router
