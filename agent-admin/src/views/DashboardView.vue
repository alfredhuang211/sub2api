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

    <SectionPanel title="使用说明" description="Agent Admin 复用 sub2api 原系统账号体系，用于管理代理层级、客户归属、分成和结算。">
      <div class="usage-guide">
        <p class="usage-guide-intro">
          管理员登录后先完成管理员授权、代理指定和客户归属配置；代理商登录后可在工作台查看自己的代理身份、经营汇总、分成记录和结算状态。所有比例调整、客户归属和结算处理都会进入审计链路，便于后续追踪。
        </p>
        <div class="usage-guide-steps">
          <div class="usage-guide-step">
            <span>01</span>
            <div>
              <strong>确认登录身份</strong>
              <p>使用原系统账号密码登录。管理员可进入代理管理和系统设置，代理商只能查看自己权限范围内的数据。</p>
            </div>
          </div>
          <div class="usage-guide-step">
            <span>02</span>
            <div>
              <strong>配置代理关系</strong>
              <p>管理员从原系统用户中指定代理商，设置等级、上级代理和分成比例，再按需调整客户归属。</p>
            </div>
          </div>
          <div class="usage-guide-step">
            <span>03</span>
            <div>
              <strong>跟踪分成结算</strong>
              <p>系统按套餐周期确认收入生成分成，结算前会扣除退款或冲正金额，可在分成记录和结算记录中核对。</p>
            </div>
          </div>
        </div>
      </div>
    </SectionPanel>

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
