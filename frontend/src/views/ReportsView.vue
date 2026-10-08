<template>
  <div>
    <div class="mb-5">
      <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Reports</h1>
      <p :class="muted">Based on payments (not table clear).</p>
    </div>
    <div :class="[card, rowEnd]">
      <label>From <input v-model="from" type="date" /></label>
      <label>To <input v-model="to" type="date" /></label>
      <button :class="btnPrimary" type="button" @click="load">Refresh</button>
    </div>

    <div
      v-if="summary"
      :class="[card, 'grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]']"
    >
      <div>
        <div :class="[muted, 'mb-1']">Orders</div>
        <strong class="font-display text-[1.6rem] font-semibold">{{ summary.order_count }}</strong>
      </div>
      <div>
        <div :class="[muted, 'mb-1']">Total sales</div>
        <strong class="font-display text-[1.6rem] font-semibold">{{ formatMoney(summary.total_cents) }}</strong>
      </div>
      <div>
        <div :class="[muted, 'mb-1']">Range</div>
        <strong class="font-display text-[1.6rem] font-semibold">{{ summary.from }} → {{ summary.to }}</strong>
      </div>
    </div>

    <div :class="card">
      <h2>Transactions</h2>
      <p v-if="!txns.length" :class="muted">No paid orders in this range.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-[0.95rem]">
          <thead>
            <tr>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Paid at</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Place</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Waiter</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Method</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Total</th>
              <th class="border-b border-line px-2 py-2.5"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in txns" :key="t.order_id" class="hover:bg-surface">
              <td class="border-b border-line px-2 py-2.5 align-top">{{ t.paid_at }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ t.table_label }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ t.waiter_name }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ t.method }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ formatMoney(t.total_cents) }}
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top">
                <RouterLink :to="{ path: `/receipt/${t.order_id}`, query: { from: 'reports' } }">Receipt</RouterLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div :class="card">
      <h2>Item sales</h2>
      <p v-if="!items.length" :class="muted">No item sales in this range.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-[0.95rem]">
          <thead>
            <tr>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Item</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Qty</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Revenue</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, i) in items" :key="it.name" class="hover:bg-surface">
              <td class="border-b border-line px-2 py-2.5 align-top">
                {{ it.name }}
                <span v-if="i === 0" :class="muted"> · most sold</span>
                <span v-else-if="i === items.length - 1 && items.length > 1" :class="muted"> · least sold</span>
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ it.quantity }}
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ formatMoney(it.revenue_cents) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <p v-if="error" class="text-danger">{{ error }}</p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { api, formatMoney } from '../api'
import { btnPrimary, card, muted, rowEnd } from '../twUi'

const from = ref(new Date().toISOString().slice(0, 10))
const to = ref(from.value)
const summary = ref(null)
const items = ref([])
const txns = ref([])
const error = ref('')

onMounted(load)

async function load() {
  error.value = ''
  try {
    const q = `from=${from.value}&to=${to.value}`
    ;[summary.value, items.value, txns.value] = await Promise.all([
      api(`/api/reports/summary?${q}`),
      api(`/api/reports/items?${q}`),
      api(`/api/reports/transactions?${q}`),
    ])
    items.value = items.value || []
    txns.value = txns.value || []
  } catch (e) {
    error.value = e.message
  }
}
</script>
