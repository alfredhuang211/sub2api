<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import {
  createMyChildAgent,
  listMyChildren,
  listMyDevelopableUsers,
  listMyOrders,
  updateMyChildRate,
  type AgentCustomer,
  type AgentOrder,
  type AgentProfile
} from '@/api/agents'
import { formatDateTime, formatMinorMoney, formatPercent } from '@/utils/format'

const loading = ref(false)
const error = ref('')
const developableUsers = ref<AgentCustomer[]>([])
const children = ref<AgentProfile[]>([])
const orders = ref<AgentOrder[]>([])
const childRateInputs = reactive<Record<number, string>>({})

const childForm = reactive({
  user_id: '',
  rate_percent: ''
})

onMounted(loadAll)

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [developableUserPage, childPage, orderPage] = await Promise.all([
      listMyDevelopableUsers({ page_size: 50 }),
      listMyChildren({ page_size: 50 }),
      listMyOrders({ page_size: 50 })
    ])
    developableUsers.value = developableUserPage.items ?? []
    children.value = childPage.items ?? []
    for (const child of children.value) {
      childRateInputs[child.id] = formatRatePercent(child.rate_bps)
    }
    orders.value = orderPage.items ?? []
  } catch (err) {
    error.value = (err as { message?: string }).message || '加载代理经营数据失败'
  } finally {
    loading.value = false
  }
}

async function submitChild() {
  await createMyChildAgent({
    user_id: Number(childForm.user_id),
    rate_bps: childForm.rate_percent ? Math.round(Number(childForm.rate_percent) * 100) : undefined
  })
  childForm.user_id = ''
  childForm.rate_percent = ''
  await loadAll()
}

function selectedDevelopableUserLabel() {
  const selected = developableUsers.value.find((item) => String(item.user_id) === childForm.user_id)
  if (!selected) return ''
  return `${selected.email} / ${sourceLabel(selected.source)}`
}

async function saveChildRate(child: AgentProfile) {
  await updateMyChildRate(child.id, parseRatePercent(childRateInputs[child.id]))
  await loadAll()
}

function formatRatePercent(rateBps: number) {
  return (Number(rateBps || 0) / 100).toFixed(2)
}

function parseRatePercent(value: string) {
  const percent = Number(value)
  return Number.isFinite(percent) ? Math.round(percent * 100) : 0
}

function sourceLabel(source: AgentCustomer['source']) {
  return source === 'manual' ? '管理员归属' : '自然推荐'
}
</script>

<template>
  <div class="page-stack">
    <p v-if="error" class="error-banner">{{ error }}</p>

    <SectionPanel title="创建下级代理" description="可从直接归属自己的自然推荐或管理员归属用户中创建下级代理">
      <form class="form-grid" @submit.prevent="submitChild">
        <label class="wide">
          <span>可发展用户</span>
          <select v-model="childForm.user_id" class="input" required>
            <option value="" disabled>选择可发展用户</option>
            <option v-for="item in developableUsers" :key="item.user_id" :value="String(item.user_id)">
              {{ item.email }} / {{ sourceLabel(item.source) }}
            </option>
          </select>
          <small v-if="childForm.user_id" class="field-hint">已选择：{{ selectedDevelopableUserLabel() }}</small>
        </label>
        <label>
          <span>比例 %</span>
          <input v-model="childForm.rate_percent" class="input" type="number" min="0" max="100" step="0.01" placeholder="默认比例" />
        </label>
        <button class="primary-button" type="submit">创建下级</button>
      </form>
    </SectionPanel>

    <SectionPanel title="我的可发展用户" description="直接归属于自己的用户，可来自自然推荐或管理员手动归属">
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>用户</th>
              <th>来源</th>
              <th>推荐码</th>
              <th>套餐</th>
              <th>周期结束</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in developableUsers" :key="item.user_id">
              <td>
                <strong>{{ item.email }}</strong>
                <small>ID {{ item.user_id }} / {{ item.username }}</small>
              </td>
              <td>{{ sourceLabel(item.source) }}</td>
              <td>{{ item.source_referral_code || '-' }}</td>
              <td>{{ item.subscription_name || '-' }}</td>
              <td>{{ formatDateTime(item.period_end_at) }}</td>
              <td>{{ item.status === 'scheduled' ? '待生效' : '有效' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!developableUsers.length && !loading" />
    </SectionPanel>

    <SectionPanel title="我的下级代理" description="可调整直属下级比例，比例必须低于自己的比例">
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>账号</th>
              <th>等级</th>
              <th>比例</th>
              <th>客户</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="child in children" :key="child.id">
              <td>
                <strong>{{ child.email }}</strong>
                <small>{{ child.username }}</small>
              </td>
              <td>{{ child.level }} 级</td>
              <td>
                <div class="percent-input">
                  <input
                    v-model="childRateInputs[child.id]"
                    class="table-input percent-table-input"
                    type="number"
                    min="0"
                    max="100"
                    step="0.01"
                    :title="formatPercent(child.rate_bps)"
                  />
                  <span>%</span>
                </div>
              </td>
              <td>{{ child.customers_count }}</td>
              <td><StatusBadge :status="child.status" /></td>
              <td><button class="link-button" type="button" @click="saveChildRate(child)">保存比例</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!children.length && !loading" />
    </SectionPanel>

    <SectionPanel title="我的客户订单" description="只读展示直接客户订单">
      <div class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>订单</th>
              <th>客户</th>
              <th>金额</th>
              <th>支付时间</th>
              <th>完成时间</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="order in orders" :key="order.id">
              <td>{{ order.order_no || `#${order.id}` }}</td>
              <td>{{ order.customer_email }}</td>
              <td>{{ formatMinorMoney(order.pay_amount) }}</td>
              <td>{{ formatDateTime(order.paid_at) }}</td>
              <td>{{ formatDateTime(order.completed_at) }}</td>
              <td>{{ order.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-if="!orders.length && !loading" />
    </SectionPanel>
  </div>
</template>
