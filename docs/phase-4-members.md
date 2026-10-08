# Phase 4 — Members

**Status:** Built  
**Product version target:** 4.0  
**Builds on:** [v3.x takeout](./phase-3-summary.md)  
**Date:** 2026-10-07

---

## Framing

**Member** = customer with an account. Not staff. Not anonymous **guest**.

Same kitchen / pay rules as today — **identity + resume**, not a new order type.

| Context | Who | Flow |
|---------|-----|------|
| Member home / app | Logged-in member | **Takeout only** (pay-first, cashless like self-order) |
| Table QR, anonymous | Guest | Dine-in as today |
| Table QR, logged-in member | Still dine-in | Same table flow; **loyalty hooks later** (points/discounts) |

UI contrast: “Member login” vs “Continue as guest.”

---

## What shipped

### Schema
- `members` table (email, password_hash, name)
- `orders.member_id` (nullable FK)

### Auth
- `POST /api/members/register`, `POST /api/members/login`
- JWT role `member` (separate from staff `waiter` / `admin`)
- Staff `/api/*` requires waiter|admin; members cannot hit staff APIs

### Member takeout
- `/api/member/takeout/...` — create, resume (`/active`), history, items, confirm, quit, cashless checkout
- One live takeout per member (`open` / `preparing` / `ready`)
- History: completed / cancelled takeout on member home
- Same staff queue as guest/counter takeout

### UI
- `/member/login` — register / login  
- `/member` — member home (CTA + active order resume)  
- `/member/takeout` — account takeout (resumes via `/active`)  
- Guest `/takeout` — intro (“how it works”) → `/takeout/order`  
- Table QR `/t/:token` — intro before menu (skipped if resuming)

### Demo
- `member@pos.local` / `member123`

---

## Explicitly out of 4.0 (still later)

- Loyalty / rewards on table QR  
- Member counter-order  
- Social login  

---

## How to try

1. Restart API (creates `members` + demo member)  
2. http://localhost:5173/member/login → login  
3. Add → confirm → pay → status  
4. Open another browser / clear nothing — login again → order resumes  
5. Staff **Takeout** queue still sees the order  

---

## One-line summary

**Members log in to own takeout (resume anywhere); dine-in stays table QR; loyalty waits for a later version.**
