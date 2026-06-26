<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import SectionPanel from '@/components/SectionPanel.vue'
import { getSettings, updateSettings } from '@/api/settings'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')
const updatedAt = ref('')

const form = reactive({
  turnstile_enabled: false,
  turnstile_site_key: ''
})

onMounted(loadSettings)

async function loadSettings() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const settings = await getSettings()
    form.turnstile_enabled = settings.turnstile_enabled
    form.turnstile_site_key = settings.turnstile_site_key || ''
    updatedAt.value = settings.updated_at || ''
  } catch (err) {
    error.value = messageFromError(err) || '设置加载失败'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (form.turnstile_enabled && !form.turnstile_site_key.trim()) {
    error.value = '启用 Turnstile 时必须填写 Site Key'
    success.value = ''
    return
  }

  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const settings = await updateSettings({
      turnstile_enabled: form.turnstile_enabled,
      turnstile_site_key: form.turnstile_site_key.trim()
    })
    form.turnstile_enabled = settings.turnstile_enabled
    form.turnstile_site_key = settings.turnstile_site_key || ''
    updatedAt.value = settings.updated_at || ''
    success.value = '设置已保存'
  } catch (err) {
    error.value = messageFromError(err) || '保存失败'
  } finally {
    saving.value = false
  }
}

function formatDate(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function messageFromError(err: unknown) {
  return err && typeof err === 'object' && 'message' in err
    ? String((err as { message?: string }).message)
    : ''
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>
    <p v-if="success" class="success-banner">{{ success }}</p>

    <SectionPanel
      title="登录安全"
      description="控制 Agent Admin 登录页是否展示 Cloudflare Turnstile 校验"
    >
      <form class="settings-form" @submit.prevent="saveSettings">
        <label class="switch-row">
          <span>
            <strong>启用 Turnstile</strong>
            <small>开启后，登录页会加载 Turnstile 组件并把校验 Token 提交给原系统登录接口。</small>
          </span>
          <input v-model="form.turnstile_enabled" type="checkbox" />
        </label>

        <label>
          <span>Site Key</span>
          <input
            v-model.trim="form.turnstile_site_key"
            class="input"
            autocomplete="off"
            placeholder="0x4AAAA..."
          />
        </label>

        <p class="field-hint">
          Secret Key 和实际校验仍由 sub2api 原系统配置；这里的 Site Key 应与原系统 Turnstile 配置保持一致。
        </p>

        <div class="form-footer">
          <span>最后更新：{{ formatDate(updatedAt) }}</span>
          <button class="secondary-button" type="button" :disabled="loading || saving" @click="loadSettings">
            重新加载
          </button>
          <button class="primary-button" type="submit" :disabled="loading || saving">
            {{ saving ? '保存中' : '保存设置' }}
          </button>
        </div>
      </form>
    </SectionPanel>
  </div>
</template>
