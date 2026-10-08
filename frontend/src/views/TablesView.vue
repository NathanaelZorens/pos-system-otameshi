<template>
  <div>
    <div class="mb-5">
      <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Tables</h1>
      <p :class="muted">
        Pick a table to order as staff, or open the guest QR link for customers. After pay the table stays busy until
        Clear table.
      </p>
    </div>
    <p v-if="error" class="text-danger">{{ error }}</p>
    <div class="grid gap-3.5 [grid-template-columns:repeat(auto-fill,minmax(160px,1fr))]">
      <div v-for="t in tables" :key="t.id" class="flex flex-col gap-1.5">
        <button
          type="button"
          class="flex min-h-[120px] w-full flex-col items-start justify-start gap-1.5 rounded-lg border border-line bg-surface-raised p-4 text-left focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold-deep"
          :class="tableBtnAccent(t)"
          @click="openTable($event, t)"
        >
          <div class="flex w-full items-center justify-between gap-2">
            <strong class="font-display text-xl font-semibold">Table {{ t.label }}</strong>
            <span
              v-if="t.open_order"
              class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold"
              :class="t.open_order.status === 'paid' ? 'bg-gold-soft text-gold-deep' : 'bg-[#e8f0e2] text-success'"
            >
              {{ t.open_order.status === 'paid' ? 'paid' : 'busy' }}
            </span>
          </div>
          <span
            v-if="t.open_order"
            class="text-sm font-medium"
            :class="t.open_order.status === 'paid' ? 'text-gold-deep' : 'text-success'"
          >
            {{ t.open_order.status === 'paid' ? 'paid · clear seats' : t.open_order.status }}
            · {{ formatMoney(t.open_order.total_cents) }}
          </span>
          <span v-else :class="muted">Free</span>
        </button>
        <div v-if="t.guest_token" class="flex justify-start gap-2.5 pl-0.5 text-sm">
          <a
            class="text-brown"
            :href="guestHref(t)"
            target="_blank"
            rel="noopener"
            @click.stop
          >
            Guest link
          </a>
          <button
            type="button"
            class="cursor-pointer border-0 bg-transparent p-0 text-sm font-semibold text-brown"
            @click.stop="showQr(t)"
          >
            QR
          </button>
        </div>
      </div>
    </div>

    <div v-if="qrTable" :class="[card, 'sticky bottom-4']">
      <h2>Guest QR — Table {{ qrTable.label }}</h2>
      <p :class="muted">Customers scan this to order (no login).</p>
      <img class="my-3 block rounded-md bg-white" :src="qrImageUrl" alt="Guest QR code" width="220" height="220" />
      <p>
        <code class="mb-3 block break-all text-sm text-ink-soft">{{ guestHref(qrTable) }}</code>
      </p>
      <div :class="row">
        <button type="button" :class="btn" @click="copyLink(qrTable)">Copy link</button>
        <button type="button" :class="btnPrimary" @click="qrTable = null">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, formatMoney } from '../api'
import { btn, btnPrimary, card, muted, row } from '../twUi'

const route = useRoute()
const router = useRouter()
const tables = ref([])
const error = ref('')
const qrTable = ref(null)

const qrImageUrl = computed(() => {
  if (!qrTable.value) return ''
  const data = encodeURIComponent(guestHref(qrTable.value))
  return `https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${data}`
})

onMounted(load)
watch(
  () => route.path,
  (path) => {
    if (path === '/tables') load()
  },
)

function guestHref(t) {
  return `${window.location.origin}/t/${t.guest_token}`
}

function tableBtnAccent(t) {
  if (!t.open_order) return ''
  if (t.open_order.status === 'paid') return 'border-gold shadow-[inset_4px_0_0_#c49a3c]'
  return 'border-success shadow-[inset_4px_0_0_#4f6b3a]'
}

async function load() {
  try {
    tables.value = (await api('/api/dining-tables')) || []
  } catch (e) {
    error.value = e.message
  }
}

function openTable(event, t) {
  event.currentTarget?.blur()
  error.value = ''
  if (t.open_order) {
    router.push(`/orders/${t.open_order.id}`)
    return
  }
  router.push({
    path: `/tables/${t.id}/order`,
    query: { label: t.label },
  })
}

function showQr(t) {
  qrTable.value = t
}

async function copyLink(t) {
  try {
    await navigator.clipboard.writeText(guestHref(t))
  } catch {
    error.value = 'Could not copy link'
  }
}
</script>
