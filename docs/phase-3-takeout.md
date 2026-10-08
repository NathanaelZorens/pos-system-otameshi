# Phase 3 — Takeout / Pickup

**Status:** Built (see [phase-3-summary.md](./phase-3-summary.md) for 3.0–3.1 rollup)  
**Product version target:** 3.0  
**Builds on:** dine-in through [v2.6](./phase-2.6-summary.md)  
**Date:** 2026-10-06

---

## Mental model

| Lane | Physical | In the app |
|------|----------|------------|
| **Dine-in** | Seats / tables | Tables grid (unchanged) |
| **Takeout** | Pickup shelf / waiting area | **Separate takeout queue** |

Same kitchen food; **different queue**. Do not mix takeout into open tables.

---

## Locked decisions

### Entry
- **Dine-in:** table QR / staff tables (current) — not a dine-in↔takeout toggle on the table page.
- **Takeout:** own public link `/takeout` — guest first; logged-in **member** later (see [phase 4](./phase-4-members.md)).

### Who / pay
- **Guest takeout:** **pay first**, cashless only (`card` / `qr` / `wallet`) — same mock gateway idea as guest dine-in.
- **Guest cash / pay-at-pickup:** **out of v3** (no-show + trust complexity).
- **Staff takeout + cash at counter:** see [phase 3.1](./phase-3.1-counter-takeout.md) (built).

### Line rules (same as dine-in)
- Items start **`pending`** → guest/staff **Confirm** → **`confirmed`**.
- Confirm required before pay (uniformity with dine-in).

### Pay timing (different from dine-in)
- Dine-in: eat-then-pay (preparing can start on Confirm).
- Takeout: **Confirm locks lines → Pay → then kitchen progress** (don’t cook unpaid guest takeout).

### Order statuses (takeout)
| Status | Meaning |
|--------|---------|
| `open` | Building cart |
| `preparing` | Paid; being made |
| `ready` | Bagged; waiting for pickup (**not** dine-in’s `served`) |
| `completed` | Handed to customer (staff) |
| `cancelled` | Quit / force cancel |

After successful pay, order enters **`preparing`** (no separate `paid` status on the takeout path).

### Staff actions
- Mark **Ready** when bag is on the pickup shelf.
- Mark **Completed / Picked up** when handed over.
- Force cancel: same as dine-in (`open` / `preparing` / `ready`).

### Guest UX
- Name (optional) + **pickup code** for calling out.
- After pay: status page (preparing → ready) — polls until completed.
- No self-complete; staff marks completed.

---

## What shipped

### API
- Guest: `GET/POST /api/guest/takeout/...` (menu, create, items, confirm, quit, cashless checkout)
- Staff: `GET /api/takeout/queue`, `POST /api/takeout/{id}/ready`, `POST /api/takeout/{id}/complete`
- Schema: nullable `dining_table_id`, `source=takeout`, `pickup_code`, statuses `ready` / `completed`

### UI
- Guest: http://localhost:5173/takeout  
- Staff: **Takeout** nav → `/takeout-queue`

---

## Explicitly out of v3

- Guest cash / pay on arrival  
- Staff takeout POS lane (incl. cash)  
- Logged-in customer / loyalty  
- Delivery / third-party apps  
- Promotions  
- Full Pen UI redesign  

---

## One-line summary

**Guest opens takeout link → add → confirm → pay cashless → preparing → staff marks ready → customer picks up → staff marks completed — on a queue separate from tables.**
