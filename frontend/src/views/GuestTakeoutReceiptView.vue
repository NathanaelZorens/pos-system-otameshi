<template>
  <div class="guest">
    <div v-if="order" class="auth-shell" style="min-height: auto; padding: 1rem 0">
      <div class="card auth-card" style="text-align: center">
        <p class="eyebrow">Otameshi</p>
        <h1 style="margin: 0 0 0.35rem">You’re paid</h1>
        <p class="muted" style="margin-top: 0">
          Takeout · {{ order.pickup_code || '—' }}
          <span v-if="order.payment"> · {{ order.payment.method }}</span>
        </p>
        <div class="pickup-box">
          <div class="muted">Pickup code</div>
          <div class="pickup-code">{{ order.pickup_code || '—' }}</div>
        </div>
        <p style="font-family: var(--font-display); font-size: 1.25rem; font-weight: 600">
          Total {{ formatMoney(order.total_cents) }}
        </p>
      </div>
      <div class="row" style="flex-direction: column; width: 100%; max-width: 420px">
        <button type="button" class="primary" style="width: 100%; height: 48px" @click="downloadOrderReceipt(order)">
          Download receipt
        </button>
        <RouterLink to="/takeout/order" class="muted" style="text-align: center">← Back to order</RouterLink>
      </div>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-else-if="!order" class="muted">Loading…</p>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { takeoutApi, formatMoney } from '../api'
import { downloadOrderReceipt } from '../receiptDownload'

const route = useRoute()
const order = ref(null)
const error = ref('')

onMounted(async () => {
  try {
    const data = await takeoutApi(`/orders/${route.params.orderId}`)
    data.items = data.items || []
    order.value = data
  } catch (e) {
    error.value = e.message
  }
})
</script>

<style scoped>
.pickup-box {
  margin: 1rem 0;
  padding: 1rem;
  border-radius: var(--r-lg);
  background: var(--gold-soft);
}
</style>
