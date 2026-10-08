import { computed, ref } from 'vue'

const memberToken = ref(localStorage.getItem('memberToken') || '')
const member = ref(JSON.parse(localStorage.getItem('member') || 'null'))

export function useMemberAuth() {
  return {
    memberToken,
    member,
    isMemberLoggedIn: computed(() => Boolean(memberToken.value)),
    loginMember(token, nextMember) {
      memberToken.value = token
      member.value = nextMember
      localStorage.setItem('memberToken', token)
      localStorage.setItem('member', JSON.stringify(nextMember))
    },
    logoutMember() {
      memberToken.value = ''
      member.value = null
      localStorage.removeItem('memberToken')
      localStorage.removeItem('member')
    },
  }
}
