<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { adjustSettlement, listMySettlements, listSettlements, markSettlementPaid, type AgentSettlement, type SettlementStatus } from '@/api/agents'
import { formatDateTime, formatMinorMoney } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const settlements = ref<AgentSettlement[]>([])
const scope = ref<'admin' | 'agent'>('admin')
const adjustForms = ref<Record<number, { reverse_amount: string; status: SettlementStatus; reason: string }>>({})

onMounted(loadSettlements)

async function loadSettlements() {
  loading.value = true
  error.value = ''
  try {
    const page = scope.value === 'admin' ? await listSettlements() : await listMySettlements()
    settlements.value = page.items ?? []
    for (const item of settlements.value) {
      adjustForms.value[item.id] = adjustForms.value[item.id] || {
        reverse_amount: String(item.reverse_amount / 100),
        status: item.status,
        reason: ''
      }
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载结算记录失败'
  } finally {
    loading.value = false
  }
}

async function markPaid(settlement: AgentSettlement) {
  await markSettlementPaid(settlement.id)
  await loadSettlements()
}

async function submitAdjust(settlement: AgentSettlement) {
  const form = adjustForms.value[settlement.id]
  if (!form?.reason.trim()) return
  await adjustSettlement(settlement.id, {
    reverse_amount: Math.round(Number(form.reverse_amount || 0) * 100),
    status: form.status,
    reason: form.reason.trim()
  })
  form.reason = ''
  await loadSettlements()
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="结算记录" description="最低 100 元，冻结 5 天，自然月月底结算">
      <template #actions>
        <div class="segmented">
          <button type="button" :class="{ active: scope === 'admin' }" @click="scope = 'admin'; loadSettlements()">管理员</button>
          <button type="button" :class="{ active: scope === 'agent' }" @click="scope = 'agent'; loadSettlements()">代理商</button>
        </div>
      </template>

      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>代理</th>
              <th>月份</th>
              <th>金额</th>
              <th>冲正</th>
              <th>净额</th>
              <th>冻结至</th>
              <th>支付时间</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in settlements" :key="item.id">
              <td>{{ item.agent_email }}</td>
              <td>{{ item.period_month }}</td>
              <td>{{ formatMinorMoney(item.amount) }}</td>
              <td>{{ formatMinorMoney(item.reverse_amount) }}</td>
              <td>{{ formatMinorMoney(item.net_amount) }}</td>
              <td>{{ formatDateTime(item.frozen_until) }}</td>
              <td>{{ formatDateTime(item.paid_at) }}</td>
              <td><StatusBadge :status="item.status" /></td>
              <td>
                <button v-if="scope === 'admin' && item.status === 'payable'" class="link-button" type="button" @click="markPaid(item)">
                  标记已支付
                </button>
                <div v-if="scope === 'admin'" class="row-actions settlement-adjust">
                  <input v-model="adjustForms[item.id].reverse_amount" class="table-input" type="number" min="0" step="0.01" title="冲正金额" />
                  <select v-model="adjustForms[item.id].status" class="table-input" title="状态">
                    <option value="pending">待结算</option>
                    <option value="frozen">冻结中</option>
                    <option value="payable">可结算</option>
                    <option value="paid">已结算</option>
                    <option value="reversed">已冲正</option>
                  </select>
                  <input v-model="adjustForms[item.id].reason" class="input compact-input" placeholder="调整原因" />
                  <button class="link-button" type="button" @click="submitAdjust(item)">调整</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!settlements.length && !loading" />
    </SectionPanel>
  </div>
</template>
