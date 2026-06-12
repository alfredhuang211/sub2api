<script setup lang="ts">
import { onMounted, ref } from 'vue'
import MetricCard from '@/components/MetricCard.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import EmptyState from '@/components/EmptyState.vue'
import { getAdminAgentSummary, getMyAgentDashboard, getMyAgentProfile, type AgentProfile, type AgentSummary } from '@/api/agents'
import { formatMinorMoney } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const adminSummary = ref<AgentSummary | null>(null)
const mySummary = ref<AgentSummary | null>(null)
const myProfile = ref<AgentProfile | null>(null)

onMounted(loadDashboard)

async function loadDashboard() {
  loading.value = true
  error.value = ''

  const results = await Promise.allSettled([
    getAdminAgentSummary(),
    getMyAgentDashboard(),
    getMyAgentProfile()
  ])

  if (results[0].status === 'fulfilled') adminSummary.value = results[0].value
  if (results[1].status === 'fulfilled') mySummary.value = results[1].value
  if (results[2].status === 'fulfilled') myProfile.value = results[2].value

  if (results.every((result) => result.status === 'rejected')) {
    const firstRejected = results.find((result) => result.status === 'rejected')
    const reason = firstRejected?.status === 'rejected' ? (firstRejected.reason as { message?: string }) : null
    error.value = reason?.message || '无法加载工作台数据，请确认后端代理 API 已启用'
  }

  loading.value = false
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <section class="metrics-grid">
      <MetricCard label="代理总数" :value="String(adminSummary?.total_agents ?? '-')" detail="管理员视角" />
      <MetricCard label="有效代理" :value="String(adminSummary?.active_agents ?? '-')" detail="不含禁用代理" />
      <MetricCard label="周期确认收入" :value="formatMinorMoney(adminSummary?.confirmed_revenue)" detail="到期套餐周期" />
      <MetricCard label="可结算金额" :value="formatMinorMoney(adminSummary?.payable_amount)" detail="扣除冲正后" />
    </section>

    <div class="content-grid">
      <SectionPanel title="我的代理身份" description="代理商登录后可查看自己的等级、上级和结算概况">
        <div v-if="myProfile" class="definition-list">
          <span>账号</span><strong>{{ myProfile.email }}</strong>
          <span>等级</span><strong>{{ myProfile.level }} 级代理</strong>
          <span>上级</span><strong>{{ myProfile.parent_email || '-' }}</strong>
          <span>状态</span><strong>{{ myProfile.status === 'active' ? '启用' : '禁用' }}</strong>
        </div>
        <EmptyState v-else-if="!loading" message="当前登录用户不是代理商，或代理端 API 尚未启用" />
      </SectionPanel>

      <SectionPanel title="我的经营汇总" description="上级代理只展示下级汇总，不展示下级客户明细">
        <div class="metrics-grid compact">
          <MetricCard label="直接客户" :value="String(mySummary?.direct_customers ?? '-')" />
          <MetricCard label="下级代理" :value="String(mySummary?.child_agents ?? '-')" />
          <MetricCard label="分成金额" :value="formatMinorMoney(mySummary?.commission_amount)" />
          <MetricCard label="冲正金额" :value="formatMinorMoney(mySummary?.reversed_amount)" />
        </div>
      </SectionPanel>
    </div>
  </div>
</template>
