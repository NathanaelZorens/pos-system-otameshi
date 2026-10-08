<template>
  <div class="guest">
    <header class="guest-header">
      <strong>Takeout</strong>
      <span class="muted">Pay first · pickup</span>
      <RouterLink to="/takeout">How it works</RouterLink>
      <RouterLink to="/member/login">Member login</RouterLink>
    </header>

    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else-if="mode === 'tracking' && order">
      <div class="card">
        <p class="muted">Pickup code</p>
        <h1 class="pickup-code">{{ order.pickup_code || '—' }}</h1>
        <p v-if="order.customer_name">{{ order.customer_name }}</p>
        <p class="status-line">
          <span :class="'badge-' + order.status">{{ statusLabel }}</span>
        </p>
        <p v-if="order.status === 'preparing'" class="muted">We're preparing your order. This page updates automatically.</p>
        <p v-else-if="order.status === 'ready'" class="muted">Ready for pickup — please come to the counter.</p>
        <p v-else-if="order.status === 'completed'" class="muted">Picked up. Thank you!</p>
        <p v-else-if="order.status === 'cancelled'" class="muted">This order was cancelled.</p>
        <hr />
        <ul class="order-lines">
          <li v-for="it in (order.items || [])" :key="it.id" class="order-line">
            <div>
              <strong>{{ it.name_snapshot }}</strong>
              <div class="muted">{{ it.quantity }}× · {{ formatMoney(it.line_total_cents) }}</div>
            </div>
          </li>
        </ul>
        <p><strong>Total: {{ formatMoney(order.total_cents) }}</strong></p>
        <p v-if="order.payment" class="muted">Paid via {{ order.payment.method }}</p>
        <div class="row" style="margin-top: 0.75rem">
          <RouterLink
            v-if="order.payment"
            :to="`/takeout/receipt/${order.id}`"
          >
            <button type="button" class="primary">Show receipt</button>
          </RouterLink>
          <button
            v-if="order.status === 'completed' || order.status === 'cancelled'"
            type="button"
            class="primary"
            @click="startFresh"
          >
            New takeout order
          </button>
        </div>
      </div>
    </template>

    <template v-else-if="mode === 'draft'">
      <div class="card">
        <label>
          Your name (optional)
          <input v-model="customerName" placeholder="Alex" style="width: 100%; margin-top: 0.25rem" />
        </label>
        <p class="muted" style="margin-top: 0.75rem">
          Pay with card / QR / wallet before kitchen starts. You'll get a pickup code.
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
        <p class="muted">
          Status: {{ order.status }}
          <span v-if="order.pickup_code"> · Code {{ order.pickup_code }}</span>
        </p>
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
            <div class="row qty-controls" v-if="canEdit && it.status === 'pending'">
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

        <div v-if="canEdit" class="row">
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
          <p v-else-if="!canPay" class="muted">Confirmed — pay to start kitchen, or ask staff to cancel.</p>
        </div>
      </div>

      <div v-if="canEdit" class="card">
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
      message="You can start a new takeout order afterward. Enter a reason."
      confirm-label="Quit"
      :busy="busy"
      @confirm="submitQuit"
      @cancel="quitModalOpen = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import {
  takeoutApi,
  formatMoney,
  getTakeoutOrderId,
  setTakeoutSession,
  clearTakeoutSession,
} from '../api'
import CancelReasonModal from '../components/CancelReasonModal.vue'
import { categoryFiltersFromMenu, filterMenuByCategory } from '../menuFilters'

const loading = ref(true)
const error = ref('')
const mode = ref('draft') // draft | order | tracking
const order = ref(null)
const menu = ref([])
const categoryFilter = ref('')
const customerName = ref('')
const payMethod = ref('qr')
const busy = ref(false)
const addQtys = reactive({})
const quitModalOpen = ref(false)

const canEdit = computed(() => order.value?.status === 'open')
const canPay = computed(
  () => canEdit.value && (order.value?.confirmed_count || 0) > 0 && !(order.value?.pending_count > 0),
)

const statusLabel = computed(() => {
  switch (order.value?.status) {
    case 'preparing':
      return 'Preparing'
    case 'ready':
      return 'Ready for pickup'
    case 'completed':
      return 'Completed'
    case 'cancelled':
      return 'Cancelled'
    default:
      return order.value?.status || ''
  }
})

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

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  stopPolling()
  if (!order.value?.id) return
  if (!['preparing', 'ready'].includes(order.value.status)) return
  pollTimer = setInterval(() => {
    refreshOrder()
  }, 2500)
}

async function refreshOrder() {
  if (!order.value?.id || busy.value) return
  try {
    const data = await takeoutApi(`/orders/${order.value.id}`)
    data.items = data.items || []
    order.value = data
    if (['preparing', 'ready', 'completed', 'cancelled'].includes(data.status)) {
      mode.value = 'tracking'
    }
    if (data.status === 'completed' || data.status === 'cancelled') {
      stopPolling()
      clearTakeoutSession()
    }
  } catch {
    // keep last known state
  }
}

function onVisibility() {
  if (document.visibilityState === 'visible' && ['preparing', 'ready'].includes(order.value?.status)) {
    refreshOrder()
  }
}

watch(
  () => order.value?.status,
  (status) => {
    if (['preparing', 'ready'].includes(status)) startPolling()
    else stopPolling()
  },
)

onMounted(() => {
  boot()
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisibility)
})

async function loadActiveOrder(orderId) {
  const data = await takeoutApi(`/orders/${orderId}`)
  data.items = data.items || []
  order.value = data
  if (['preparing', 'ready', 'completed', 'cancelled'].includes(data.status)) {
    mode.value = 'tracking'
    if (['preparing', 'ready'].includes(data.status)) startPolling()
    if (data.status === 'completed' || data.status === 'cancelled') clearTakeoutSession()
    return true
  }
  if (data.status === 'open') {
    mode.value = 'order'
    menu.value = (await takeoutApi('/menu')) || []
    return true
  }
  clearTakeoutSession()
  return false
}

async function boot() {
  loading.value = true
  error.value = ''
  try {
    const existing = getTakeoutOrderId()
    if (existing) {
      try {
        const ok = await loadActiveOrder(existing)
        if (ok) return
      } catch {
        clearTakeoutSession()
      }
    }
    mode.value = 'draft'
    menu.value = (await takeoutApi('/menu')) || []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function startFresh() {
  clearTakeoutSession()
  order.value = null
  mode.value = 'draft'
  error.value = ''
  boot()
}

async function startOrder(m) {
  error.value = ''
  busy.value = true
  try {
    const data = await takeoutApi('/orders', {
      method: 'POST',
      body: JSON.stringify({
        customer_name: customerName.value.trim() || null,
        items: [{ menu_item_id: m.id, quantity: addQty(m.id) }],
      }),
    })
    data.items = data.items || []
    if (data.guest_claim) setTakeoutSession(data.id, data.guest_claim)
    else setTakeoutSession(data.id, '')
    order.value = data
    mode.value = 'order'
    addQtys[m.id] = 1
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function addItem(menuItemId) {
  error.value = ''
  busy.value = true
  try {
    const data = await takeoutApi(`/orders/${order.value.id}/items`, {
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
    const data = await takeoutApi(`/orders/${order.value.id}/items/${item.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ quantity }),
    })
    data.items = data.items || []
    order.value = data
    if (data.status === 'cancelled') {
      clearTakeoutSession()
      order.value = null
      mode.value = 'draft'
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
    const data = await takeoutApi(`/orders/${order.value.id}/confirm`, { method: 'POST' })
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
    const data = await takeoutApi(`/orders/${order.value.id}/checkout`, {
      method: 'POST',
      body: JSON.stringify({ method: payMethod.value }),
    })
    data.items = data.items || []
    order.value = data
    mode.value = 'tracking'
    startPolling()
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
    await takeoutApi(`/orders/${order.value.id}/quit`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
    quitModalOpen.value = false
    clearTakeoutSession()
    order.value = null
    mode.value = 'draft'
    menu.value = (await takeoutApi('/menu')) || []
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.pickup-code {
  font-size: 2.5rem;
  letter-spacing: 0.2em;
  margin: 0.25rem 0 0.5rem;
}
.status-line {
  margin: 0.75rem 0;
}
.badge-preparing {
  color: var(--gold-deep);
  font-weight: 600;
}
.badge-ready {
  color: var(--success);
  font-weight: 700;
}
.badge-completed,
.badge-cancelled {
  color: var(--ink-muted);
  font-weight: 600;
}
</style>
