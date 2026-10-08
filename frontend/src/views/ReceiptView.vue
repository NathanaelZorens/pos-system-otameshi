<template>
  <div v-if="order" class="flex flex-col items-center gap-5 px-4 pb-8 pt-6">
    <div :class="[card, 'w-full max-w-[420px] text-center']">
      <p class="mb-2 font-display text-xs font-semibold uppercase tracking-[0.18em] text-gold-deep">Otameshi</p>
      <h1 class="m-0 mb-1.5 font-display text-ink">Receipt</h1>
      <p :class="[muted, 'mt-0']">
        <template v-if="isTakeout"> Takeout · {{ order.pickup_code || '—' }} · {{ order.created_at }} </template>
        <template v-else> Table {{ order.table_label }} · {{ order.created_at }} </template>
      </p>

      <div v-if="isTakeout" class="my-4 rounded-lg bg-gold-soft p-4">
        <div :class="muted">Pickup code</div>
        <div class="font-display text-[2.5rem] font-bold tracking-wide text-ink">{{ order.pickup_code || '—' }}</div>
      </div>

      <ul class="my-4 list-none p-0 text-left">
        <li
          v-for="it in order.items || []"
          :key="it.id"
          class="flex justify-between gap-4 border-b border-line py-1.5 text-[0.95rem]"
        >
          <span>{{ it.quantity }}× {{ it.name_snapshot }}</span>
          <strong>{{ formatMoney(it.line_total_cents) }}</strong>
        </li>
      </ul>
      <div class="mt-3 flex justify-between font-display text-[1.35rem] font-semibold">
        <span>Total</span>
        <strong>{{ formatMoney(order.total_cents) }}</strong>
      </div>
      <p v-if="order.payment" :class="muted">Paid via {{ order.payment.method }}</p>
    </div>
    <div class="no-print flex flex-wrap items-center gap-2">
      <button type="button" :class="btn" @click="printReceipt">Print</button>
      <RouterLink :to="back.to">
        <button type="button" :class="btnPrimary">{{ back.label }}</button>
      </RouterLink>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api, formatMoney } from '../api'
import { btn, btnPrimary, card, muted } from '../twUi'

const route = useRoute()
const order = ref(null)

const isTakeout = computed(() => order.value?.source === 'takeout')

const back = computed(() => {
  switch (route.query.from) {
    case 'reports':
      return { to: '/admin/reports', label: '← Reports' }
    case 'order':
      return { to: `/orders/${route.params.id}`, label: '← Order' }
    case 'takeout':
      return { to: '/takeout-queue', label: '← Queue' }
    default:
      return { to: '/tables', label: '← Tables' }
  }
})

onMounted(async () => {
  const data = await api(`/api/orders/${route.params.id}`)
  data.items = data.items || []
  order.value = data
})

function printReceipt() {
  window.print()
}
</script>
