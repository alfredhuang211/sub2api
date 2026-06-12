<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import { listAuditLogs, type AgentAuditLog } from '@/api/agents'
import { formatDateTime } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const logs = ref<AgentAuditLog[]>([])

onMounted(loadLogs)

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const page = await listAuditLogs()
    logs.value = page.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载审计日志失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="审计日志" description="记录指定代理、层级调整、比例调整、客户归属、禁用恢复和账务调整">
      <template #actions>
        <button class="secondary-button" type="button" @click="loadLogs">刷新</button>
      </template>

      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>操作人</th>
              <th>角色</th>
              <th>动作</th>
              <th>对象</th>
              <th>原因</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id">
              <td>{{ log.operator_email }}</td>
              <td>{{ log.operator_role }}</td>
              <td>{{ log.action }}</td>
              <td>{{ log.target_type }} #{{ log.target_id }}</td>
              <td>{{ log.reason || '-' }}</td>
              <td>{{ formatDateTime(log.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!logs.length && !loading" />
    </SectionPanel>
  </div>
</template>
