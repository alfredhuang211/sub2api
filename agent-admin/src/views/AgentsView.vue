<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import {
  createAgent,
  disableAgent,
  forceAdjustAgent,
  listAgents,
  restoreAgent,
  updateAgent,
  type AgentLevel,
  type AgentProfile
} from '@/api/agents'
import { formatMinorMoney, formatPercent } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const agents = ref<AgentProfile[]>([])
const search = ref('')

const form = reactive({
  user_id: '',
  level: 1 as AgentLevel,
  parent_agent_id: '',
  rate_percent: ''
})

const forceReasons = reactive<Record<number, string>>({})

onMounted(loadAgents)

async function loadAgents() {
  loading.value = true
  error.value = ''
  try {
    const page = await listAgents({ search: search.value })
    agents.value = page.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载代理列表失败'
  } finally {
    loading.value = false
  }
}

async function submitAgent() {
  const payload = {
    user_id: Number(form.user_id),
    level: form.level,
    parent_agent_id: form.parent_agent_id ? Number(form.parent_agent_id) : null,
    rate_bps: form.rate_percent ? Math.round(Number(form.rate_percent) * 100) : undefined
  }
  await createAgent(payload)
  form.user_id = ''
  form.parent_agent_id = ''
  form.rate_percent = ''
  await loadAgents()
}

async function saveRate(agent: AgentProfile) {
  const reason = forceReasons[agent.id]?.trim()
  if (reason) {
    await forceAdjustAgent(agent.id, { rate_bps: agent.rate_bps, reason })
    forceReasons[agent.id] = ''
  } else {
    await updateAgent(agent.id, { rate_bps: agent.rate_bps })
  }
  await loadAgents()
}

async function toggleAgent(agent: AgentProfile) {
  if (agent.status === 'active') {
    await disableAgent(agent.id)
  } else {
    await restoreAgent(agent.id)
  }
  await loadAgents()
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="指定代理" description="管理员从原系统用户中指定 1/2/3 级代理">
      <form class="form-grid" @submit.prevent="submitAgent">
        <label>
          <span>用户 ID</span>
          <input v-model="form.user_id" class="input" type="number" min="1" required />
        </label>
        <label>
          <span>代理等级</span>
          <select v-model.number="form.level" class="input">
            <option :value="1">1 级代理</option>
            <option :value="2">2 级代理</option>
            <option :value="3">3 级代理</option>
          </select>
        </label>
        <label>
          <span>上级代理 ID</span>
          <input v-model="form.parent_agent_id" class="input" type="number" min="1" placeholder="2/3 级必填" />
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
        <table class="data-table">
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
                <input
                  v-model.number="agent.rate_bps"
                  class="table-input"
                  type="number"
                  min="0"
                  step="1"
                  :title="formatPercent(agent.rate_bps)"
                />
              </td>
              <td>{{ agent.customers_count }}</td>
              <td>{{ formatMinorMoney(agent.payable_amount) }}</td>
              <td><StatusBadge :status="agent.status" /></td>
              <td>
                <input v-model="forceReasons[agent.id]" class="input compact-input" placeholder="填写后按强制调整审计" />
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
