<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <h1 class="m-0 font-display text-ink">New takeout</h1>
      <RouterLink to="/takeout-queue">← Queue</RouterLink>
    </div>
    <p :class="muted">Counter order — nothing is saved until you add the first item. Pay before kitchen starts.</p>

    <div :class="card">
      <div :class="row">
        <label>
          Customer name (optional)
          <input v-model="customerName" placeholder="Alex" />
        </label>
        <span :class="muted">Status: draft</span>
      </div>
      <h2>Current order</h2>
      <p :class="muted">No items yet. Choose qty and Add to start.</p>
      <p><strong>Total: {{ formatMoney(0) }}</strong></p>
    </div>

    <div :class="card">
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
          class="flex items-center justify-between gap-3 border-b border-line py-3.5"
          :class="{ 'opacity-55': m.is_sold_out }"
        >
          <div>
            <strong>{{ m.name }}</strong>
            <div :class="muted">{{ m.category || 'Uncategorized' }} · {{ formatMoney(m.price_cents) }}</div>
          </div>
          <div :class="row" v-if="!m.is_sold_out">
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
            <button type="button" :class="btnPrimary" @click="addItem(m)" :disabled="busy">
              {{ busy ? 'Starting…' : 'Add' }}
            </button>
            <button type="button" :class="btn" @click="toggleSoldOut(m)" :disabled="busy">Sold out</button>
          </div>
          <div :class="row" v-else>
            <span :class="muted">Sold out</span>
            <button type="button" :class="btn" @click="toggleSoldOut(m)" :disabled="busy">Mark available</button>
          </div>
        </div>
      </div>
    </div>

    <p v-if="error" class="text-danger">{{ error }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, formatMoney } from '../api'
import { categoryFiltersFromMenu, filterMenuByCategory } from '../menuFilters'
import { btn, btnPill, btnPillActive, btnPrimary, card, muted, row } from '../twUi'

const router = useRouter()

const menu = ref([])
const categoryFilter = ref('')
const customerName = ref('')
const error = ref('')
const menuError = ref('')
const busy = ref(false)
const addQtys = reactive({})

const categoryFilters = computed(() => categoryFiltersFromMenu(menu.value))
const filteredMenu = computed(() => filterMenuByCategory(menu.value, categoryFilter.value))

onMounted(loadMenu)

function addQty(menuItemId) {
  return addQtys[menuItemId] || 1
}

function setAddQty(menuItemId, value) {
  addQtys[menuItemId] = Math.max(1, parseInt(value, 10) || 1)
}

function bumpAddQty(menuItemId, delta) {
  addQtys[menuItemId] = Math.max(1, addQty(menuItemId) + delta)
}

async function loadMenu() {
  try {
    menu.value = (await api('/api/menu')) || []
  } catch (e) {
    menuError.value = e.message
  }
}

async function addItem(m) {
  error.value = ''
  busy.value = true
  try {
    const order = await api('/api/takeout/orders', {
      method: 'POST',
      body: JSON.stringify({
        customer_name: customerName.value.trim() || null,
        items: [{ menu_item_id: m.id, quantity: addQty(m.id) }],
      }),
    })
    router.replace(`/orders/${order.id}`)
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
</script>
