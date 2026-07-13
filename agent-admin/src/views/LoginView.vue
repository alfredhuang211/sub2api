<script setup lang="ts">
import { nextTick, onMounted, ref, shallowRef, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login, login2FA, saveAuthTokens } from '@/api/auth'
import { getLocalDemoLoginResponse, loadLocalDemoLoginEntry } from '@/api/localDemo'
import { getCurrentUser } from '@/api/session'
import { getPublicSettings } from '@/api/settings'

declare global {
  interface Window {
    turnstile?: {
      render: (
        element: HTMLElement,
        options: {
          sitekey: string
          callback: (token: string) => void
          'expired-callback': () => void
          'error-callback': () => void
        }
      ) => string
      reset: (widgetId?: string) => void
      remove: (widgetId: string) => void
    }
  }
}

const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const turnstileToken = ref('')
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const turnstileContainer = ref<HTMLElement | null>(null)
const turnstileWidgetId = ref('')
const totpCode = ref('')
const tempToken = ref('')
const maskedEmail = ref('')
const loading = ref(false)
const settingsLoading = ref(true)
const error = ref('')
const demoLoginEntry = shallowRef<Component | null>(null)

onMounted(() => {
  loadPublicSettings()
  loadDemoLoginEntry()
})

async function loadDemoLoginEntry() {
  demoLoginEntry.value = await loadLocalDemoLoginEntry()
}

async function loadPublicSettings() {
  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled === true
    turnstileSiteKey.value = settings.turnstile_site_key || ''
  } catch {
    turnstileEnabled.value = false
    turnstileSiteKey.value = ''
  } finally {
    settingsLoading.value = false
  }

  if (turnstileEnabled.value && turnstileSiteKey.value) {
    await nextTick()
    try {
      await renderTurnstile()
    } catch {
      error.value = 'Turnstile 加载失败，请刷新后重试'
    }
  }
}

async function submitLogin() {
  loading.value = true
  error.value = ''
  try {
    if (!tempToken.value && settingsLoading.value) {
      error.value = '登录设置加载中，请稍后'
      return
    }
    if (!tempToken.value && turnstileEnabled.value && turnstileSiteKey.value && !turnstileToken.value) {
      error.value = '请先完成 Turnstile 校验'
      return
    }

    const response = tempToken.value
      ? await login2FA({ temp_token: tempToken.value, totp_code: totpCode.value })
      : await login({
          email: email.value,
          password: password.value,
          turnstile_token: turnstileToken.value.trim() || undefined
        })

    if (response.requires_2fa && response.temp_token) {
      tempToken.value = response.temp_token
      maskedEmail.value = response.user_email_masked || email.value
      return
    }

    if (!response.access_token) {
      error.value = '登录成功但未返回访问令牌'
      return
    }

    saveAuthTokens(response)
    try {
      await getCurrentUser(true)
    } catch {
      error.value = '登录成功，但身份信息加载失败，请稍后重试或刷新页面'
      return
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch (err) {
    const message = err && typeof err === 'object' && 'message' in err ? String((err as { message?: string }).message) : ''
    error.value = message || '登录失败，请检查账号密码'
    resetTurnstile()
  } finally {
    loading.value = false
  }
}

async function enterDemoMode() {
  loading.value = true
  error.value = ''
  try {
    const response = await getLocalDemoLoginResponse()
    if (!response) return
    saveAuthTokens(response)
    await getCurrentUser(true)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch (err) {
    const message = err && typeof err === 'object' && 'message' in err ? String((err as { message?: string }).message) : ''
    error.value = message || '操作失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

async function renderTurnstile() {
  if (!turnstileContainer.value || turnstileWidgetId.value || !turnstileSiteKey.value) {
    return
  }
  await loadTurnstileScript()
  if (!window.turnstile || !turnstileContainer.value) {
    throw new Error('turnstile unavailable')
  }
  turnstileWidgetId.value = window.turnstile.render(turnstileContainer.value, {
    sitekey: turnstileSiteKey.value,
    callback: (token: string) => {
      turnstileToken.value = token
      error.value = ''
    },
    'expired-callback': () => {
      turnstileToken.value = ''
      error.value = 'Turnstile 校验已过期，请重新验证'
    },
    'error-callback': () => {
      turnstileToken.value = ''
      error.value = 'Turnstile 校验失败，请重试'
    }
  })
}

function resetTurnstile() {
  if (turnstileWidgetId.value && window.turnstile) {
    window.turnstile.reset(turnstileWidgetId.value)
    turnstileToken.value = ''
  }
}

function loadTurnstileScript() {
  if (window.turnstile) {
    return Promise.resolve()
  }
  const existing = document.querySelector<HTMLScriptElement>('script[data-agent-admin-turnstile]')
  if (existing) {
    return new Promise<void>((resolve, reject) => {
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener('error', () => reject(new Error('turnstile script load failed')), { once: true })
    })
  }
  return new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    script.async = true
    script.defer = true
    script.dataset.agentAdminTurnstile = 'true'
    script.addEventListener('load', () => resolve(), { once: true })
    script.addEventListener('error', () => reject(new Error('turnstile script load failed')), { once: true })
    document.head.appendChild(script)
  })
}
</script>

<template>
  <main class="login-shell">
    <section class="login-panel">
      <div class="brand login-brand">
        <span class="brand-mark">S2</span>
        <div>
          <strong>代理经销商平台</strong>
          <small>Sub2API Agent Admin</small>
        </div>
      </div>

      <form class="login-form" @submit.prevent="submitLogin">
        <component
          :is="demoLoginEntry"
          v-if="demoLoginEntry && !tempToken"
          :loading="loading"
          @enter="enterDemoMode"
        />

        <label v-if="!tempToken">
          <span>邮箱</span>
          <input v-model.trim="email" class="input" type="email" autocomplete="username" required />
        </label>

        <label v-if="!tempToken">
          <span>密码</span>
          <input v-model="password" class="input" type="password" autocomplete="current-password" required />
        </label>

        <div v-if="!tempToken && turnstileEnabled && turnstileSiteKey" class="turnstile-box">
          <div ref="turnstileContainer"></div>
        </div>

        <label v-if="tempToken">
          <span>两步验证码 {{ maskedEmail }}</span>
          <input v-model.trim="totpCode" class="input" inputmode="numeric" maxlength="6" autocomplete="one-time-code" required />
        </label>

        <p v-if="error" class="error-banner">{{ error }}</p>

        <button class="primary-button login-button" type="submit" :disabled="loading">
          {{ loading ? '登录中' : settingsLoading && !tempToken ? '加载中' : tempToken ? '验证并登录' : '登录' }}
        </button>
      </form>
    </section>
  </main>
</template>
