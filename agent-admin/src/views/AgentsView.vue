<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import {
  createAgent,
  disableAgent,
  forceAdjustAgent,
  listAgents,
  restoreAgent,
  searchAssignableUsers,
  updateAgent,
  type AgentLevel,
  type AgentProfile,
  type UserOption
} from '@/api/agents'
import { formatMinorMoney, formatPercent } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const agents = ref<AgentProfile[]>([])
const parentAgents = ref<AgentProfile[]>([])
const search = ref('')
const userSearch = ref('')
const userOptions = ref<UserOption[]>([])
const userSearchLoading = ref(false)

const form = reactive({
  user_id: '',
  level: 1 as AgentLevel,
  parent_agent_id: '',
  rate_percent: ''
})

const forceReasons = reactive<Record<number, string>>({})
const rateInputs = reactive<Record<number, string>>({})
const parentAgentOptions = computed(() =>
  form.level === 1
    ? []
    : parentAgents.value.filter((agent) => agent.level === form.level - 1 && agent.status === 'active')
)

onMounted(async () => {
  await Promise.all([loadAgents(), loadParentAgents()])
})

async function loadAgents() {
  loading.value = true
  error.value = ''
  try {
    const page = await listAgents({ search: search.value })
    agents.value = page.items ?? []
    for (const agent of agents.value) {
      rateInputs[agent.id] = formatRatePercent(agent.rate_bps)
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载代理列表失败'
  } finally {
    loading.value = false
  }
}

async function loadParentAgents() {
  const page = await listAgents({ page_size: 100 })
  parentAgents.value = page.items ?? []
}

async function searchUsers() {
  const keyword = userSearch.value.trim()
  if (!keyword) {
    userOptions.value = []
    return
  }
  userSearchLoading.value = true
  error.value = ''
  try {
    const page = await searchAssignableUsers({ search: keyword, page_size: 20 })
    userOptions.value = page.items ?? []
    if (!userOptions.value.some((item) => String(item.id) === form.user_id)) {
      form.user_id = ''
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '搜索用户失败'
  } finally {
    userSearchLoading.value = false
  }
}

async function submitAgent() {
  const payload = {
    user_id: Number(form.user_id),
    level: form.level,
    parent_agent_id: form.level === 1 ? null : form.parent_agent_id ? Number(form.parent_agent_id) : null,
    rate_bps: form.rate_percent ? Math.round(Number(form.rate_percent) * 100) : undefined
  }
  await createAgent(payload)
  form.user_id = ''
  userSearch.value = ''
  userOptions.value = []
  form.parent_agent_id = ''
  form.rate_percent = ''
  await Promise.all([loadAgents(), loadParentAgents()])
}

async function saveRate(agent: AgentProfile) {
  const reason = forceReasons[agent.id]?.trim()
  const rateBps = parseRatePercent(rateInputs[agent.id])
  if (reason) {
    await forceAdjustAgent(agent.id, { rate_bps: rateBps, reason })
    forceReasons[agent.id] = ''
  } else {
    await updateAgent(agent.id, { rate_bps: rateBps })
  }
  await loadAgents()
}

function formatRatePercent(rateBps: number) {
  return (Number(rateBps || 0) / 100).toFixed(2)
}

function parseRatePercent(value: string) {
  const percent = Number(value)
  return Number.isFinite(percent) ? Math.round(percent * 100) : 0
}

async function toggleAgent(agent: AgentProfile) {
  if (agent.status === 'active') {
    await disableAgent(agent.id)
  } else {
    await restoreAgent(agent.id)
  }
  await loadAgents()
}

function selectedUserLabel() {
  const selected = userOptions.value.find((item) => String(item.id) === form.user_id)
  if (!selected) return ''
  return `${selected.email} / ${selected.username || '-'}`
}

function syncParentSelection() {
  if (form.level === 1 || !parentAgentOptions.value.some((agent) => String(agent.id) === form.parent_agent_id)) {
    form.parent_agent_id = ''
  }
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="指定代理" description="管理员从原系统用户中指定 1/2/3 级代理">
      <form class="form-grid" @submit.prevent="submitAgent">
        <label class="wide">
          <span>用户邮箱</span>
          <div class="inline-field">
            <input v-model.trim="userSearch" class="input" type="search" placeholder="输入邮箱或用户名搜索" @keyup.enter.prevent="searchUsers" />
            <button class="secondary-button" type="button" :disabled="userSearchLoading" @click="searchUsers">
              {{ userSearchLoading ? '搜索中' : '搜索' }}
            </button>
          </div>
        </label>
        <label class="wide">
          <span>选择用户</span>
          <select v-model="form.user_id" class="input" required>
            <option value="" disabled>请选择用户</option>
            <option v-for="user in userOptions" :key="user.id" :value="String(user.id)">
              {{ user.email }} / {{ user.username || '-' }}
            </option>
          </select>
          <small v-if="form.user_id" class="field-hint">已选择：{{ selectedUserLabel() }}</small>
        </label>
        <label>
          <span>代理等级</span>
          <select v-model.number="form.level" class="input" @change="syncParentSelection">
            <option :value="1">1 级代理</option>
            <option :value="2">2 级代理</option>
            <option :value="3">3 级代理</option>
          </select>
        </label>
        <label>
          <span>上级代理</span>
          <select v-model="form.parent_agent_id" class="input" :disabled="form.level === 1" :required="form.level > 1">
            <option value="">{{ form.level === 1 ? '无需上级' : '选择上级代理' }}</option>
            <option v-for="agent in parentAgentOptions" :key="agent.id" :value="String(agent.id)">
              {{ agent.email }} / {{ agent.level }} 级
            </option>
          </select>
        </label>
        <label>
          <span>比例 %</span>
          <input v-model="form.rate_percent" class="input" type="number" min="0" max="100" step="0.01" placeholder="默认比例" />
        </label>
        <button class="primary-button" type="submit">保存代理</button>
      </form>
    </SectionPanel>

    <SectionPanel title="代理列表" description="禁用代理后不再产生新分成，历史数据保留">
      <template #actions>
        <div class="toolbar inline">
          <input v-model="search" class="search-input" type="search" placeholder="搜索邮箱或用户名" @keyup.enter="loadAgents" />
          <button class="secondary-button" type="button" @click="loadAgents">刷新</button>
        </div>
      </template>

      <div class="table-wrap">
        <table class="data-table agents-table">
          <thead>
            <tr>
              <th>代理账号</th>
              <th>等级</th>
              <th>上级</th>
              <th>比例</th>
              <th>客户</th>
              <th>可结算</th>
              <th>状态</th>
              <th>强制调整原因</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="agent in agents" :key="agent.id">
              <td>
                <strong>{{ agent.email }}</strong>
                <small>{{ agent.username }}</small>
              </td>
              <td>{{ agent.level }} 级</td>
              <td>{{ agent.parent_email || '-' }}</td>
              <td>
                <div class="percent-input">
                  <input
                    v-model="rateInputs[agent.id]"
                    class="table-input percent-table-input"
                    type="number"
                    min="0"
                    max="100"
                    step="0.01"
                    :title="formatPercent(agent.rate_bps)"
                  />
                  <span>%</span>
                </div>
              </td>
              <td>{{ agent.customers_count }}</td>
              <td>{{ formatMinorMoney(agent.payable_amount) }}</td>
              <td><StatusBadge :status="agent.status" /></td>
              <td>
                <input v-model="forceReasons[agent.id]" class="input reason-input" placeholder="填写后按强制调整审计" />
              </td>
              <td class="row-actions">
                <button class="link-button" type="button" @click="saveRate(agent)">保存比例</button>
                <button class="link-button" type="button" @click="toggleAgent(agent)">
                  {{ agent.status === 'active' ? '禁用' : '恢复' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!agents.length && !loading" />
    </SectionPanel>
  </div>
</template>
