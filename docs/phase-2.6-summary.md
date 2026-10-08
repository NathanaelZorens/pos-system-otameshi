# Version 2.6 — Progress Summary

**Product version:** 2.6  
**Status:** Done (menu categories CRUD + filter chips)  
**Date snapshot:** 2026-10-06  

**Builds on:** [Version 2.5](./phase-2.5-summary.md)  
**Why 2.6 not 3.0:** Extends existing menu organization; not a new venue/path (takeout, promos, UI redesign stay for later versions).

---

## What shipped

### Menu categories
- Table `menu_categories` (name, sort_order, soft-delete)
- Existing free-text `menu_items.category` values migrated into categories + `category_id`
- Admin **Menu** screen: add / rename / sort / remove categories
- New menu items pick a category from a dropdown
- Cannot delete a category while active items still use it

### Ordering UI
- Staff draft / open order and guest menus: **All** + category filter chips (from items that have a category)
- Uneven category sizes (1 item vs many) allowed — business content, not enforced by the app

---

## Explicitly still later (not 2.6)

- Promotions / timed discounts / tags engine  
- Guest claim re-issue  
- Takeout, KDS, real payments, Pen → Vue UI redesign  

---

## How to try

1. Restart API (migration creates/links categories)  
2. Admin → **Menu** → Categories: add/edit/remove  
3. Add a dish with a category  
4. Open a table or guest link → use category chips to filter  

---

## Suggested next

Close dine-in learning core, or see **[v3.x takeout summary](./phase-3-summary.md)** (self-order + counter done).
