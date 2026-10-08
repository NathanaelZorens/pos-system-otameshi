<template>
  <div>
    <div class="mb-5">
      <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Cancelled</h1>
      <p :class="muted">Quit and force-cancel history with reasons.</p>
    </div>

    <div :class="[card, rowEnd]">
      <label>From <input v-model="from" type="date" /></label>
      <label>To <input v-model="to" type="date" /></label>
      <button :class="btnPrimary" type="button" @click="load">Refresh</button>
    </div>

    <div
      v-if="data"
      :class="[card, 'grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]']"
    >
      <div>
        <div :class="[muted, 'mb-1']">Total cancelled</div>
        <strong class="font-display text-[1.6rem] font-semibold">{{ data.total_count }}</strong>
      </div>
      <div>
        <div :class="[muted, 'mb-1']">Range</div>
        <strong class="font-display text-[1.6rem] font-semibold">{{ data.from }} → {{ data.to }}</strong>
      </div>
    </div>

    <div :class="card">
      <h2>List</h2>
      <p v-if="!orders.length" :class="muted">No cancelled orders in this range.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-[0.95rem]">
          <thead>
            <tr>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Cancelled at</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Reason</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Table</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Source</th>
              <th class="border-b border-line px-2 py-2.5 text-left font-semibold text-ink-muted">Waiter</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Confirmed</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Pending</th>
              <th class="border-b border-line px-2 py-2.5 text-right font-semibold text-ink-muted">Total</th>
              <th class="border-b border-line px-2 py-2.5"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.order_id" class="hover:bg-surface">
              <td class="border-b border-line px-2 py-2.5 align-top">{{ o.cancelled_at }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ o.cancel_reason || '—' }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ o.table_label }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ o.source }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top">{{ o.waiter_name }}</td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ o.confirmed_count }}
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ o.pending_count }}
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top text-right font-display font-semibold whitespace-nowrap">
                {{ formatMoney(o.total_cents) }}
              </td>
              <td class="border-b border-line px-2 py-2.5 align-top">
                <RouterLink :to="{ path: `/orders/${o.order_id}`, query: { from: 'cancelled' } }">View</RouterLink>
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
import { computed, onMounted, ref } from 'vue'
import { api, formatMoney } from '../api'
import { btnPrimary, card, muted, rowEnd } from '../twUi'

const from = ref(new Date().toISOString().slice(0, 10))
const to = ref(from.value)
const data = ref(null)
const error = ref('')

const orders = computed(() => data.value?.orders || [])

onMounted(load)

async function load() {
  error.value = ''
  try {
    data.value = await api(`/api/reports/cancelled?from=${from.value}&to=${to.value}`)
  } catch (e) {
    error.value = e.message
  }
}
</script>
