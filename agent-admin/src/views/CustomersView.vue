<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import { assignCustomer, listAgentCustomers, listAgents, type AgentCustomer, type AgentProfile } from '@/api/agents'
import { formatDateTime, formatMinorMoney } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const agents = ref<AgentProfile[]>([])
const selectedAgentId = ref<number | null>(null)
const customers = ref<AgentCustomer[]>([])

const form = reactive({
  customer_user_id: '',
  agent_id: '',
  reason: ''
})

onMounted(async () => {
  await loadAgents()
  await loadCustomers()
})

async function loadAgents() {
  const page = await listAgents({ page_size: 100 })
  agents.value = page.items ?? []
  selectedAgentId.value = selectedAgentId.value ?? agents.value[0]?.id ?? null
}

async function loadCustomers() {
  if (!selectedAgentId.value) return
  loading.value = true
  error.value = ''
  try {
    const page = await listAgentCustomers(selectedAgentId.value)
    customers.value = page.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载客户归属失败'
  } finally {
    loading.value = false
  }
}

async function submitAssign() {
  await assignCustomer({
    customer_user_id: Number(form.customer_user_id),
    agent_id: Number(form.agent_id),
    reason: form.reason.trim()
  })
  form.customer_user_id = ''
  form.reason = ''
  await loadCustomers()
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="手动归属客户" description="管理员调整原因必填，默认从下一个完整套餐周期生效">
      <form class="form-grid" @submit.prevent="submitAssign">
        <label>
          <span>客户用户 ID</span>
          <input v-model="form.customer_user_id" class="input" type="number" min="1" required />
        </label>
        <label>
          <span>目标代理</span>
          <select v-model="form.agent_id" class="input" required>
            <option value="" disabled>选择代理</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.email }} / {{ agent.level }} 级
            </option>
          </select>
        </label>
        <label class="wide">
          <span>调整原因</span>
          <input v-model="form.reason" class="input" required placeholder="必填，用于审计" />
        </label>
        <button class="primary-button" type="submit">保存归属</button>
      </form>
    </SectionPanel>

    <SectionPanel title="客户列表" description="客户同一时间只能归属一个代理">
      <template #actions>
        <select v-model.number="selectedAgentId" class="input compact-input" @change="loadCustomers">
          <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.email }}</option>
        </select>
      </template>

      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>客户</th>
              <th>来源</th>
              <th>套餐</th>
              <th>周期结束</th>
              <th>周期确认收入</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="customer in customers" :key="customer.id">
              <td>
                <strong>{{ customer.email }}</strong>
                <small>{{ customer.username }}</small>
              </td>
              <td>{{ customer.source === 'manual' ? '管理员归属' : '自然推荐' }}</td>
              <td>{{ customer.subscription_name || '-' }}</td>
              <td>{{ formatDateTime(customer.period_end_at) }}</td>
              <td>{{ formatMinorMoney(customer.confirmed_revenue) }}</td>
              <td>{{ customer.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!customers.length && !loading" />
    </SectionPanel>
  </div>
</template>
