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
      <div :class="card">
        <h2>Add item</h2>
        <form :class="row" @submit.prevent="createItem">
          <input v-model="form.name" placeholder="Name" required />
          <input v-model.number="form.price_cents" type="number" min="0" placeholder="Price (yen)" required />
          <select v-model="form.category_id">
            <option value="">No category</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
          <button :class="btnPrimary" type="submit">Add</button>
        </form>
        <p v-if="error" class="text-danger">{{ error }}</p>
      </div>

      <div :class="card">
        <h2>Items</h2>
        <div class="grid gap-2">
          <div
            v-for="m in menu"
            :key="m.id"
            class="flex items-center justify-between gap-3 border-b border-line py-3.5"
          >
            <div>
              <strong>{{ m.name }}</strong>
              <div :class="muted">{{ formatMoney(m.price_cents) }} · {{ m.category || '—' }}</div>
            </div>
            <div :class="row">
              <button type="button" :class="m.is_sold_out ? btnSoft : btn" @click="toggleSoldOut(m)">
                {{ m.is_sold_out ? 'Mark available' : 'Sold out' }}
              </button>
              <button type="button" :class="btnDanger" @click="removeItem(m.id)">Remove</button>
            </div>
          </div>
          <p v-if="!menu.length" :class="muted">No items yet.</p>
        </div>
      </div>
    </template>

    <template v-else>
      <div :class="card">
        <h2>Categories</h2>
        <form :class="row" @submit.prevent="createCategory">
          <input v-model="catForm.name" placeholder="Category name" required />
          <button :class="btnPrimary" type="submit">Add category</button>
        </form>
        <p v-if="catError" class="text-danger">{{ catError }}</p>
        <div class="mt-3 grid gap-2">
          <div
            v-for="c in categories"
            :key="c.id"
            class="flex items-center justify-between gap-3 border-b border-line py-3.5"
          >
            <div v-if="editingCatId !== c.id">
              <strong>{{ c.name }}</strong>
              <div :class="muted">sort {{ c.sort_order }}</div>
            </div>
            <form v-else :class="row" @submit.prevent="saveCategory(c)">
              <input v-model="editCatName" required />
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
    </template>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api, formatMoney } from '../api'
import { btn, btnDanger, btnPill, btnPillActive, btnPrimary, btnSoft, card, muted, row } from '../twUi'

const tab = ref('items')
const menu = ref([])
const categories = ref([])
const error = ref('')
const catError = ref('')
const form = reactive({ name: '', price_cents: 0, category_id: '' })
const catForm = reactive({ name: '' })
const editingCatId = ref('')
const editCatName = ref('')
const editCatSort = ref(0)

onMounted(load)

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

async function createItem() {
  error.value = ''
  try {
    await api('/api/menu', {
      method: 'POST',
      body: JSON.stringify({
        name: form.name,
        price_cents: form.price_cents,
        category_id: form.category_id || null,
      }),
    })
    form.name = ''
    form.price_cents = 0
    form.category_id = ''
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

async function removeItem(id) {
  if (!confirm('Remove this menu item?')) return
  await api(`/api/menu/${id}`, { method: 'DELETE' })
  await load()
}
</script>
