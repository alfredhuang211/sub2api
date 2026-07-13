<script setup lang="ts">
defineProps<{
  loading: boolean
}>()

defineEmits<{
  enter: []
}>()
</script>

<template>
  <div class="demo-login-entry">
    <p>当前为本地演示模式，可不连接数据库直接进入后台查看页面和样例数据。</p>
    <button class="secondary-button login-button" type="button" :disabled="loading" @click="$emit('enter')">
      进入本地演示
    </button>
  </div>
</template>

<style scoped>
.demo-login-entry {
  display: grid;
  gap: 12px;
}

.demo-login-entry p {
  margin-bottom: 0;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  background: #eff6ff;
  color: #1e40af;
  padding: 12px 14px;
  font-size: 13px;
  line-height: 1.6;
}
</style>
