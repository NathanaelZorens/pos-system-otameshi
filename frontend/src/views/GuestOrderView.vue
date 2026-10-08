<template>
  <div class="guest">
    <header class="guest-header">
      <strong style="font-family: var(--font-display); font-size: 1.35rem">Table {{ tableLabel || '…' }}</strong>
      <span class="status-pill busy">Guest</span>
    </header>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <div v-else-if="blocked" class="card">
      <h1>Table in use</h1>
      <p>{{ blockedReason }}</p>
      <p class="muted">Please ask a waiter to help with this table.</p>
    </div>

    <div v-else-if="mode === 'intro'" class="auth-shell" style="min-height: 60vh; padding: 1rem 0">
      <div style="width: 100%">
        <p class="eyebrow">Table {{ tableLabel }}</p>
        <h1 style="margin: 0 0 0.75rem; font-size: 2.4rem">Welcome</h1>
        <p class="muted" style="margin: 0 0 1.25rem; line-height: 1.45">
          Order from your phone. Confirm dishes when ready, then pay when you ask for the bill.
        </p>
        <button type="button" class="primary" style="width: 100%; height: 48px" @click="beginOrdering" :disabled="busy">
          {{ busy ? 'Loading…' : 'Start ordering' }}
        </button>
        <p class="muted" style="margin: 1rem 0 0; text-align: center">
          Ask staff if this isn’t your table.
        </p>
      </div>
    </div>

    <template v-else-if="mode === 'draft'">
      <div class="card">
        <label>
          Your name (optional)
          <input v-model="customerName" placeholder="Alex" style="width: 100%; margin-top: 0.25rem" />
        </label>
        <p class="muted" style="margin-top: 0.75rem">
          Nothing is saved until you add the first item.
        </p>
        <p><strong>Total: {{ formatMoney(0) }}</strong></p>
      </div>
      <div class="card">
        <h2>Menu</h2>
        <div class="row cat-filters" v-if="categoryFilters.length">
          <button type="button" :class="{ primary: !categoryFilter }" @click="categoryFilter = ''">All</button>
          <button
            v-for="c in categoryFilters"
            :key="c.id"
            type="button"
            :class="{ primary: categoryFilter === c.id }"
            @click="categoryFilter = c.id"
          >
            {{ c.name }}
          </button>
        </div>
        <div class="menu-list">
          <div v-for="m in filteredMenu" :key="m.id" class="menu-item" :class="{ 'sold-out': m.is_sold_out }">
            <div>
              <strong>{{ m.name }}</strong>
              <div class="muted">{{ m.category || '—' }} · {{ formatMoney(m.price_cents) }}</div>
            </div>
            <div class="row" v-if="!m.is_sold_out">
              <button type="button" @click="bumpAddQty(m.id, -1)" :disabled="busy || addQty(m.id) <= 1">−</button>
              <input class="qty-input" type="number" min="1" :value="addQty(m.id)" @change="setAddQty(m.id, $event.target.value)" />
              <button type="button" @click="bumpAddQty(m.id, 1)" :disabled="busy">+</button>
              <button type="button" class="primary" @click="startOrder(m)" :disabled="busy">Add</button>
            </div>
            <span v-else class="muted">Sold out</span>
          </div>
        </div>
      </div>
    </template>

    <template v-else-if="order">
      <div class="card">
        <p class="muted">Order: {{ order.status }}</p>
        <h2>Your order</h2>
        <ul class="order-lines">
          <li v-for="it in (order.items || [])" :key="it.id" class="order-line">
            <div>
              <strong>{{ it.name_snapshot }}</strong>
              <div class="muted">
                {{ formatMoney(it.unit_price_cents) }} each
                · <span :class="it.status === 'confirmed' ? 'badge-confirmed' : 'badge-pending'">{{ it.status }}</span>
              </div>
            </div>
            <div class="row qty-controls" v-if="isActiveOrder && it.status === 'pending'">
              <button type="button" @click="changeQty(it, it.quantity - 1)" :disabled="busy">−</button>
              <span class="qty">{{ it.quantity }}</span>
              <button type="button" @click="changeQty(it, it.quantity + 1)" :disabled="busy">+</button>
              <button type="button" class="danger" @click="changeQty(it, 0)" :disabled="busy">Remove</button>
            </div>
            <div v-else class="qty">{{ it.quantity }}×</div>
            <strong>{{ formatMoney(it.line_total_cents) }}</strong>
          </li>
        </ul>
        <p><strong>Total: {{ formatMoney(order.total_cents) }}</strong></p>
        <p v-if="order.pending_count" class="muted">{{ order.pending_count }} pending — confirm before paying.</p>

        <div v-if="isActiveOrder" class="row">
          <button
            v-if="order.pending_count > 0"
            type="button"
            class="primary"
            @click="confirmItems"
            :disabled="busy"
          >
            Confirm
          </button>
          <template v-if="canPay">
            <select v-model="payMethod">
              <option value="card">Card</option>
              <option value="qr">QR</option>
              <option value="wallet">Wallet</option>
            </select>
            <button class="primary" type="button" @click="checkout" :disabled="busy">
              Pay
            </button>
          </template>
          <button
            v-if="!(order.confirmed_count > 0)"
            class="danger"
            type="button"
            @click="quitOrder"
            :disabled="busy"
          >
            Quit
          </button>
          <p v-else-if="!canPay" class="muted">Already confirmed — ask staff if you need to stop the order.</p>
        </div>
      </div>

      <div v-if="isActiveOrder" class="card">
        <h2>Add more</h2>
        <div class="row cat-filters" v-if="categoryFilters.length">
          <button type="button" :class="{ primary: !categoryFilter }" @click="categoryFilter = ''">All</button>
          <button
            v-for="c in categoryFilters"
            :key="c.id"
            type="button"
            :class="{ primary: categoryFilter === c.id }"
            @click="categoryFilter = c.id"
          >
            {{ c.name }}
          </button>
        </div>
        <div class="menu-list">
          <div v-for="m in filteredMenu" :key="m.id" class="menu-item" :class="{ 'sold-out': m.is_sold_out }">
            <div>
              <strong>{{ m.name }}</strong>
              <div class="muted">{{ formatMoney(m.price_cents) }}</div>
            </div>
            <div class="row" v-if="!m.is_sold_out">
              <button type="button" @click="bumpAddQty(m.id, -1)" :disabled="busy || addQty(m.id) <= 1">−</button>
              <input class="qty-input" type="number" min="1" :value="addQty(m.id)" @change="setAddQty(m.id, $event.target.value)" />
              <button type="button" @click="bumpAddQty(m.id, 1)" :disabled="busy">+</button>
              <button type="button" class="primary" @click="addItem(m.id)" :disabled="busy">Add</button>
            </div>
            <span v-else class="muted">Sold out</span>
          </div>
        </div>
      </div>
    </template>

    <CancelReasonModal
      :open="quitModalOpen"
      title="Quit order"
      message="You can start again if the table is still free. Enter a reason."
      confirm-label="Quit"
      :busy="busy"
      @confirm="submitQuit"
      @cancel="quitModalOpen = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { guestApi, formatMoney, setGuestClaim, clearGuestClaim } from '../api'
import CancelReasonModal from '../components/CancelReasonModal.vue'
import { categoryFiltersFromMenu, filterMenuByCategory } from '../menuFilters'

const route = useRoute()
const router = useRouter()
const token = computed(() => route.params.token)

const loading = ref(true)
const error = ref('')
const blocked = ref(false)
const blockedReason = ref('')
const tableLabel = ref('')
const mode = ref('intro') // intro | draft | order
const order = ref(null)
const menu = ref([])
const categoryFilter = ref('')
const customerName = ref('')
const payMethod = ref('qr')
const busy = ref(false)
const addQtys = reactive({})
const quitModalOpen = ref(false)

const isActiveOrder = computed(() =>
  ['open', 'preparing', 'served'].includes(order.value?.status),
)

const canPay = computed(
  () =>
    isActiveOrder.value &&
    (order.value?.confirmed_count || 0) > 0 &&
    !(order.value?.pending_count > 0),
)

const categoryFilters = computed(() => categoryFiltersFromMenu(menu.value))
const filteredMenu = computed(() => filterMenuByCategory(menu.value, categoryFilter.value))

let pollTimer = null

function addQty(id) {
  return addQtys[id] || 1
}
function setAddQty(id, value) {
  addQtys[id] = Math.max(1, parseInt(value, 10) || 1)
}
function bumpAddQty(id, delta) {
  addQtys[id] = Math.max(1, addQty(id) + delta)
}

function forgetGuestSession() {
  clearGuestClaim(token.value)
  // legacy key from pre-claim resume
  sessionStorage.removeItem(`guestOrder:${token.value}`)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  stopPolling()
  if (!order.value?.id || !isActiveOrder.value) return
  pollTimer = setInterval(() => {
    refreshOrder()
  }, 2500)
}

async function refreshOrder() {
  if (!order.value?.id || busy.value) return
  try {
    const data = await guestApi(token.value, `/orders/${order.value.id}`)
    data.items = data.items || []
    if (data.status === 'paid') {
      stopPolling()
      forgetGuestSession()
      router.replace(`/t/${token.value}/receipt/${data.id}`)
      return
    }
    if (data.status === 'cancelled') {
      stopPolling()
      forgetGuestSession()
      order.value = null
      mode.value = 'draft'
      blocked.value = false
      return
    }
    order.value = data
    if (data.table_label) tableLabel.value = data.table_label
  } catch {
    // keep last known state; next poll retries
  }
}

function onVisibility() {
  if (document.visibilityState === 'visible' && isActiveOrder.value) {
    refreshOrder()
  }
}

watch(isActiveOrder, (active) => {
  if (active) startPolling()
  else stopPolling()
})

onMounted(() => {
  boot()
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisibility)
})

async function loadActiveOrder(orderId) {
  const data = await guestApi(token.value, `/orders/${orderId}`)
  data.items = data.items || []
  if (data.status === 'paid') {
    forgetGuestSession()
    router.replace(`/t/${token.value}/receipt/${orderId}`)
    return false
  }
  if (!['open', 'preparing', 'served'].includes(data.status)) {
    forgetGuestSession()
    return false
  }
  order.value = data
  tableLabel.value = data.table_label
  mode.value = 'order'
  menu.value = (await guestApi(token.value, '/menu')) || []
  startPolling()
  return true
}

async function boot() {
  loading.value = true
  error.value = ''
  try {
    const status = await guestApi(token.value, '/')
    tableLabel.value = status.table_label

    if (status.resume_order_id) {
      const ok = await loadActiveOrder(status.resume_order_id)
      if (ok) return
    }

    if (status.blocked) {
      blocked.value = true
      blockedReason.value = status.blocked_reason || 'This table is busy.'
      return
    }

    mode.value = 'intro'
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function beginOrdering() {
  error.value = ''
  busy.value = true
  try {
    menu.value = (await guestApi(token.value, '/menu')) || []
    mode.value = 'draft'
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function startOrder(m) {
  error.value = ''
  busy.value = true
  try {
    const data = await guestApi(token.value, '/orders', {
      method: 'POST',
      body: JSON.stringify({
        customer_name: customerName.value.trim() || null,
        items: [{ menu_item_id: m.id, quantity: addQty(m.id) }],
      }),
    })
    data.items = data.items || []
    if (data.guest_claim) setGuestClaim(token.value, data.guest_claim)
    order.value = data
    mode.value = 'order'
    blocked.value = false
    addQtys[m.id] = 1
  } catch (e) {
    error.value = e.message
    if (/already has an open order/i.test(e.message)) {
      blocked.value = true
      blockedReason.value = 'This table already has an open order. Please ask staff for help.'
      mode.value = 'draft'
    }
  } finally {
    busy.value = false
  }
}

async function addItem(menuItemId) {
  error.value = ''
  busy.value = true
  try {
    const data = await guestApi(token.value, `/orders/${order.value.id}/items`, {
      method: 'POST',
      body: JSON.stringify({ menu_item_id: menuItemId, quantity: addQty(menuItemId) }),
    })
    data.items = data.items || []
    order.value = data
    addQtys[menuItemId] = 1
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function changeQty(item, quantity) {
  error.value = ''
  busy.value = true
  try {
    const data = await guestApi(token.value, `/orders/${order.value.id}/items/${item.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ quantity }),
    })
    data.items = data.items || []
    order.value = data
    if (data.status === 'cancelled') {
      forgetGuestSession()
      order.value = null
      mode.value = 'draft'
      blocked.value = false
    }
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function confirmItems() {
  error.value = ''
  busy.value = true
  try {
    const data = await guestApi(token.value, `/orders/${order.value.id}/confirm`, { method: 'POST' })
    data.items = data.items || []
    order.value = data
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function checkout() {
  error.value = ''
  busy.value = true
  try {
    const data = await guestApi(token.value, `/orders/${order.value.id}/checkout`, {
      method: 'POST',
      body: JSON.stringify({ method: payMethod.value }),
    })
    forgetGuestSession()
    router.push(`/t/${token.value}/receipt/${data.id}`)
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function quitOrder() {
  quitModalOpen.value = true
}

async function submitQuit(reason) {
  error.value = ''
  busy.value = true
  try {
    await guestApi(token.value, `/orders/${order.value.id}/quit`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
    quitModalOpen.value = false
    forgetGuestSession()
    order.value = null
    mode.value = 'draft'
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.steps {
  margin: 1rem 0;
  padding-left: 1.25rem;
}
.steps li {
  margin: 0.4rem 0;
}
</style>
