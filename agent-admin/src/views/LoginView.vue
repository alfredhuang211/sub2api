<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login, login2FA, saveAuthTokens } from '@/api/auth'

const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const turnstileToken = ref('')
const totpCode = ref('')
const tempToken = ref('')
const maskedEmail = ref('')
const loading = ref(false)
const error = ref('')

async function submitLogin() {
  loading.value = true
  error.value = ''
  try {
    const response = tempToken.value
      ? await login2FA({ temp_token: tempToken.value, totp_code: totpCode.value })
      : await login({ email: email.value, password: password.value, turnstile_token: turnstileToken.value.trim() })

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
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    router.replace(redirect)
  } catch (err) {
    const message = err && typeof err === 'object' && 'message' in err ? String((err as { message?: string }).message) : ''
    error.value = message || '登录失败，请检查账号密码'
  } finally {
    loading.value = false
  }
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
        <label v-if="!tempToken">
          <span>邮箱</span>
          <input v-model.trim="email" class="input" type="email" autocomplete="username" required />
        </label>

        <label v-if="!tempToken">
          <span>密码</span>
          <input v-model="password" class="input" type="password" autocomplete="current-password" required />
        </label>

        <label v-if="!tempToken">
          <span>Turnstile Token</span>
          <input v-model.trim="turnstileToken" class="input" autocomplete="off" placeholder="原系统启用校验时填写" />
        </label>

        <label v-if="tempToken">
          <span>两步验证码 {{ maskedEmail }}</span>
          <input v-model.trim="totpCode" class="input" inputmode="numeric" maxlength="6" autocomplete="one-time-code" required />
        </label>

        <p v-if="error" class="error-banner">{{ error }}</p>

        <button class="primary-button login-button" type="submit" :disabled="loading">
          {{ loading ? '登录中' : tempToken ? '验证并登录' : '登录' }}
        </button>
      </form>
    </section>
  </main>
</template>
