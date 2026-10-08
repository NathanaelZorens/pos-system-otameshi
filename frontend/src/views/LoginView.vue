<template>
  <div class="auth-shell">
    <div class="auth-brand">
      <p class="eyebrow">Otameshi</p>
      <h1>Staff sign in</h1>
      <p class="muted">Waiter and admin access for tables, takeout, and reports.</p>
    </div>
    <div class="card auth-card">
      <form @submit.prevent="submit">
        <label>
          Email
          <input v-model="email" type="email" required autocomplete="username" />
        </label>
        <label>
          Password
          <input v-model="password" type="password" required autocomplete="current-password" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="primary" type="submit" :disabled="loading">
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
      <p class="muted" style="margin: 1rem 0 0; text-align: center">
        Demo: waiter@pos.local / waiter123
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useAuth } from '../auth'

const router = useRouter()
const { login } = useAuth()
const email = ref('waiter@pos.local')
const password = ref('waiter123')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const data = await api('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email: email.value, password: password.value }),
    })
    login(data.token, data.user)
    router.push('/tables')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-brand {
  text-align: center;
  max-width: 360px;
}

.auth-brand h1 {
  margin: 0 0 0.5rem;
  font-size: 2.4rem;
}

.auth-brand .muted {
  margin: 0;
}
</style>
