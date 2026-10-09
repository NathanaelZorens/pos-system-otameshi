<template>
  <div v-if="order">
    <div class="no-print mb-3 flex flex-wrap items-center justify-between gap-2">
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" :class="btn" @click="goBack" :disabled="busy">
          {{ isTakeout ? '← Queue' : '← Tables' }}
        </button>
        <h1 v-if="isTakeout" class="m-0 font-display text-ink">
          Takeout
          <span
            v-if="order.pickup_code"
            class="pickup-inline ml-2 font-display font-bold tracking-widest text-gold-deep"
          >
            {{ order.pickup_code }}
          </span>
        </h1>
        <h1 v-else class="m-0 font-display text-ink">Table {{ order.table_label }}</h1>
        <span
          class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold"
          :class="statusPillClass"
        >
          {{ order.status }}
        </span>
      </div>
    </div>

    <div
      class="grid gap-4"
      :class="canEditLines ? 'lg:grid-cols-[minmax(300px,0.42fr)_minmax(0,1fr)] lg:items-start' : ''"
    >
      <div
        class="rounded-xl border border-line bg-surface-raised px-5 py-[1.15rem] lg:sticky lg:top-3"
        :class="order.status === 'paid' ? 'border-gold' : ''"
      >
        <div class="flex flex-wrap items-center gap-2">
          <label class="min-w-48 flex-1">
            Customer name (optional)
            <input v-model="customerName" placeholder="Alex" />
          </label>
        </div>
        <h2>Current order</h2>
        <p v-if="!(order.items || []).length" class="text-sm text-ink-muted">No items yet.</p>
        <ul class="mb-4 list-none p-0">
          <li
            v-for="it in order.items || []"
            :key="it.id"
            class="border-b border-line py-2.5"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <strong>{{ it.name_snapshot }}</strong>
                <div class="mt-1 flex flex-wrap items-center gap-1.5 text-sm text-ink-muted">
                  <template v-if="hasDiscount(it.list_unit_price_cents, it.unit_price_cents)">
                    <span class="line-through opacity-70">{{ formatMoney(it.list_unit_price_cents) }}</span>
                    <span>{{ formatMoney(it.unit_price_cents) }} each</span>
                    <DiscountBadge
                      :list-cents="it.list_unit_price_cents"
                      :unit-cents="it.unit_price_cents"
                      :stored-label="it.discount_label || ''"
                    />
                  </template>
                  <template v-else>
                    <span>{{ formatMoney(it.unit_price_cents) }} each</span>
                  </template>
                  <span
                    :class="
                      it.status === 'confirmed' ? 'font-semibold text-success' : 'font-semibold text-gold-deep'
                    "
                  >
                    · {{ it.status }}
                  </span>
                </div>
              </div>
              <strong class="shrink-0 font-display">{{ formatMoney(it.line_total_cents) }}</strong>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-2" v-if="canEditLines && it.status === 'pending'">
              <button type="button" :class="btn" @click="changeQty(it, it.quantity - 1)" :disabled="busy">−</button>
              <span class="min-w-6 text-center font-semibold">{{ it.quantity }}</span>
              <button type="button" :class="btn" @click="changeQty(it, it.quantity + 1)" :disabled="busy">+</button>
              <button type="button" :class="btnDanger" @click="changeQty(it, 0)" :disabled="busy">Remove</button>
            </div>
            <div v-else-if="!(canEditLines && it.status === 'pending')" class="mt-1 text-sm font-semibold text-ink-soft">
              {{ it.quantity }}×
            </div>
          </li>
        </ul>
        <p class="font-display text-xl font-semibold">
          Total: {{ formatMoney(order.total_cents) }}
        </p>
        <p v-if="canEditLines && order.pending_count" class="text-sm text-ink-muted">
          {{ order.pending_count }} pending — confirm before paying.
        </p>

        <div v-if="canEditLines" class="no-print mt-3 flex flex-wrap items-center gap-2">
          <button
            v-if="order.pending_count > 0"
            type="button"
            :class="btnPrimary"
            @click="confirmItems"
            :disabled="busy"
          >
            Confirm
          </button>
          <button v-if="canMarkServed" type="button" :class="btn" @click="markServed" :disabled="busy">
            Mark served
          </button>
          <template v-if="canPay">
            <select v-model="payMethod">
              <option value="cash">Cash</option>
              <option value="card">Card</option>
              <option value="qr">QR</option>
              <option value="wallet">Wallet</option>
            </select>
            <button :class="btnGold" type="button" @click="checkout" :disabled="busy">Pay</button>
          </template>
          <button
            v-if="!(order.confirmed_count > 0)"
            :class="btnDanger"
            type="button"
            @click="quitOrder"
            :disabled="busy"
          >
            Quit
          </button>
          <button v-else :class="btnDanger" type="button" @click="forceCancelOrder" :disabled="busy">
            Force cancel
          </button>
        </div>
        <div
          v-else-if="isTakeout && (order.status === 'preparing' || order.status === 'ready')"
          class="no-print mt-3 flex flex-wrap items-center gap-2"
        >
          <p class="text-sm text-ink-muted">
            Paid — on the takeout queue as <strong>{{ order.status }}</strong>
            <span v-if="order.pickup_code"> · code {{ order.pickup_code }}</span>
          </p>
          <RouterLink to="/takeout-queue"><button type="button" :class="btnPrimary">Open queue</button></RouterLink>
          <button type="button" :class="btnDanger" @click="forceCancelOrder" :disabled="busy">Force cancel</button>
        </div>
        <div v-else-if="order.status === 'paid'" class="no-print mt-3 flex flex-wrap items-center gap-2">
          <p class="text-sm text-ink-muted">
            Paid — table still occupied until you clear it
            <span v-if="order.served_at"> · served</span>
            <span v-else> · food not marked served yet</span>
          </p>
          <button v-if="canMarkServed" type="button" :class="btn" @click="markServed" :disabled="busy">
            Mark served
          </button>
          <button type="button" :class="btnPrimary" @click="clearTable" :disabled="busy">Clear table</button>
          <RouterLink :to="{ path: `/receipt/${order.id}`, query: { from: 'order' } }">
            <button type="button" :class="btn">View receipt</button>
          </RouterLink>
        </div>
        <div v-else-if="order.status === 'cleared'" class="no-print mt-3 flex flex-wrap items-center gap-2">
          <p class="text-sm text-ink-muted">Table cleared</p>
          <RouterLink :to="{ path: `/receipt/${order.id}`, query: { from: 'order' } }">
            <button type="button" :class="btnPrimary">View receipt</button>
          </RouterLink>
          <RouterLink to="/tables">← Tables</RouterLink>
        </div>
        <div v-else-if="order.status === 'completed'" class="no-print mt-3 flex flex-wrap items-center gap-2">
          <p class="text-sm text-ink-muted">Picked up / completed</p>
          <RouterLink to="/takeout-queue">← Queue</RouterLink>
        </div>
        <div v-else-if="order.status === 'cancelled'" class="no-print mt-3 flex flex-wrap items-center gap-2">
          <p class="text-sm text-ink-muted">
            Cancelled
            <span v-if="order.cancel_reason"> · {{ order.cancel_reason }}</span>
            <span v-if="order.cancelled_at"> · {{ order.cancelled_at }}</span>
          </p>
          <RouterLink :to="cancelledBack.to">{{ cancelledBack.label }}</RouterLink>
        </div>
      </div>

      <div
        v-if="canEditLines"
        class="no-print rounded-xl border border-line bg-surface-raised px-5 py-[1.15rem]"
      >
        <h2>Menu</h2>
        <p v-if="menuError" class="text-danger">{{ menuError }}</p>
        <div class="mb-3 flex flex-wrap gap-1.5" v-if="categoryFilters.length">
          <button type="button" :class="!categoryFilter ? btnPillActive : btnPill" @click="categoryFilter = ''">
            All
          </button>
          <button
            v-for="c in categoryFilters"
            :key="c.id"
            type="button"
            :class="categoryFilter === c.id ? btnPillActive : btnPill"
            @click="categoryFilter = c.id"
          >
            {{ c.name }}
          </button>
        </div>
        <div class="grid gap-2">
          <div
            v-for="m in filteredMenu"
            :key="m.id"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-line py-3.5"
            :class="{ 'opacity-55': m.is_sold_out }"
          >
            <div class="min-w-0 flex-1">
              <strong>{{ m.name }}</strong>
              <div class="mt-1 flex flex-wrap items-center gap-1.5 text-sm text-ink-muted">
                <span>{{ m.category || 'Uncategorized' }}</span>
                <template v-if="hasDiscount(m.price_cents, m.unit_price_cents ?? m.price_cents)">
                  <span class="line-through opacity-70">{{ formatMoney(m.price_cents) }}</span>
                  <span class="font-semibold text-ink">{{ formatMoney(m.unit_price_cents) }}</span>
                  <DiscountBadge
                    :list-cents="m.price_cents"
                    :unit-cents="m.unit_price_cents"
                    :stored-label="m.discount_label || ''"
                  />
                </template>
                <template v-else>
                  <span>· {{ formatMoney(m.price_cents) }}</span>
                </template>
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-2" v-if="!m.is_sold_out">
              <button type="button" :class="btn" @click="bumpAddQty(m.id, -1)" :disabled="busy || addQty(m.id) <= 1">
                −
              </button>
              <input
                class="qty-input !mt-0 !w-[3.25rem] max-w-[3.25rem] text-center"
                type="number"
                min="1"
                :value="addQty(m.id)"
                @change="setAddQty(m.id, $event.target.value)"
              />
              <button type="button" :class="btn" @click="bumpAddQty(m.id, 1)" :disabled="busy">+</button>
              <button type="button" :class="btnPrimary" @click="addItem(m.id)" :disabled="busy">Add</button>
              <button type="button" :class="btn" @click="toggleSoldOut(m)" :disabled="busy">Sold out</button>
            </div>
            <div class="flex flex-wrap items-center gap-2" v-else>
              <span class="text-sm text-ink-muted">Sold out</span>
              <button type="button" :class="btn" @click="toggleSoldOut(m)" :disabled="busy">Mark available</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <p v-if="error" class="mt-3 text-danger">{{ error }}</p>

    <CancelReasonModal
      :open="cancelModal.open"
      :title="cancelModal.title"
      :message="cancelModal.message"
      :confirm-label="cancelModal.confirmLabel"
      :busy="busy"
      @confirm="submitCancel"
      @cancel="closeCancelModal"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, formatMoney } from '../api'
import CancelReasonModal from '../components/CancelReasonModal.vue'
import DiscountBadge from '../components/DiscountBadge.vue'
import { hasDiscount } from '../discountBadge'
import { categoryFiltersFromMenu, filterMenuByCategory } from '../menuFilters'
import { btn, btnDanger, btnGold, btnPill, btnPillActive, btnPrimary } from '../twUi'

const route = useRoute()
const router = useRouter()
const order = ref(null)
const menu = ref([])
const categoryFilter = ref('')
const payMethod = ref('cash')
const customerName = ref('')
const error = ref('')
const menuError = ref('')
const busy = ref(false)
const addQtys = reactive({})
const cancelModal = reactive({
  open: false,
  mode: '',
  title: '',
  message: '',
  confirmLabel: '',
})

const isTakeout = computed(() => order.value?.source === 'takeout')

const statusPillClass = computed(() => {
  const s = order.value?.status
  if (s === 'paid') return 'bg-gold-soft text-gold-deep'
  if (s === 'cancelled') return 'bg-danger-soft text-danger'
  return 'bg-[#e8f0e2] text-success'
})

const canEditLines = computed(() => {
  if (!order.value) return false
  if (isTakeout.value) return order.value.status === 'open'
  return ['open', 'preparing', 'served'].includes(order.value.status)
})

const canPay = computed(
  () =>
    canEditLines.value &&
    (order.value?.confirmed_count || 0) > 0 &&
    !(order.value?.pending_count > 0),
)

const canMarkServed = computed(() => {
  if (!order.value || isTakeout.value || order.value.served_at) return false
  return order.value.status === 'preparing' || order.value.status === 'paid'
})

const categoryFilters = computed(() => categoryFiltersFromMenu(menu.value))
const filteredMenu = computed(() => filterMenuByCategory(menu.value, categoryFilter.value))

const cancelledBack = computed(() => {
  if (route.query.from === 'cancelled') {
    return { to: '/admin/cancelled', label: '← Back to cancelled' }
  }
  if (isTakeout.value) {
    return { to: '/takeout-queue', label: '← Queue' }
  }
  return { to: '/tables', label: 'Back to tables' }
})

onMounted(async () => {
  await Promise.all([loadOrder(), loadMenu()])
})

function addQty(menuItemId) {
  return addQtys[menuItemId] || 1
}

function setAddQty(menuItemId, value) {
  const n = Math.max(1, parseInt(value, 10) || 1)
  addQtys[menuItemId] = n
}

function bumpAddQty(menuItemId, delta) {
  addQtys[menuItemId] = Math.max(1, addQty(menuItemId) + delta)
}

async function loadOrder() {
  try {
    const data = await api(`/api/orders/${route.params.id}`)
    data.items = data.items || []
    order.value = data
    customerName.value = order.value.customer_name || ''
  } catch (e) {
    error.value = e.message
  }
}

async function loadMenu() {
  try {
    menu.value = (await api('/api/menu')) || []
  } catch (e) {
    menuError.value = e.message
  }
}

async function addItem(menuItemId) {
  error.value = ''
  busy.value = true
  try {
    const data = await api(`/api/orders/${route.params.id}/items`, {
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
    const data = await api(`/api/orders/${route.params.id}/items/${item.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ quantity }),
    })
    data.items = data.items || []
    order.value = data
    if (data.status === 'cancelled') {
      goBack()
    }
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function toggleSoldOut(m) {
  menuError.value = ''
  try {
    await api(`/api/menu/${m.id}/sold-out`, {
      method: 'PATCH',
      body: JSON.stringify({ sold_out: !m.is_sold_out }),
    })
    await loadMenu()
  } catch (e) {
    menuError.value = e.message
  }
}

async function confirmItems() {
  error.value = ''
  busy.value = true
  try {
    const data = await api(`/api/orders/${route.params.id}/confirm`, { method: 'POST' })
    data.items = data.items || []
    order.value = data
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function markServed() {
  error.value = ''
  busy.value = true
  try {
    const data = await api(`/api/orders/${route.params.id}/serve`, { method: 'POST' })
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
    order.value = await api(`/api/orders/${route.params.id}/checkout`, {
      method: 'POST',
      body: JSON.stringify({ method: payMethod.value }),
    })
    if (order.value.source === 'takeout') {
      router.push('/takeout-queue')
      return
    }
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function clearTable() {
  error.value = ''
  busy.value = true
  try {
    order.value = await api(`/api/orders/${route.params.id}/clear-table`, { method: 'POST' })
    router.push('/tables')
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function goBack() {
  router.push(isTakeout.value ? '/takeout-queue' : '/tables')
}

async function quitOrder() {
  cancelModal.mode = 'quit'
  cancelModal.title = 'Quit order'
  cancelModal.message = 'Nothing has been confirmed yet. Enter a reason to discard this check.'
  cancelModal.confirmLabel = 'Quit'
  cancelModal.open = true
}

async function forceCancelOrder() {
  cancelModal.mode = 'force'
  cancelModal.title = 'Force cancel'
  cancelModal.message =
    'Items may already be confirmed or in progress. Use only for rare cases. Enter a reason.'
  cancelModal.confirmLabel = 'Force cancel'
  cancelModal.open = true
}

function closeCancelModal() {
  cancelModal.open = false
  cancelModal.mode = ''
}

async function submitCancel(reason) {
  error.value = ''
  busy.value = true
  try {
    const path =
      cancelModal.mode === 'force'
        ? `/api/orders/${route.params.id}/force-cancel`
        : `/api/orders/${route.params.id}/quit`
    await api(path, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
    closeCancelModal()
    goBack()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.pickup-inline {
  letter-spacing: 0.12em;
}
</style>
