<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <h1 class="m-0 font-display text-ink">New takeout</h1>
      <RouterLink to="/takeout-queue">← Queue</RouterLink>
    </div>
    <p :class="muted">Counter order — nothing is saved until you add the first item. Pay before kitchen starts.</p>

    <div class="grid gap-4 lg:grid-cols-[minmax(300px,0.42fr)_minmax(0,1fr)] lg:items-start">
      <div :class="[card, 'lg:sticky lg:top-3 lg:mb-0']">
        <div :class="row">
          <label class="min-w-48 flex-1">
            Customer name (optional)
            <input v-model="customerName" placeholder="Alex" />
          </label>
          <span :class="muted">Status: draft</span>
        </div>
        <h2>Current order</h2>
        <p :class="muted">No items yet. Choose qty and Add to start.</p>
        <p class="font-display text-xl font-semibold">Total: {{ formatMoney(0) }}</p>
      </div>

      <div :class="[card, 'lg:mb-0']">
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
    </div>

    <p v-if="error" class="mt-3 text-danger">{{ error }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, formatMoney } from '../api'
import DiscountBadge from '../components/DiscountBadge.vue'
import { hasDiscount } from '../discountBadge'
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
