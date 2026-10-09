# Version 5.x — Progress Summary

**Product version:** 5.0  
**Status:** Done (discount rules + featured price; smoke-tested)  
**Date snapshot:** 2026-10-09  
**Commit:** `v5.x update` (`9912bc3`)

**Builds on:** [Phase 4 — Members](./phase-4-members.md)  
**Detail brief:** [phase-5-discounts.md](./phase-5-discounts.md)

---

## Mental model

Admin-defined **discount rules** change line prices when an item is added. One rule per line; no stacking. Loyalty / points stay later.

| Empty schedule field | Meaning |
|----------------------|---------|
| Dates | Indefinite |
| Daily time window | All day |
| Weekdays | Every day (Mon=1 … Sun=7) |

**Resolve:** largest yen savings among matching item + category rules, unless a matching **item** rule has `is_featured_price` — then pick among featured only.

---

## What shipped

### Discount engine
- `discount_rules` (+ `is_featured_price` for item rules)
- Order-line snapshots: `list_unit_price_cents`, `discount_rule_id`, `discount_label`
- Resolve on every add-item path (staff, guest, takeout, member)
- Featured conflict check at **admin set-time only** (`POST /api/discount-rules/preview-featured`)

### Admin
- **Discounts** (`/admin/discounts`) — CRUD, schedule, Active, Featured checkbox + worse-deal warning
- Two-column layout; brown = Add, gold/yellow tint = Edit
- **Menu** — same layout; **Edit item** (name / price / category) via existing `PUT /api/menu/{id}`

### Ordering UX
- Dual-column draft / order UIs (menu + ticket)
- `DiscountBadge` on lines; list price + charged price when discounted
- Receipt shows discount label under the line

---

## Explicitly still later (not 5.0)

- Multi-category items / stacking  
- Monthly recurrence / future-schedule conflict scan  
- Coupons, **loyalty points** (v6-ish)  
- Guest Tailwind redesign  

---

## How to try

1. Restart API (migration / ensure `discount_rules`)  
2. Admin → **Discounts** → e.g. coffee −¥50, 07:00–10:00  
3. Within window: add item → badge + charged price  
4. Outside window / Active off → new lines full price; old lines keep snapshot  
5. Featured item rule cheaper than category deal → warning; guest still pays featured price  

Demo logins: [how-to-run-fe-be.md](./how-to-run-fe-be.md).

---

## One-line summary

**Timed item/category discounts with cheapest-wins (and optional Featured override); loyalty still waits.**

---

## Suggested next

1. **Loyalty / points** (member table QR + takeout hooks)  
2. Guest UI redesign (Pen → Vue / Tailwind)  
3. Coupons or multi-category stacking if needed before loyalty  
