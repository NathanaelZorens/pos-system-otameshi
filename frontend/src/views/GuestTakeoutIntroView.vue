<template>
  <div class="auth-shell">
    <div style="width: 100%">
      <p class="eyebrow">Takeout</p>
      <h1 style="margin: 0 0 0.75rem; font-size: 2.1rem">Order & pick up</h1>
      <p class="muted" style="margin: 0 0 1.5rem; line-height: 1.45">
        Pay first on your phone. We’ll prep your order — show the pickup code at the counter.
      </p>
      <div class="row" style="flex-direction: column; align-items: stretch">
        <RouterLink to="/takeout/order">
          <button type="button" class="primary" style="width: 100%; height: 48px">
            Continue as guest
          </button>
        </RouterLink>
        <RouterLink to="/member/login">
          <button type="button" class="soft" style="width: 100%; height: 48px">
            I have a member login
          </button>
        </RouterLink>
      </div>
    </div>
  </div>
</template>
