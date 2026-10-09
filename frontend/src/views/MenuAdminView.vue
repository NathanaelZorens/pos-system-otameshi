<template>
  <div>
    <div class="mb-5">
      <h1 class="m-0 mb-1.5 font-display text-[2rem] font-semibold text-ink">Menu</h1>
      <p :class="muted">Categories, prices, sold-out toggles.</p>
    </div>

    <div class="mb-3 flex flex-wrap gap-1.5" role="tablist" aria-label="Menu admin">
      <button
        type="button"
        role="tab"
        :class="tab === 'items' ? btnPillActive : btnPill"
        :aria-selected="tab === 'items'"
        @click="tab = 'items'"
      >
        Items
      </button>
      <button
        type="button"
        role="tab"
        :class="tab === 'categories' ? btnPillActive : btnPill"
        :aria-selected="tab === 'categories'"
        @click="tab = 'categories'"
      >
        Categories
      </button>
    </div>

    <template v-if="tab === 'items'">
      <div class="grid gap-4 lg:grid-cols-[minmax(300px,0.42fr)_minmax(0,1fr)] lg:items-start">
        <div :class="[editingItemId ? cardFormEdit : cardFormAdd, 'mb-0 lg:sticky lg:top-4']">
          <h2>{{ editingItemId ? 'Edit item' : 'Add item' }}</h2>
          <form class="grid gap-3" @submit.prevent="saveItem">
            <label>
              Name
              <input v-model="form.name" placeholder="Name" required />
            </label>
            <label>
              Price (yen)
              <input v-model.number="form.price_cents" type="number" min="0" placeholder="Price (yen)" required />
            </label>
            <label>
              Category
              <select v-model="form.category_id">
                <option value="">No category</option>
                <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
            </label>
            <div :class="row">
              <button :class="btnPrimary" type="submit">{{ editingItemId ? 'Save' : 'Add' }}</button>
              <button v-if="editingItemId" type="button" :class="btn" @click="resetItemForm">Cancel edit</button>
            </div>
          </form>
          <p v-if="error" class="mt-2 text-danger">{{ error }}</p>
        </div>

        <div :class="[card, 'mb-0']">
          <h2>Items</h2>
          <div class="grid gap-2">
            <div
              v-for="m in menu"
              :key="m.id"
              class="flex flex-wrap items-center justify-between gap-3 border-b border-line py-3.5 last:border-b-0"
            >
              <div class="min-w-0 flex-1">
                <strong>{{ m.name }}</strong>
                <div :class="muted">{{ formatMoney(m.price_cents) }} · {{ m.category || '—' }}</div>
              </div>
              <div :class="row">
                <button type="button" :class="btn" @click="startEditItem(m)">Edit</button>
                <button type="button" :class="m.is_sold_out ? btnSoft : btn" @click="toggleSoldOut(m)">
                  {{ m.is_sold_out ? 'Mark available' : 'Sold out' }}
                </button>
                <button type="button" :class="btnDanger" @click="removeItem(m)">Remove</button>
              </div>
            </div>
            <p v-if="!menu.length" :class="muted">No items yet.</p>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="grid gap-4 lg:grid-cols-[minmax(300px,0.42fr)_minmax(0,1fr)] lg:items-start">
        <div :class="[card, 'mb-0 lg:sticky lg:top-4']">
          <h2>Add category</h2>
          <form class="grid gap-3" @submit.prevent="createCategory">
            <label>
              Name
              <input v-model="catForm.name" placeholder="Category name" required />
            </label>
            <div :class="row">
              <button :class="btnPrimary" type="submit">Add category</button>
            </div>
          </form>
          <p v-if="catError" class="mt-2 text-danger">{{ catError }}</p>
        </div>

        <div :class="[card, 'mb-0']">
          <h2>Categories</h2>
          <div class="grid gap-2">
            <div
              v-for="c in categories"
              :key="c.id"
              class="flex flex-wrap items-center justify-between gap-3 border-b border-line py-3.5 last:border-b-0"
            >
              <div v-if="editingCatId !== c.id" class="min-w-0 flex-1">
                <strong>{{ c.name }}</strong>
                <div :class="muted">sort {{ c.sort_order }}</div>
              </div>
              <form v-else class="flex min-w-0 flex-1 flex-wrap items-center gap-2" @submit.prevent="saveCategory(c)">
                <input v-model="editCatName" required class="min-w-32 flex-1" />
                <input v-model.number="editCatSort" type="number" class="!w-20" title="Sort order" />
                <button :class="btnPrimary" type="submit">Save</button>
                <button type="button" :class="btn" @click="cancelEditCat">Cancel</button>
              </form>
              <div :class="row" v-if="editingCatId !== c.id">
                <button type="button" :class="btn" @click="startEditCat(c)">Edit</button>
                <button :class="btnDanger" type="button" @click="removeCategory(c)">Remove</button>
              </div>
            </div>
            <p v-if="!categories.length" :class="muted">No categories yet.</p>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api, formatMoney } from '../api'
import { btn, btnDanger, btnPill, btnPillActive, btnPrimary, btnSoft, card, cardFormAdd, cardFormEdit, muted, row } from '../twUi'

const tab = ref('items')
const menu = ref([])
const categories = ref([])
const error = ref('')
const catError = ref('')
const editingItemId = ref('')
const form = reactive(emptyItemForm())
const catForm = reactive({ name: '' })
const editingCatId = ref('')
const editCatName = ref('')
const editCatSort = ref(0)

onMounted(load)

function emptyItemForm() {
  return { name: '', price_cents: 0, category_id: '' }
}

function resetItemForm() {
  editingItemId.value = ''
  Object.assign(form, emptyItemForm())
  error.value = ''
}

function startEditItem(m) {
  editingItemId.value = m.id
  form.name = m.name
  form.price_cents = m.price_cents
  form.category_id = m.category_id || ''
  error.value = ''
}

async function load() {
  ;[menu.value, categories.value] = await Promise.all([api('/api/menu?all=1'), api('/api/categories')])
  menu.value = menu.value || []
  categories.value = categories.value || []
}

async function createCategory() {
  catError.value = ''
  try {
    await api('/api/categories', {
      method: 'POST',
      body: JSON.stringify({ name: catForm.name.trim() }),
    })
    catForm.name = ''
    await load()
  } catch (e) {
    catError.value = e.message
  }
}

function startEditCat(c) {
  editingCatId.value = c.id
  editCatName.value = c.name
  editCatSort.value = c.sort_order
}

function cancelEditCat() {
  editingCatId.value = ''
}

async function saveCategory(c) {
  catError.value = ''
  try {
    await api(`/api/categories/${c.id}`, {
      method: 'PUT',
      body: JSON.stringify({
        name: editCatName.value.trim(),
        sort_order: editCatSort.value,
      }),
    })
    editingCatId.value = ''
    await load()
  } catch (e) {
    catError.value = e.message
  }
}

async function removeCategory(c) {
  if (!confirm(`Remove category “${c.name}”? Items must be unassigned or moved first.`)) return
  catError.value = ''
  try {
    await api(`/api/categories/${c.id}`, { method: 'DELETE' })
    await load()
  } catch (e) {
    catError.value = e.message
  }
}

async function saveItem() {
  error.value = ''
  const body = JSON.stringify({
    name: form.name.trim(),
    price_cents: form.price_cents,
    category_id: form.category_id || null,
  })
  try {
    if (editingItemId.value) {
      await api(`/api/menu/${editingItemId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/menu', { method: 'POST', body })
    }
    resetItemForm()
    await load()
  } catch (e) {
    error.value = e.message
  }
}

async function toggleSoldOut(m) {
  await api(`/api/menu/${m.id}/sold-out`, {
    method: 'PATCH',
    body: JSON.stringify({ sold_out: !m.is_sold_out }),
  })
  await load()
}

async function removeItem(m) {
  if (!confirm(`Remove “${m.name}”?`)) return
  error.value = ''
  try {
    await api(`/api/menu/${m.id}`, { method: 'DELETE' })
    if (editingItemId.value === m.id) resetItemForm()
    await load()
  } catch (e) {
    error.value = e.message
  }
}
</script>
