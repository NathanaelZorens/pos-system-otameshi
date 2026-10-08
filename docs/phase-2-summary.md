# Phase 2 — Progress Summary

**Status:** Done (customer QR dine-in baseline)  
**Current product:** [Version 2.5](./phase-2.5-summary.md) (eat-then-pay, fulfillment, guest claim)  
**Builds on:** [Phase 1 summary](./phase-1-summary.md)

---

## Goal

Let a **guest** open a table menu via QR/link (no login), order if the table is free, pay with mock cashless, and view a receipt. If the table already has an open order → **block** the guest.

---

## What shipped

### Guest access (no JWT)
- Each dining table has a `guest_token`
- Public URL: `/t/{guest_token}`
- Staff Tables screen: **Guest link** + **QR** (external QR image for the link)

### Guest flow (Phase 2 baseline)
- Free table → draft menu (create-on-first-item)
- First Add → creates `source=guest` order
- Qty add / change / remove (same rules as staff at the time)
- Pay: **card / qr / wallet** only (mock gateway)
- Receipt at `/t/{token}/receipt/{orderId}`
- If table already has an open order → **block** new guest session

### API (public)
- `GET /api/guest/t/{token}/`
- `GET /api/guest/t/{token}/menu`
- `POST /api/guest/t/{token}/orders`
- Guest order item / checkout routes

### Staff unchanged
- Waiter/admin JWT flows from Phase 1 still work
- Tables list shows open order **source** (`staff` | `guest`)
- Reports show “Guest” when there is no waiter

**Later (v2.5):** confirm/quit/force cancel, preparing/served, guest claim, polling — see [phase-2.5-summary.md](./phase-2.5-summary.md).

---

## Locked decisions

| Topic | Choice |
|-------|--------|
| Guest auth | Table `guest_token` in URL (not JWT) |
| Open order conflict | **Block** guest (v2.5 adds claim resume — see 2.5 doc) |
| Guest payment | Cashless mock only |
| Join waiter order | Not in Phase 2 |

---

## How to try it

1. Staff login → **Tables** → **QR** or **Guest link** on a free table  
2. Open link in another tab / phone → add items → pay → receipt  
3. Staff-open a table first, then open guest link → should see **blocked**  

Full v2.5 smoke tests: [phase-2.5-summary.md](./phase-2.5-summary.md#how-to-try-v25-smoke-path).

---

## Out of Phase 2

- Guest joining an existing staff order  
- Staff re-issue claim / resume link  
- Takeout / pickup status  
- Real payments  
- Kitchen `preparing` / `served` board  
- UI redesign  

---

## Suggested next

Cancelled-ops list, staff re-issue claim, Phase 3 (takeout), or UI polish.
