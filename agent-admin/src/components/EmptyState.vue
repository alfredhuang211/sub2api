<script setup lang="ts">
defineProps<{
  message?: string
}>()
</script>

<template>
  <p class="empty-state">{{ message || '暂无数据' }}</p>
</template>
