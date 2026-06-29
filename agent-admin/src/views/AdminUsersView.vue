<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import {
  grantAgentAdmin,
  listAgentAdminUsers,
  revokeAgentAdmin,
  searchAgentAdminCandidates,
  type AgentAdminUser,
  type UserOption
} from '@/api/agents'
import { formatDateTime } from '@/utils/format'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')
const search = ref('')
const userSearch = ref('')
const selectedUserId = ref('')
const admins = ref<AgentAdminUser[]>([])
const userOptions = ref<UserOption[]>([])
const userSearchLoading = ref(false)

onMounted(loadAdmins)

async function loadAdmins() {
  loading.value = true
  error.value = ''
  try {
    const page = await listAgentAdminUsers({ search: search.value, page_size: 100 })
    admins.value = page.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载管理员授权失败'
  } finally {
    loading.value = false
  }
}

async function searchUsers() {
  const keyword = userSearch.value.trim()
  if (!keyword) {
    userOptions.value = []
    selectedUserId.value = ''
    return
  }
  userSearchLoading.value = true
  error.value = ''
  success.value = ''
  try {
    const page = await searchAgentAdminCandidates({ search: keyword, page_size: 20 })
    userOptions.value = page.items ?? []
    if (!userOptions.value.some((item) => String(item.id) === selectedUserId.value)) {
      selectedUserId.value = ''
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || '搜索用户失败'
  } finally {
    userSearchLoading.value = false
  }
}

async function submitGrant() {
  if (!selectedUserId.value) return
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    await grantAgentAdmin(Number(selectedUserId.value))
    selectedUserId.value = ''
    userSearch.value = ''
    userOptions.value = []
    success.value = '管理员授权已保存'
    await loadAdmins()
  } catch (err) {
    error.value = (err as { message?: string }).message || '授权失败'
  } finally {
    saving.value = false
  }
}

async function revokeAdmin(admin: AgentAdminUser) {
  if (admin.source === 'base' || admin.status !== 'active') return
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    await revokeAgentAdmin(admin.id)
    success.value = '管理员授权已撤销'
    await loadAdmins()
  } catch (err) {
    error.value = (err as { message?: string }).message || '撤销失败'
  } finally {
    saving.value = false
  }
}

function selectedUserLabel() {
  const selected = userOptions.value.find((item) => String(item.id) === selectedUserId.value)
  if (!selected) return ''
  return `${selected.email} / ${selected.username || '-'}`
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>
    <p v-if="success" class="success-banner">{{ success }}</p>

    <SectionPanel title="新增管理员授权" description="基础管理员可以把原系统用户授权为 agent-admin 管理员">
      <form class="form-grid" @submit.prevent="submitGrant">
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
          <select v-model="selectedUserId" class="input" required>
            <option value="" disabled>请选择用户</option>
            <option v-for="user in userOptions" :key="user.id" :value="String(user.id)">
              {{ user.email }} / {{ user.username || '-' }}
            </option>
          </select>
          <small v-if="selectedUserId" class="field-hint">已选择：{{ selectedUserLabel() }}</small>
        </label>
        <button class="primary-button" type="submit" :disabled="saving || !selectedUserId">
          {{ saving ? '保存中' : '授权管理员' }}
        </button>
      </form>
    </SectionPanel>

    <SectionPanel title="管理员列表" description="原系统管理员为基础管理员，授权管理员只能管理代理业务">
      <template #actions>
        <div class="toolbar inline">
          <input v-model="search" class="search-input" type="search" placeholder="搜索邮箱或用户名" @keyup.enter="loadAdmins" />
          <button class="secondary-button" type="button" @click="loadAdmins">刷新</button>
        </div>
      </template>

      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>账号</th>
              <th>类型</th>
              <th>状态</th>
              <th>授权人</th>
              <th>授权时间</th>
              <th>撤销时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="admin in admins" :key="`${admin.source}-${admin.user_id}`">
              <td>
                <strong>{{ admin.email }}</strong>
                <small>{{ admin.username || '-' }}</small>
              </td>
              <td>{{ admin.source === 'base' ? '基础管理员' : '授权管理员' }}</td>
              <td><StatusBadge :status="admin.status" /></td>
              <td>{{ admin.created_by_email || '-' }}</td>
              <td>{{ formatDateTime(admin.created_at) }}</td>
              <td>{{ formatDateTime(admin.revoked_at) }}</td>
              <td>
                <button
                  v-if="admin.source === 'delegated' && admin.status === 'active'"
                  class="link-button"
                  type="button"
                  :disabled="saving"
                  @click="revokeAdmin(admin)"
                >
                  撤销授权
                </button>
                <span v-else>-</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!admins.length && !loading" />
    </SectionPanel>
  </div>
</template>
