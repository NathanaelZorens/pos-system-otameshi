# Phase 5 — Discount rules

**Status:** Built (smoke-tested)  
**Product version target:** 5.0  
**Builds on:** [v4.x members](./phase-4-members.md)  
**Progress summary:** [phase-5-summary.md](./phase-5-summary.md)  
**Date:** 2026-10-08 → 2026-10-09

---

## Framing

Admin-defined **rules** that change line prices at add-item time. Loyalty stays later (v6-ish).

| Empty field | Meaning |
|-------------|---------|
| Campaign dates (`starts_on` / `ends_on`) | **Indefinite** |
| Daily window (`start_time` / `end_time`) | **All day** |
| Weekdays | **Every day** (ISO: Mon=1 … Sun=7) |

Store clock = **API server local time**.

---

## What shipped

### Schema
- `discount_rules` table (`is_featured_price` for item rules)
- `order_items`: `list_unit_price_cents`, `discount_rule_id`, `discount_label` (snapshots)

### Resolve
- Target: one **item** or one **category**
- Type: **fixed** yen or **percent**
- One discount per line (no stacking)
- Default: **largest yen savings** among matching item *and* category rules
- If any matching **item** rule has `is_featured_price`, pick among those featured rules only (cheapest featured wins)
- Applied on every create/add-item path (staff, guest, takeout, member)
- Conflict warning is **admin set-time only** (no runtime guest block)

### Admin API (admin only)
- `GET/POST /api/discount-rules`
- `PUT/DELETE /api/discount-rules/{id}`
- `POST /api/discount-rules/preview-featured` — advisory worse-deal check when enabling Featured

### UI
- Staff nav → **Discounts** (`/admin/discounts`)
- Item rules: **Prioritize as Featured price** checkbox + warning if a better deal currently matches
- Order lines show list price + discount label when discounted
- Receipt shows discount label under the line

---

## Out of v5
- Multi-category items / stacking
- Monthly recurrence
- Coupons, loyalty points
- Guest Tailwind redesign
- Future-schedule conflict scanning

---

## Smoke path
1. Admin → Discounts → create breakfast rule (e.g. coffee item, −¥50, 07:00–10:00, leave dates empty)
2. Within window: staff add that item → line shows discount; total uses charged price
3. Outside window (or toggle Active off): new lines use full price; old lines keep snapshot
4. Category rule + item rule on same product → **cheapest** wins unless the item rule is Featured
5. Featured item rule that saves less than another matching rule → admin sees warning; save still allowed; guest is charged the featured price
