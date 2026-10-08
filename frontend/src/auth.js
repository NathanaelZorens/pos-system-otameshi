import { computed, ref } from 'vue'

const token = ref(localStorage.getItem('token') || '')
const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))

export function useAuth() {
  return {
    token,
    user,
    isLoggedIn: computed(() => Boolean(token.value)),
    login(nextToken, nextUser) {
      token.value = nextToken
      user.value = nextUser
      localStorage.setItem('token', nextToken)
      localStorage.setItem('user', JSON.stringify(nextUser))
    },
    logout() {
      token.value = ''
      user.value = null
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    },
  }
}
