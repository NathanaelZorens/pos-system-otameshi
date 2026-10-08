# Version 3.x — Progress Summary

**Product versions:** 3.0 → 3.1 (+ small polish)  
**Status:** Done for takeout core (self-order + counter)  
**Date snapshot:** 2026-10-06  

**Builds on:** [Version 2.6](./phase-2.6-summary.md) (dine-in + categories)  
**Detail briefs:** [phase-3-takeout.md](./phase-3-takeout.md) · [phase-3.1-counter-takeout.md](./phase-3.1-counter-takeout.md)

---

## Mental model

| Lane | Entry | Pay | After pay |
|------|--------|-----|-----------|
| **Dine-in** | Tables / table QR | Eat-then-pay | Table receipt |
| **Self-order takeout** | Public `/takeout` | Cashless first | Phone status + pickup code |
| **Counter takeout** | Staff **New takeout** | Any method first | Same takeout queue |

Same kitchen queue for both takeout doors: `open` → pay → `preparing` → `ready` → `completed`. Not mixed into open tables.

---

## What shipped

### 3.0 — Self-order takeout
- Schema: `source=takeout`, nullable `dining_table_id`, `pickup_code`, statuses `ready` / `completed`
- Guest APIs under `/api/guest/takeout/...` (claim, confirm, cashless checkout → preparing)
- Guest UI: `/takeout` (cart → confirm → pay → status poll)
- Staff queue: `/takeout-queue` (ready / picked up / force cancel)
- Confirm does **not** start kitchen for takeout (pay does)

### 3.1 — Counter-order takeout
- `POST /api/takeout/orders` (waiter/admin); sets `waiter_id`, no guest claim
- Staff checkout via `/api/orders/{id}/checkout` → `preparing` (cash/card/QR/wallet)
- UI: queue → **New takeout** → `/takeout-queue/new` → order screen → back to queue

### 3.x polish
- Queue **Find pickup** (code or name filter)
- Staff receipt: dine-in shows **table**; takeout shows **pickup code**
- Order UIs: **Pay hidden** until confirm and no pending lines

---

## Explicitly still later

- Pen → Vue UI redesign (optional Tailwind with that pass)  
- Logged-in guest / order-from-home resume  
- Receipt auto-print / deeper receipt ops  
- Guest cash / pay-at-pickup  
- Promotions, delivery, real payment gateway  

---

## How to try

1. Restart API if needed; FE `npm run dev`  
2. **Self-order:** http://localhost:5173/takeout → add → confirm → pay → watch status  
3. **Staff:** login → **Takeout** → queue search / mark ready / picked up  
4. **Counter:** **New takeout** → confirm → pay (e.g. cash) → appears on queue  
5. **Reports** → open a takeout receipt → big pickup code  

Demo logins: see [how-to-run-fe-be.md](./how-to-run-fe-be.md).

---

## Suggested next

Pause on takeout core, or start one new door:

1. **[Phase 4 — Members](./phase-4-members.md)** — built (auth + takeout resume)  
2. **UI redesign** (Pen + optional Tailwind)  
3. **Loyalty** (points on QR when member logged in)  
4. **Receipt ops** (print after counter pay, etc.)
