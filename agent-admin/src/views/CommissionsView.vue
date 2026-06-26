<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { listCommissions, listMyCommissions, type AgentCommission } from '@/api/agents'
import { getCurrentUser } from '@/api/session'
import { formatDateTime, formatMinorMoney, formatPercent } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const commissions = ref<AgentCommission[]>([])
const scope = ref<'admin' | 'agent'>('admin')
const isAdmin = ref(false)
const isAgent = ref(false)

onMounted(async () => {
  const user = await getCurrentUser()
  isAdmin.value = user.is_admin
  isAgent.value = user.is_agent
  scope.value = user.is_admin ? 'admin' : 'agent'
  await loadCommissions()
})

async function loadCommissions() {
  loading.value = true
  error.value = ''
  try {
    const page = scope.value === 'admin' ? await listCommissions() : await listMyCommissions()
    commissions.value = page.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载分成记录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="分成记录" description="每日定时任务扫描已到期套餐周期后生成">
      <template #actions>
        <div v-if="isAdmin && isAgent" class="segmented">
          <button type="button" :class="{ active: scope === 'admin' }" @click="scope = 'admin'; loadCommissions()">管理员</button>
          <button type="button" :class="{ active: scope === 'agent' }" @click="scope = 'agent'; loadCommissions()">代理商</button>
        </div>
      </template>

      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>客户</th>
              <th>订单</th>
              <th>周期</th>
              <th>确认收入</th>
              <th>比例</th>
              <th>分成</th>
              <th>冲正</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in commissions" :key="item.id">
              <td>{{ item.customer_email }}</td>
              <td>#{{ item.order_id }}</td>
              <td>{{ formatDateTime(item.period_start_at) }} - {{ formatDateTime(item.period_end_at) }}</td>
              <td>{{ formatMinorMoney(item.confirmed_revenue) }}</td>
              <td>{{ formatPercent(item.rate_bps) }}</td>
              <td>{{ formatMinorMoney(item.commission_amount) }}</td>
              <td>{{ formatMinorMoney(item.reverse_amount) }} {{ item.reverse_reason_type || '' }}</td>
              <td><StatusBadge :status="item.status" /></td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!commissions.length && !loading" />
    </SectionPanel>
  </div>
</template>
