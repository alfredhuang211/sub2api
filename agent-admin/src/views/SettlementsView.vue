<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { adjustSettlement, listMySettlements, listSettlements, registerSettlementPayment, type AgentSettlement, type SettlementStatus } from '@/api/agents'
import { getCurrentUser } from '@/api/session'
import { formatDateTime, formatMinorMoney } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const settlements = ref<AgentSettlement[]>([])
const scope = ref<'admin' | 'agent'>('admin')
const isAdmin = ref(false)
const isAgent = ref(false)
const adjustForms = ref<Record<number, { reverse_amount: string; status: SettlementStatus; reason: string }>>({})
const paymentForms = ref<Record<number, { amount: string; payment_method: string; payment_reference: string; paid_at: string; remark: string }>>({})

onMounted(async () => {
  const user = await getCurrentUser()
  isAdmin.value = user.is_admin
  isAgent.value = user.is_agent
  scope.value = user.is_admin ? 'admin' : 'agent'
  await loadSettlements()
})

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
      paymentForms.value[item.id] = paymentForms.value[item.id] || {
        amount: String((item.payment_amount ?? item.net_amount) / 100),
        payment_method: item.payment_method ?? '',
        payment_reference: item.payment_reference ?? '',
        paid_at: '',
        remark: item.payment_remark ?? ''
      }
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载结算记录失败'
  } finally {
    loading.value = false
  }
}

async function submitPayment(settlement: AgentSettlement) {
  const form = paymentForms.value[settlement.id]
  const amount = Math.round(Number(form?.amount || 0) * 100)
  if (!form || amount <= 0) {
    error.value = '结算金额必须大于 0'
    return
  }
  error.value = ''
  try {
    await registerSettlementPayment(settlement.id, {
      amount,
      payment_method: form.payment_method || undefined,
      payment_reference: form.payment_reference.trim() || undefined,
      paid_at: form.paid_at ? new Date(form.paid_at).toISOString() : undefined,
      remark: form.remark.trim() || undefined
    })
    await loadSettlements()
  } catch (err) {
    error.value = (err as { message?: string }).message || '登记结算失败'
  }
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

function paymentMethodLabel(method: string) {
  const labels: Record<string, string> = {
    bank_transfer: '银行转账',
    alipay: '支付宝',
    wechat_pay: '微信支付',
    cash: '现金',
    other: '其他'
  }
  return labels[method] ?? method
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="结算记录" description="最低 100 元，冻结 5 天，自然月月底结算">
      <template #actions>
        <div v-if="isAdmin && isAgent" class="segmented">
          <button type="button" :class="{ active: scope === 'admin' }" @click="scope = 'admin'; loadSettlements()">管理员</button>
          <button type="button" :class="{ active: scope === 'agent' }" @click="scope = 'agent'; loadSettlements()">代理商</button>
        </div>
      </template>

      <div class="table-wrap">
        <table class="data-table settlements-table">
          <thead>
            <tr>
              <th>代理</th>
              <th>月份</th>
              <th>金额</th>
              <th>冲正</th>
              <th>净额</th>
              <th>冻结至</th>
              <th>结算登记</th>
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
              <td>
                <strong>{{ item.payment_amount ? formatMinorMoney(item.payment_amount) : '-' }}</strong>
                <small>{{ formatDateTime(item.paid_at) }}</small>
                <small v-if="item.payment_method">{{ paymentMethodLabel(item.payment_method) }}</small>
                <small v-if="item.payment_reference">流水号：{{ item.payment_reference }}</small>
                <small v-if="item.payment_remark">备注：{{ item.payment_remark }}</small>
                <small v-if="item.payment_operator_email">登记人：{{ item.payment_operator_email }}</small>
              </td>
              <td><StatusBadge :status="item.status" /></td>
              <td>
                <div v-if="scope === 'admin' && item.status === 'payable'" class="settlement-payment">
                  <input v-model="paymentForms[item.id].amount" class="table-input" type="number" min="0.01" step="0.01" title="结算金额" />
                  <select v-model="paymentForms[item.id].payment_method" class="table-input" title="支付方式">
                    <option value="">支付方式</option>
                    <option value="bank_transfer">银行转账</option>
                    <option value="alipay">支付宝</option>
                    <option value="wechat_pay">微信支付</option>
                    <option value="cash">现金</option>
                    <option value="other">其他</option>
                  </select>
                  <input v-model="paymentForms[item.id].paid_at" class="table-input settlement-datetime" type="datetime-local" title="支付时间" />
                  <input v-model="paymentForms[item.id].payment_reference" class="input compact-input" placeholder="流水号，可选" />
                  <input v-model="paymentForms[item.id].remark" class="input compact-input" placeholder="备注，可选" />
                  <button class="link-button" type="button" @click="submitPayment(item)">登记结算</button>
                </div>
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
