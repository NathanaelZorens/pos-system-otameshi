<template>
  <div class="layout" :class="{ guest: isPublic }">
    <header v-if="user && !isPublic" class="top-nav no-print">
      <RouterLink class="brand" to="/tables">Otameshi</RouterLink>
      <nav class="nav-links" aria-label="Staff">
        <RouterLink to="/tables">Tables</RouterLink>
        <RouterLink to="/takeout-queue">Takeout</RouterLink>
        <RouterLink v-if="user.role === 'admin'" to="/admin/menu">Menu</RouterLink>
        <RouterLink v-if="user.role === 'admin'" to="/admin/reports">Reports</RouterLink>
        <RouterLink v-if="user.role === 'admin'" to="/admin/cancelled">Cancelled</RouterLink>
      </nav>
      <div class="top-nav-right">
        <span class="nav-user">{{ user.name }}</span>
        <button type="button" class="nav-logout" @click="onLogout">Logout</button>
      </div>
    </header>
    <RouterView />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from './auth'

const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()
const isPublic = computed(() => Boolean(route.meta.guest || route.meta.member))

function onLogout() {
  logout()
  router.push('/login')
}
</script>
