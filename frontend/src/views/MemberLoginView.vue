<template>
  <div class="auth-shell">
    <div class="auth-brand">
      <p class="eyebrow">Member</p>
      <h1>{{ mode === 'register' ? 'Join' : 'Sign in' }}</h1>
      <p class="muted">Resume takeout on any phone.</p>
    </div>
    <div class="card auth-card">
      <form @submit.prevent="submit">
        <label v-if="mode === 'register'">
          Name
          <input v-model="name" required />
        </label>
        <label>
          Email
          <input v-model="email" type="email" required autocomplete="username" />
        </label>
        <label>
          Password
          <input v-model="password" type="password" required minlength="6" autocomplete="current-password" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="primary" type="submit" :disabled="loading">
          {{ loading ? '…' : mode === 'register' ? 'Create account' : 'Sign in' }}
        </button>
      </form>
      <p style="margin: 1rem 0 0; text-align: center">
        <button type="button" style="border: none; background: none; color: var(--brown); padding: 0" @click="toggleMode">
          {{ mode === 'register' ? 'Have an account? Sign in' : 'New here? Register' }}
        </button>
      </p>
      <p class="muted" style="margin: 0.75rem 0 0; text-align: center">
        <RouterLink to="/takeout">Continue as guest</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useMemberAuth } from '../memberAuth'

const router = useRouter()
const { loginMember } = useMemberAuth()

const mode = ref('login')
const name = ref('')
const email = ref('member@pos.local')
const password = ref('member123')
const error = ref('')
const loading = ref(false)

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  error.value = ''
}

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const path = mode.value === 'register' ? '/api/members/register' : '/api/members/login'
    const body =
      mode.value === 'register'
        ? { email: email.value, password: password.value, name: name.value }
        : { email: email.value, password: password.value }
    const data = await api(path, { method: 'POST', body: JSON.stringify(body) })
    loginMember(data.token, data.member)
    router.replace('/member')
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
  max-width: 320px;
}
.auth-brand h1 {
  margin: 0 0 0.4rem;
  font-size: 2.2rem;
}
.auth-brand .muted {
  margin: 0;
}
</style>
