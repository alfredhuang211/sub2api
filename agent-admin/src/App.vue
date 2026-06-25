<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { clearAuthTokens } from '@/api/auth'

const route = useRoute()
const router = useRouter()

const navItems = [
  { path: '/dashboard', label: '工作台' },
  { path: '/agents', label: '代理管理' },
  { path: '/agent-operations', label: '代理经营' },
  { path: '/customers', label: '客户归属' },
  { path: '/commissions', label: '分成记录' },
  { path: '/settlements', label: '结算记录' },
  { path: '/audit-logs', label: '审计日志' }
]

const pageTitle = computed(() => String(route.meta.title || '工作台'))
const isPublicPage = computed(() => Boolean(route.meta.public))

function logout() {
  clearAuthTokens()
  router.replace('/login')
}
</script>

<template>
  <RouterView v-if="isPublicPage" />

  <main v-else class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">S2</span>
        <div>
          <strong>代理经销商平台</strong>
          <small>Sub2API Agent Admin</small>
        </div>
      </div>

      <nav class="nav-list" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
          :to="item.path"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
    </aside>

    <section class="workspace">
      <header class="topbar">
        <div>
          <p class="eyebrow">Agent Admin</p>
          <h1>{{ pageTitle }}</h1>
        </div>
        <div class="identity-chip">
          <span>共用原系统账号体系</span>
          <button type="button" class="chip-action" @click="logout">退出</button>
        </div>
      </header>

      <RouterView />
    </section>
  </main>
</template>
