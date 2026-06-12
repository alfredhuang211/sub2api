<script setup lang="ts">
defineProps<{
  label: string
  value: string
  detail?: string
}>()
</script>

<template>
  <section class="metric-card">
    <span>{{ label }}</span>
    <strong>{{ value }}</strong>
    <small v-if="detail">{{ detail }}</small>
  </section>
</template>
