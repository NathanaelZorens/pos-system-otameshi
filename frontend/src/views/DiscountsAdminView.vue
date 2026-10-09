<template>
  <div>
    <div class="mb-5">
      <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Discounts</h1>
      <p :class="muted">
        Rules for item or category. Empty time = all day; empty weekdays = every day; empty dates = indefinite.
      </p>
    </div>

    <div class="grid gap-4 lg:grid-cols-[minmax(300px,0.42fr)_minmax(0,1fr)] lg:items-start">
      <div :class="[editingId ? cardFormEdit : cardFormAdd, 'mb-0 lg:sticky lg:top-4']">
        <h2>{{ editingId ? 'Edit rule' : 'Add rule' }}</h2>
        <form class="grid gap-3" @submit.prevent="save">
          <label>
            Name
            <input v-model="form.name" placeholder="Breakfast coffee −¥50" required />
          </label>
          <div :class="row">
            <label class="min-w-40 flex-1">
              Target type
              <select v-model="form.target_type">
                <option value="item">Item</option>
                <option value="category">Category</option>
              </select>
            </label>
            <label class="min-w-48 flex-1">
              Target
              <select v-model="form.target_id" required>
                <option value="" disabled>Select…</option>
                <option v-for="t in targetOptions" :key="t.id" :value="t.id">{{ t.name }}</option>
              </select>
            </label>
          </div>
          <div :class="row">
            <label class="min-w-40 flex-1">
              Type
              <select v-model="form.discount_type">
                <option value="fixed">Fixed yen (−¥)</option>
                <option value="percent">Percent (%)</option>
              </select>
            </label>
            <label class="min-w-32 flex-1">
              Amount
              <input v-model.number="form.amount" type="number" min="1" :max="form.discount_type === 'percent' ? 100 : undefined" required />
            </label>
          </div>
          <div :class="row">
            <label class="min-w-28 flex-1">
              Start time
              <input v-model="form.start_time" type="time" />
            </label>
            <label class="min-w-28 flex-1">
              End time
              <input v-model="form.end_time" type="time" />
            </label>
          </div>
          <fieldset class="m-0 border-0 p-0">
            <legend class="mb-1 text-sm font-medium text-ink-soft">Weekdays (empty = every day)</legend>
            <div class="flex flex-wrap gap-2">
              <label v-for="d in weekdayOptions" :key="d.value" class="flex items-center gap-1.5 text-sm text-ink">
                <input v-model="form.weekdaySet" type="checkbox" :value="d.value" class="!mt-0 !w-auto" />
                {{ d.label }}
              </label>
            </div>
          </fieldset>
          <div :class="row">
            <label class="min-w-36 flex-1">
              Starts on
              <input v-model="form.starts_on" type="date" />
            </label>
            <label class="min-w-36 flex-1">
              Ends on
              <input v-model="form.ends_on" type="date" />
            </label>
          </div>
          <label class="flex items-center gap-2 text-sm text-ink">
            <input v-model="form.is_active" type="checkbox" class="!mt-0 !w-auto" />
            Active
          </label>
          <label
            v-if="form.target_type === 'item'"
            class="flex items-start gap-2 text-sm text-ink"
          >
            <input v-model="form.is_featured_price" type="checkbox" class="!mt-0.5 !w-auto" />
            <span>
              Prioritize as Featured price
              <span :class="muted" class="mt-0.5 block">
                When on, this item rule wins over cheaper matching rules for this product.
              </span>
            </span>
          </label>
          <p
            v-if="featuredWarning"
            class="m-0 rounded-md border border-[#e8c9a0] bg-[#fff8ef] px-3 py-2 text-sm text-[#7a4a12]"
          >
            {{ featuredWarning }}
          </p>
          <div :class="row">
            <button :class="btnPrimary" type="submit" :disabled="busy">
              {{ editingId ? 'Save' : 'Add rule' }}
            </button>
            <button v-if="editingId" type="button" :class="btn" @click="resetForm">Cancel edit</button>
          </div>
        </form>
        <p v-if="error" class="mt-2 text-danger">{{ error }}</p>
      </div>

      <div :class="[card, 'mb-0']">
        <h2>Rules</h2>
        <p v-if="!rules.length" :class="muted">No discount rules yet.</p>
        <div class="grid gap-2">
          <div
            v-for="r in rules"
            :key="r.id"
            class="flex flex-wrap items-start justify-between gap-3 border-b border-line py-3.5 last:border-b-0"
          >
            <div class="min-w-0 flex-1">
              <strong>{{ r.name }}</strong>
              <span
                class="ml-2 inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold"
                :class="r.is_active ? 'bg-[#e8f0e2] text-success' : 'bg-danger-soft text-danger'"
              >
                {{ r.is_active ? 'active' : 'off' }}
              </span>
              <span
                v-if="r.is_featured_price"
                class="ml-2 inline-flex items-center rounded-full bg-[#efe8df] px-2 py-0.5 text-xs font-semibold text-ink-soft"
              >
                featured
              </span>
              <div :class="muted">
                {{ r.target_type }} · {{ r.target_name || r.target_id }}
                ·
                {{ r.discount_type === 'fixed' ? `−¥${r.amount}` : `−${r.amount}%` }}
              </div>
              <div :class="muted">{{ describeSchedule(r) }}</div>
            </div>
            <div :class="row">
              <button type="button" :class="btn" @click="startEdit(r)">Edit</button>
              <button type="button" :class="btnDanger" @click="remove(r)">Remove</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'
import { btn, btnDanger, btnPrimary, card, cardFormAdd, cardFormEdit, muted, row } from '../twUi'

const weekdayOptions = [
  { value: '1', label: 'Mon' },
  { value: '2', label: 'Tue' },
  { value: '3', label: 'Wed' },
  { value: '4', label: 'Thu' },
  { value: '5', label: 'Fri' },
  { value: '6', label: 'Sat' },
  { value: '7', label: 'Sun' },
]

const rules = ref([])
const menu = ref([])
const categories = ref([])
const error = ref('')
const busy = ref(false)
const editingId = ref('')
const featuredWarning = ref('')
let previewTimer = null

const form = reactive(emptyForm())

const targetOptions = computed(() => {
  if (form.target_type === 'category') {
    return categories.value.map((c) => ({ id: c.id, name: c.name }))
  }
  return menu.value.map((m) => ({ id: m.id, name: m.name }))
})

watch(
  () => form.target_type,
  () => {
    if (!editingId.value) form.target_id = ''
    if (form.target_type !== 'item') {
      form.is_featured_price = false
      featuredWarning.value = ''
    }
  },
)

watch(
  () => [
    form.is_featured_price,
    form.target_type,
    form.target_id,
    form.discount_type,
    form.amount,
    editingId.value,
  ],
  () => {
    scheduleFeaturedPreview()
  },
)

onMounted(load)

function emptyForm() {
  return {
    name: '',
    target_type: 'item',
    target_id: '',
    discount_type: 'fixed',
    amount: 50,
    start_time: '',
    end_time: '',
    weekdaySet: [],
    starts_on: '',
    ends_on: '',
    is_active: true,
    is_featured_price: false,
  }
}

function resetForm() {
  editingId.value = ''
  featuredWarning.value = ''
  Object.assign(form, emptyForm())
}

function scheduleFeaturedPreview() {
  if (previewTimer) clearTimeout(previewTimer)
  previewTimer = setTimeout(runFeaturedPreview, 280)
}

async function runFeaturedPreview() {
  featuredWarning.value = ''
  if (!form.is_featured_price || form.target_type !== 'item' || !form.target_id) return
  if (!form.amount || form.amount <= 0) return
  try {
    const result = await api('/api/discount-rules/preview-featured', {
      method: 'POST',
      body: JSON.stringify({
        exclude_rule_id: editingId.value || '',
        target_type: form.target_type,
        target_id: form.target_id,
        discount_type: form.discount_type,
        amount: form.amount,
      }),
    })
    if (result?.worse_deal && result.message) {
      featuredWarning.value = result.message
    }
  } catch {
    // Preview is advisory; ignore network blips while typing.
  }
}

async function load() {
  error.value = ''
  try {
    ;[rules.value, menu.value, categories.value] = await Promise.all([
      api('/api/discount-rules'),
      api('/api/menu?all=1'),
      api('/api/categories'),
    ])
    rules.value = rules.value || []
    menu.value = menu.value || []
    categories.value = categories.value || []
  } catch (e) {
    error.value = e.message
  }
}

function startEdit(r) {
  editingId.value = r.id
  form.name = r.name
  form.target_type = r.target_type
  form.target_id = r.target_id
  form.discount_type = r.discount_type
  form.amount = r.amount
  form.start_time = r.start_time || ''
  form.end_time = r.end_time || ''
  form.weekdaySet = r.weekdays ? r.weekdays.split(',').map((s) => s.trim()).filter(Boolean) : []
  form.starts_on = r.starts_on || ''
  form.ends_on = r.ends_on || ''
  form.is_active = r.is_active
  form.is_featured_price = !!r.is_featured_price && r.target_type === 'item'
  scheduleFeaturedPreview()
}

function payload() {
  const weekdays = form.weekdaySet.length ? [...form.weekdaySet].sort().join(',') : null
  return {
    name: form.name.trim(),
    target_type: form.target_type,
    target_id: form.target_id,
    discount_type: form.discount_type,
    amount: form.amount,
    start_time: form.start_time || null,
    end_time: form.end_time || null,
    weekdays,
    starts_on: form.starts_on || null,
    ends_on: form.ends_on || null,
    is_active: form.is_active,
    is_featured_price: form.target_type === 'item' ? !!form.is_featured_price : false,
  }
}

async function save() {
  error.value = ''
  busy.value = true
  try {
    const body = JSON.stringify(payload())
    if (editingId.value) {
      await api(`/api/discount-rules/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/discount-rules', { method: 'POST', body })
    }
    resetForm()
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function remove(r) {
  if (!confirm(`Remove rule “${r.name}”?`)) return
  error.value = ''
  try {
    await api(`/api/discount-rules/${r.id}`, { method: 'DELETE' })
    if (editingId.value === r.id) resetForm()
    await load()
  } catch (e) {
    error.value = e.message
  }
}

function describeSchedule(r) {
  const bits = []
  if (r.start_time || r.end_time) {
    bits.push(`${r.start_time || '00:00'}–${r.end_time || '23:59'}`)
  } else {
    bits.push('all day')
  }
  if (r.weekdays) {
    const map = Object.fromEntries(weekdayOptions.map((d) => [d.value, d.label]))
    bits.push(
      r.weekdays
        .split(',')
        .map((w) => map[w.trim()] || w)
        .join(', '),
    )
  } else {
    bits.push('every day')
  }
  if (r.starts_on || r.ends_on) {
    bits.push(`${r.starts_on || '…'} → ${r.ends_on || '…'}`)
  } else {
    bits.push('indefinite')
  }
  return bits.join(' · ')
}
</script>
