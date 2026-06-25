<script setup lang="ts">
defineProps<{
  status: string
}>()

const labels: Record<string, string> = {
  active: '启用',
  scheduled: '待生效',
  disabled: '禁用',
  pending: '待结算',
  frozen: '冻结中',
  payable: '可结算',
  paid: '已结算',
  reversed: '已冲正'
}
</script>

<template>
  <span class="status-badge" :class="`status-${status}`">
    {{ labels[status] || status }}
  </span>
</template>
