# Order flow decisions (post Phase 2)

Living product decisions for dine-in ordering, confirm, cancel, and fulfillment.  
Related: [phase-1-summary.md](./phase-1-summary.md) · [phase-2-summary.md](./phase-2-summary.md) · [phase-2.5-summary.md](./phase-2.5-summary.md) · [phase-2.6-summary.md](./phase-2.6-summary.md) (categories)

**Last updated:** 2026-10-05

---

## Payment model

- **Eat then pay** (not pay-first).
- **Pay** settles the whole open check at the end.
- **Pay blocked** while any line is still `pending` (must Confirm first).
- Guest pay methods: cashless mock only (`card` / `qr` / `wallet`).
- Staff pay: cash + cashless mock.

---

## Two layers (do not mix)

| Layer | Values | Meaning |
|-------|--------|---------|
| **Line (dish)** | `pending` → `confirmed` | Is this item official? |
| **Order (visit)** | `open` → `preparing` → `served` → `paid` → **`cleared`** / `cancelled` | Fulfillment + seats |

- **Pay** may happen before or after Mark served (bill when they ask).
- **Mark served** is tracked via `served_at` (independent of pay). Button stays on **paid** until served, then hides (no undo).
- **Clear table** (staff): only when `paid` — seats are physically free. Until then the table stays occupied.

- **Confirm** locks lines; it is not the same as preparing/served.
- Mid-meal: can **add more** → new `pending` lines → **Confirm** again (batch confirm).

---

## Line rules (built)

- New items start **`pending`** (editable / removable).
- **Confirm** (guest + waiter): all current pending → **`confirmed`**.
- Confirmed lines: **no** qty change / remove.
- Adding the same dish again after confirm creates a **new pending** line (does not bump confirmed qty).
- Whole-check abandon: see Quit / Force cancel below.

---

## Quit vs Force cancel (built)

| Action | Who | When | Meaning |
|--------|-----|------|---------|
| **Quit** | Guest + waiter | Only if **no confirmed lines** yet | Customer doesn’t intend to buy; discard early |
| **Force cancel** | **Waiter/admin only** (not guest) | Even if preparing/served; while not paid | Rare escape hatch; strong confirm dialog |
| Guest mid-confirm dish change | — | **Not allowed** | Later: staff **void/replace** line if needed |

Rename UI “Cancel order” → **Quit** where it means early abandon.  
Staff override → **Force cancel**.  
Both ask for a **free-text reason** (required). Stored as `cancel_reason`.  
Removing the last item still cancels the check (**no** reason prompt; reason may be blank).  
Admin list: `/admin/cancelled`.

---

## Fulfillment (built)

- **Order-level** preparing/served (not per-item for now).
- **`preparing`:** set **automatically on first Confirm** (kitchen has real work). Further confirms stay in preparing (or return to preparing if needed later).
- **`served`:** **waiter** taps when food is brought out.
- **No chef view** for now.

### Cancel rules once fulfillment exists

- **Quit** — only before confirmed (and thus before preparing).
- **Force cancel** — staff only; allowed in rare cases after preparing/served.
- Normal guest cancel after confirm — **no**.

---

## Guest QR

- Access via permanent `guest_token` URL (not JWT). QR stays valid for the table.
- On guest create, server mints a **`guest_claim`** (opaque secret). Client stores it in **`localStorage`** and sends `X-Guest-Claim` on guest order APIs.
- Matching claim → **resume** the open order (refresh / reopen tab).
- Open order + no/wrong claim → **blocked** (“ask staff”). Staff POS completes the check.
- Claim cleared on pay / quit / cancel. Paid receipt readable without claim.
- Lost claim (incognito closed, cleared storage, other phone) → staff finishes the order.

---

## Explicitly deferred

- Item-level preparing/served  
- Chef / KDS view  
- Staff void-single-line with reason (workaround for wrong dish)  
- Manager-only force cancel after served  
- Real payments / refunds  

---

## What to build next (suggested order)

1. **Quit rename + rules** — **done**
2. **Force cancel (staff)** — **done**
3. **Fulfillment statuses** — **done** (`preparing` on Confirm; waiter Mark served; tables badges)
4. **Wire cancel policy to fulfillment** — **done** (Quit still gated by confirmed lines; Force cancel works on open/preparing/served)
5. **Cancelled / force-cancel ops list** — **done** (`/admin/cancelled`; free-text `cancel_reason` on Quit / Force cancel)  
6. Optional later: staff re-issue guest claim, void line, join open order, takeout, UI redesign.  
7. **Menu categories CRUD** — **done** (v2.6)  
8. Promotions / timed discounts — **done** (v5 — [summary](./phase-5-summary.md) · [brief](./phase-5-discounts.md))
