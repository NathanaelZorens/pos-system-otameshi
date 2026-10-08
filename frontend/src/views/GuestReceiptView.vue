<template>
  <div class="guest" v-if="order">
    <div class="card" style="text-align: center">
      <p class="eyebrow">Otameshi</p>
      <h1 style="margin: 0 0 0.35rem">You’re paid</h1>
      <p class="muted" style="margin-top: 0">
        Table {{ order.table_label }}
        <span v-if="order.payment"> · {{ order.payment.method }}</span>
        · {{ order.created_at }}
      </p>

      <ul class="receipt-lines">
        <li v-for="it in (order.items || [])" :key="it.id">
          <span>{{ it.quantity }}× {{ it.name_snapshot }}</span>
          <strong>{{ formatMoney(it.line_total_cents) }}</strong>
        </li>
      </ul>
      <p style="font-family: var(--font-display); font-size: 1.35rem; font-weight: 600; margin: 0.75rem 0">
        Total {{ formatMoney(order.total_cents) }}
      </p>
      <p class="muted">Thank you — show this if staff ask. Want something else? Ask a waiter (table stays yours until they clear it).</p>

      <div class="row" style="margin-top: 1.25rem; justify-content: center">
        <button type="button" class="primary" @click="downloadOrderReceipt(order)">Download receipt</button>
      </div>
      <p class="muted" style="margin-top: 0.75rem">Downloads a PDF slip you can keep or share.</p>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
  <p v-else-if="error" class="error">{{ error }}</p>
  <p v-else class="muted">Loading…</p>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { guestApi, formatMoney } from '../api'
import { downloadOrderReceipt } from '../receiptDownload'

const route = useRoute()
const token = computed(() => route.params.token)
const order = ref(null)
const error = ref('')

onMounted(async () => {
  try {
    const data = await guestApi(token.value, `/orders/${route.params.orderId}`)
    data.items = data.items || []
    order.value = data
  } catch (e) {
    error.value = e.message
  }
})
</script>

<style scoped>
.receipt-lines {
  list-style: none;
  margin: 1rem 0 0;
  padding: 0;
  text-align: left;
}
.receipt-lines li {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.45rem 0;
  border-bottom: 1px solid var(--line);
  font-size: 0.95rem;
}
</style>
