/** Shared Tailwind class strings for staff UI (cafe tokens from style.css @theme). */

export const btn =
  'cursor-pointer rounded-md border border-line bg-surface-raised px-4 py-[0.6rem] font-semibold text-ink focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold-deep active:enabled:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-55'

export const btnPrimary = `${btn} border-brown bg-brown text-surface`
export const btnGold = `${btn} border-gold bg-gold text-ink`
export const btnDanger = `${btn} border-danger bg-danger text-white`
export const btnSoft = `${btn} border-transparent bg-gold-soft text-gold-deep`
export const btnPill = `${btn} rounded-full px-3.5 py-1.5 text-sm font-semibold text-ink-soft`
export const btnPillActive = `${btnPill} border-brown bg-brown text-surface`

export const card = 'mb-4 rounded-xl border border-line bg-surface-raised px-5 py-[1.15rem]'
export const muted = 'text-sm text-ink-muted'
export const row = 'flex flex-wrap items-center gap-2'
export const rowEnd = 'flex flex-wrap items-end gap-2'
