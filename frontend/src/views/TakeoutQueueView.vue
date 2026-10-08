<template>
  <div>
    <div class="mb-5 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Takeout queue</h1>
        <p :class="[muted, 'm-0']">
          Pay-first orders waiting for kitchen and pickup.
          <a :href="guestHref" target="_blank" rel="noopener">Guest link</a>
        </p>
      </div>
      <RouterLink to="/takeout-queue/new">
        <button type="button" :class="btnPrimary">New takeout</button>
      </RouterLink>
    </div>
    <div class="mb-4 mt-3 flex flex-wrap items-end gap-3" v-if="queue.length || searchQuery">
      <label class="max-w-xs flex-1">
        Find pickup
        <input v-model="searchQuery" type="search" placeholder="Code or name…" autocomplete="off" />
      </label>
      <button v-if="searchQuery" type="button" :class="btn" @click="searchQuery = ''">Clear</button>
    </div>
    <p v-if="error" class="text-danger">{{ error }}</p>
    <p v-if="!loading && !queue.length" :class="muted">No preparing or ready takeout orders.</p>
    <p v-else-if="!loading && queue.length && !filteredQueue.length" :class="muted">
      No match for “{{ searchQuery.trim() }}”.
    </p>

    <div class="flex flex-col gap-3">
      <div
        v-for="row in filteredQueue"
        :key="row.id"
        :class="[card, row.status === 'ready' ? 'border-success' : '']"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="flex flex-wrap items-start gap-4">
            <span
              class="inline-flex min-h-16 min-w-16 items-center justify-center rounded-md bg-gold-soft p-2 font-display text-[1.35rem] font-bold text-gold-deep"
            >
              {{ row.pickup_code || '—' }}
            </span>
            <div>
              <div :class="rowClass">
                <strong>{{ row.customer_name || 'Guest' }}</strong>
                <span
                  class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold"
                  :class="row.status === 'ready' ? 'bg-[#e8f0e2] text-success' : 'bg-gold-soft text-gold-deep'"
                >
                  {{ row.status }}
                </span>
              </div>
              <div :class="muted">
                {{ row.item_count }} item{{ row.item_count === 1 ? '' : 's' }}
                · {{ formatMoney(row.total_cents) }}
              </div>
            </div>
          </div>
          <div :class="rowClass">
            <button type="button" :class="btn" @click="toggleDetail(row.id)" :disabled="busy">
              {{ expanded === row.id ? 'Hide' : 'Details' }}
            </button>
            <button
              v-if="row.status === 'preparing'"
              type="button"
              :class="btnGold"
              :disabled="busy"
              @click="markReady(row.id)"
            >
              Mark ready
            </button>
            <button
              v-if="row.status === 'ready'"
              type="button"
              :class="btnPrimary"
              :disabled="busy"
              @click="markComplete(row.id)"
            >
              Picked up
            </button>
            <button type="button" :class="btnDanger" :disabled="busy" @click="openForceCancel(row)">
              Force cancel
            </button>
          </div>
        </div>

        <div v-if="expanded === row.id" class="mt-3 border-t border-line pt-3">
          <p v-if="detailLoading === row.id" :class="muted">Loading…</p>
          <template v-else-if="details[row.id]">
            <ul class="mb-4 list-none p-0">
              <li
                v-for="it in details[row.id].items || []"
                :key="it.id"
                class="border-b border-line py-2.5"
              >
                {{ it.quantity }}× {{ it.name_snapshot }} — {{ formatMoney(it.line_total_cents) }}
              </li>
            </ul>
            <p v-if="details[row.id].payment" :class="muted">
              Paid {{ details[row.id].payment.method }} · {{ details[row.id].payment.paid_at }}
            </p>
          </template>
        </div>
      </div>
    </div>

    <CancelReasonModal
      :open="cancelModalOpen"
      title="Force cancel takeout"
      message="Cancels a paid or unpaid takeout order. Enter a reason for ops."
      confirm-label="Force cancel"
      :busy="busy"
      @confirm="submitForceCancel"
      @cancel="cancelModalOpen = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, formatMoney } from '../api'
import CancelReasonModal from '../components/CancelReasonModal.vue'
import { btn, btnDanger, btnGold, btnPrimary, card, muted, row as rowClass } from '../twUi'

const route = useRoute()
const queue = ref([])
const details = reactive({})
const expanded = ref('')
const detailLoading = ref('')
const loading = ref(true)
const error = ref('')
const busy = ref(false)
const cancelModalOpen = ref(false)
const cancelTarget = ref(null)
const searchQuery = ref('')

const guestHref = computed(() => `${window.location.origin}/takeout`)

const filteredQueue = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return queue.value
  return queue.value.filter((row) => {
    const code = (row.pickup_code || '').toLowerCase()
    const name = (row.customer_name || '').toLowerCase()
    return code.includes(q) || name.includes(q)
  })
})

let pollTimer = null

onMounted(() => {
  load()
  pollTimer = setInterval(load, 4000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

watch(
  () => route.path,
  (path) => {
    if (path === '/takeout-queue') load()
  },
)

async function load() {
  try {
    queue.value = (await api('/api/takeout/queue')) || []
    error.value = ''
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function toggleDetail(id) {
  if (expanded.value === id) {
    expanded.value = ''
    return
  }
  expanded.value = id
  if (details[id]) return
  detailLoading.value = id
  try {
    const data = await api(`/api/orders/${id}`)
    data.items = data.items || []
    details[id] = data
  } catch (e) {
    error.value = e.message
  } finally {
    detailLoading.value = ''
  }
}

async function markReady(id) {
  error.value = ''
  busy.value = true
  try {
    await api(`/api/takeout/${id}/ready`, { method: 'POST' })
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function markComplete(id) {
  error.value = ''
  busy.value = true
  try {
    await api(`/api/takeout/${id}/complete`, { method: 'POST' })
    delete details[id]
    if (expanded.value === id) expanded.value = ''
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function openForceCancel(row) {
  cancelTarget.value = row
  cancelModalOpen.value = true
}

async function submitForceCancel(reason) {
  if (!cancelTarget.value) return
  error.value = ''
  busy.value = true
  try {
    await api(`/api/orders/${cancelTarget.value.id}/force-cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
    cancelModalOpen.value = false
    cancelTarget.value = null
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>
