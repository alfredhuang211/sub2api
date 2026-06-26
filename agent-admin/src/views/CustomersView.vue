<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import {
  assignCustomer,
  listAgentCustomers,
  listAgents,
  listCustomerRelationChanges,
  searchUsers,
  type AgentCustomer,
  type AgentCustomerRelationChange,
  type AgentProfile,
  type UserOption
} from '@/api/agents'
import { formatDateTime, formatMinorMoney } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const agents = ref<AgentProfile[]>([])
const selectedAgentId = ref<number | null>(null)
const customers = ref<AgentCustomer[]>([])
const changes = ref<AgentCustomerRelationChange[]>([])
const customerSearch = ref('')
const customerOptions = ref<UserOption[]>([])
const customerSearchLoading = ref(false)

const form = reactive({
  customer_user_id: '',
  agent_id: '',
  reason: ''
})

onMounted(async () => {
  await loadAgents()
  await loadCustomers()
  await loadChanges()
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

async function loadChanges() {
  const page = await listCustomerRelationChanges({ page_size: 30 })
  changes.value = page.items ?? []
}

async function submitAssign() {
  await assignCustomer({
    customer_user_id: Number(form.customer_user_id),
    agent_id: Number(form.agent_id),
    reason: form.reason.trim()
  })
  form.customer_user_id = ''
  customerSearch.value = ''
  customerOptions.value = []
  form.reason = ''
  await loadCustomers()
  await loadChanges()
}

async function searchCustomers() {
  const keyword = customerSearch.value.trim()
  if (!keyword) {
    customerOptions.value = []
    return
  }
  customerSearchLoading.value = true
  error.value = ''
  try {
    const page = await searchUsers({ search: keyword, page_size: 20 })
    customerOptions.value = page.items ?? []
    if (!customerOptions.value.some((item) => String(item.id) === form.customer_user_id)) {
      form.customer_user_id = ''
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '搜索客户失败'
  } finally {
    customerSearchLoading.value = false
  }
}

function selectedCustomerLabel() {
  const selected = customerOptions.value.find((item) => String(item.id) === form.customer_user_id)
  if (!selected) return ''
  return `${selected.email} / ${selected.username || '-'}`
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="手动归属客户" description="管理员调整原因必填，默认从下一个完整套餐周期生效">
      <form class="form-grid" @submit.prevent="submitAssign">
        <label class="wide">
          <span>客户账号</span>
          <div class="inline-field">
            <input v-model.trim="customerSearch" class="input" type="search" placeholder="输入邮箱或用户名搜索" @keyup.enter.prevent="searchCustomers" />
            <button class="secondary-button" type="button" :disabled="customerSearchLoading" @click="searchCustomers">
              {{ customerSearchLoading ? '搜索中' : '搜索' }}
            </button>
          </div>
        </label>
        <label class="wide">
          <span>选择客户</span>
          <select v-model="form.customer_user_id" class="input" required>
            <option value="" disabled>请选择客户</option>
            <option v-for="user in customerOptions" :key="user.id" :value="String(user.id)">
              {{ user.email }} / {{ user.username || '-' }}
            </option>
          </select>
          <small v-if="form.customer_user_id" class="field-hint">已选择：{{ selectedCustomerLabel() }}</small>
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

    <SectionPanel title="归属变更记录" description="记录管理员手动调整原因和实际生效时间">
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>客户</th>
              <th>原代理</th>
              <th>目标代理</th>
              <th>生效时间</th>
              <th>操作人</th>
              <th>原因</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="change in changes" :key="change.id">
              <td>
                <strong>{{ change.customer_email }}</strong>
                <small>ID {{ change.customer_user_id }}</small>
              </td>
              <td>{{ change.from_agent_email || '-' }}</td>
              <td>{{ change.to_agent_email || '-' }}</td>
              <td>{{ formatDateTime(change.effective_at) }}</td>
              <td>{{ change.operator_email || '-' }}</td>
              <td>{{ change.reason }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!changes.length && !loading" />
    </SectionPanel>
  </div>
</template>
