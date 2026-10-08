<template>
  <div v-if="open" class="modal-backdrop" @click.self="onCancel">
    <div class="modal card" role="dialog" aria-modal="true">
      <h2>{{ title }}</h2>
      <p v-if="message" class="muted">{{ message }}</p>
      <label>
        Reason
        <textarea
          v-model="text"
          rows="3"
          placeholder="Why is this order being cancelled?"
          style="width: 100%; margin-top: 0.25rem"
          :disabled="busy"
        />
      </label>
      <p v-if="localError" class="error">{{ localError }}</p>
      <div class="row" style="margin-top: 0.75rem">
        <button type="button" class="danger" :disabled="busy" @click="onConfirm">
          {{ confirmLabel }}
        </button>
        <button type="button" :disabled="busy" @click="onCancel">Back</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Cancel order' },
  message: { type: String, default: '' },
  confirmLabel: { type: String, default: 'Confirm' },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['confirm', 'cancel'])

const text = ref('')
const localError = ref('')

watch(
  () => props.open,
  (v) => {
    if (v) {
      text.value = ''
      localError.value = ''
    }
  },
)

function onConfirm() {
  const reason = text.value.trim()
  if (!reason) {
    localError.value = 'Please enter a reason.'
    return
  }
  localError.value = ''
  emit('confirm', reason)
}

function onCancel() {
  if (props.busy) return
  emit('cancel')
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 1rem;
}
.modal {
  width: min(420px, 100%);
  margin: 0;
}
</style>
