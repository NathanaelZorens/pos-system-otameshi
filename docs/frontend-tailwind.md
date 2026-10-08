# Tailwind (Option A — gradual)



Tailwind is **wired** alongside the existing `style.css` design system. Guest/member screens still use semantic classes (`.card`, `.primary`, etc.) until migrated.



## Setup



- Vite plugin: `@tailwindcss/vite`

- **No Tailwind preflight** — avoids fighting global resets; existing CSS stays authoritative for forms, nav, body.

- Cafe tokens live in `@theme` so utilities work when you use them, e.g. `bg-bg`, `text-brown`, `rounded-lg`, `font-display`.

- Shared helpers: `frontend/src/twUi.js` (`btn`, `btnPrimary`, `card`, `muted`, …).



## Staff batch (migrated)



| Screen | Route |

|--------|--------|

| Order | `/orders/:id` |

| Draft dine-in | `/tables/:tableId/order` |

| Tables | `/tables` |

| Takeout queue | `/takeout-queue` |

| New takeout draft | `/takeout-queue/new` |

| Receipt | `/receipt/:id` |

| Menu admin | `/admin/menu` |

| Reports | `/admin/reports` |

| Cancelled | `/admin/cancelled` |



Still on global CSS: `App.vue` nav, Login, guest/member pages, `CancelReasonModal`.

