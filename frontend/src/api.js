const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

function authHeaders() {
  const token = localStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export async function api(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...(options.headers || {}),
    },
  })
  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`)
  }
  return data
}

export function guestClaimStorageKey(guestToken) {
  return `guestClaim:${guestToken}`
}

export function getGuestClaim(guestToken) {
  return localStorage.getItem(guestClaimStorageKey(guestToken)) || ''
}

export function setGuestClaim(guestToken, claim) {
  if (claim) localStorage.setItem(guestClaimStorageKey(guestToken), claim)
  else localStorage.removeItem(guestClaimStorageKey(guestToken))
}

export function clearGuestClaim(guestToken) {
  localStorage.removeItem(guestClaimStorageKey(guestToken))
}

/** Guest QR APIs — table guest_token in path; claim in X-Guest-Claim when present. */
export async function guestApi(guestToken, path, options = {}) {
  const claim = getGuestClaim(guestToken)
  const res = await fetch(`${API_BASE}/api/guest/t/${guestToken}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(claim ? { 'X-Guest-Claim': claim } : {}),
      ...(options.headers || {}),
    },
  })
  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`)
  }
  return data
}

const TAKEOUT_ORDER_KEY = 'takeoutOrderId'

export function takeoutClaimKey(orderId) {
  return `takeoutClaim:${orderId}`
}

export function getTakeoutOrderId() {
  return localStorage.getItem(TAKEOUT_ORDER_KEY) || ''
}

export function setTakeoutSession(orderId, claim) {
  if (orderId) localStorage.setItem(TAKEOUT_ORDER_KEY, orderId)
  if (orderId && claim) localStorage.setItem(takeoutClaimKey(orderId), claim)
}

export function clearTakeoutSession() {
  const id = getTakeoutOrderId()
  if (id) localStorage.removeItem(takeoutClaimKey(id))
  localStorage.removeItem(TAKEOUT_ORDER_KEY)
}

/** Member APIs — Bearer memberToken (separate from staff token). */
export async function memberApi(path, options = {}) {
  const token = localStorage.getItem('memberToken') || ''
  const res = await fetch(`${API_BASE}/api/member${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options.headers || {}),
    },
  })
  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(data.error || `Request failed (${res.status})`)
    err.data = data
    err.status = res.status
    throw err
  }
  return data
}

/** Guest takeout APIs — claim in X-Guest-Claim when a session exists. */
export async function takeoutApi(path, options = {}) {
  const orderId = getTakeoutOrderId()
  const claim = orderId ? localStorage.getItem(takeoutClaimKey(orderId)) || '' : ''
  const res = await fetch(`${API_BASE}/api/guest/takeout${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(claim ? { 'X-Guest-Claim': claim } : {}),
      ...(options.headers || {}),
    },
  })
  if (res.status === 204) return null
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`)
  }
  return data
}

// formatMoney treats stored integers as whole yen (JPY has no minor units).
export function formatMoney(yen) {
  return new Intl.NumberFormat('ja-JP', {
    style: 'currency',
    currency: 'JPY',
  }).format(yen)
}
