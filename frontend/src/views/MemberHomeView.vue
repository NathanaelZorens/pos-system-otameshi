<template>
  <div class="guest">
    <header class="guest-header">
      <span class="eyebrow" style="margin: 0">Member</span>
      <button type="button" @click="onLogout">Log out</button>
    </header>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else>
      <div class="page-head">
        <h1 style="margin: 0">Hi, {{ member?.name || 'there' }}</h1>
        <p class="muted" style="margin: 0.35rem 0 0">Your takeout hub</p>
      </div>

      <div v-if="activeOrder" class="card accent">
        <p class="eyebrow" style="margin-bottom: 0.35rem">Active order</p>
        <h2 style="margin: 0 0 0.35rem">
          {{ activeOrder.pickup_code || '—' }} · {{ activeOrder.status }}
        </h2>
        <p class="muted" style="margin: 0 0 1rem">
          {{ formatMoney(activeOrder.total_cents) }}
        </p>
        <RouterLink to="/member/takeout">
          <button type="button" class="primary" style="width: 100%">Open order</button>
        </RouterLink>
      </div>

      <RouterLink to="/member/takeout" style="display: block; margin-bottom: 1rem">
        <button type="button" class="gold" style="width: 100%; height: 48px">
          {{ activeOrder ? 'Open takeout' : 'New takeout order' }}
        </button>
      </RouterLink>

      <div class="card">
        <h2 style="margin-top: 0">Recent</h2>
        <p v-if="!history.length" class="muted">No completed or cancelled takeout yet.</p>
        <ul v-else class="history-list">
          <li v-for="h in history" :key="h.id" class="history-item">
            <div>
              <strong>{{ h.pickup_code || '—' }}</strong>
              <span class="muted"> · {{ h.status }}</span>
              <div class="muted">
                {{ h.created_at }}
                · {{ h.item_count }} item{{ h.item_count === 1 ? '' : 's' }}
                <span v-if="h.pay_method"> · {{ h.pay_method }}</span>
              </div>
              <RouterLink
                v-if="h.status === 'completed'"
                :to="`/member/receipt/${h.id}`"
                class="receipt-link"
              >
                Show receipt
              </RouterLink>
            </div>
            <strong>{{ formatMoney(h.total_cents) }}</strong>
          </li>
        </ul>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { memberApi, formatMoney } from '../api'
import { useMemberAuth } from '../memberAuth'

const router = useRouter()
const { member, logoutMember } = useMemberAuth()

const loading = ref(true)
const error = ref('')
const activeOrder = ref(null)
const history = ref([])

onMounted(boot)

async function boot() {
  loading.value = true
  error.value = ''
  try {
    const [active, hist] = await Promise.all([
      memberApi('/takeout/active'),
      memberApi('/takeout/history'),
    ])
    history.value = hist || []
    if (active.order_id) {
      activeOrder.value = await memberApi(`/takeout/orders/${active.order_id}`)
    } else {
      activeOrder.value = null
    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function onLogout() {
  logoutMember()
  router.replace('/member/login')
}
</script>

<style scoped>
.history-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.history-item {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--line);
}
.history-item:last-child {
  border-bottom: none;
}
.receipt-link {
  display: inline-block;
  margin-top: 0.35rem;
  font-size: 0.9rem;
}
</style>
