<template>
  <span
    v-if="label"
    class="inline-flex shrink-0 items-center justify-center rounded-full bg-discount-pink px-2 py-0.5 text-xs font-medium leading-none text-white"
  >
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { discountBadgeText } from '../discountBadge'

const props = defineProps({
  listCents: { type: Number, default: 0 },
  unitCents: { type: Number, default: 0 },
  /** Prefer API snapshot when it already looks like −10%=¥558 */
  storedLabel: { type: String, default: '' },
  text: { type: String, default: '' },
})

const label = computed(() => {
  if (props.text) return props.text
  return discountBadgeText(props.listCents, props.unitCents, props.storedLabel)
})
</script>
