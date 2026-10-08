# Phase 1 — Progress Summary

**Status:** Done (dine-in staff POS, usable for practice)  
**Stack:** Go API + Vue 3 + SQLite  
**Date snapshot:** 2026-10-02

When Phase 2 is finished, add a separate Phase 2 summary that links back here.

---

## Goal (Phase 1)

Build a **dine-in-only** learning POS where staff can take table orders, mock-pay, view receipts, manage menu/sold-out, and see basic reports — with a boring functional UI to redesign later.

---

## What shipped

### Auth & roles
- Staff login / logout (JWT)
- Roles: `waiter`, `admin`
- Admin-only: menu CRUD, reports
- Waiter + admin: tables, orders, sold-out toggle, checkout

### Tables & orders (dine-in)
- Seeded dining tables (1–8)
- **Create-on-first-item:** free table opens a **draft**; DB order is created only when the first item is added
- Open table continues an existing unpaid order
- Add items with **quantity** (stepper / input)
- Change qty / **remove** lines on an open order
- Removing the **last** item cancels the order → table returns to **Free**
- Cancel whole order (before payment)
- Checkout with mock payment methods: `cash` | `card` | `qr` | `wallet`
- Receipt view + browser print; **Back** depends on entry (`?from=reports|order`)

### Menu
- Admin create / soft-delete menu items
- Sold-out toggle (waiter + admin)
- Sold-out blocks **new** adds; existing order lines keep **price/name snapshots**
- Seed menu: Japanese / French / Italian names, **JPY** (whole yen in `price_cents`)

### Payments
- `PaymentGateway` interface + `MockGateway` (`provider=mock`)
- Ready to swap a real adapter later without rewriting checkout

### Admin reports
- Date-range summary (order count, total ¥)
- **Transactions table** (main view) + receipt links
- Item sales table (qty / revenue, most/least cues)

### UI stance
- Basic functional UI only; redesign deferred
- Occupied tables: Free vs Open (green status cue), not “selected/greyed out”

---

## Important design decisions (learned in Phase 1)

| Topic | Decision |
|-------|----------|
| Venue | Dine-in only (no takeout / reservation / loyalty yet) |
| Inventory | Sold-out toggle only (no ingredient stock) |
| Money | Integer yen; display as JPY |
| Order creation | Persist on first item, not on table click |
| Empty order | Not kept; last-item remove cancels order |
| Prices on checks | Snapshot on `order_items` |
| Payments | Mock behind interface |
| Multi-entry detail | Receipt `?from=` for correct Back |
| Admin ordering | Allowed (owner covering a shift) |

---

## Bugs / fixes worth remembering

1. Empty order returned `items: null` → Vue crash on blank order screen → always return `[]`
2. Nav Logout missing after login → reactive auth store
3. Table button looked “stuck selected” → focus + status styling (then Free/Open semantics clarified)
4. Create-on-click left **Open · ¥0** → moved to draft / create-on-first-item
5. Couldn’t remove items / add qty > 1 → PATCH item qty + menu qty steppers

---

## Demo accounts

| Email | Password | Role |
|-------|----------|------|
| `admin@pos.local` | `admin123` | admin |
| `waiter@pos.local` | `waiter123` | waiter |

## Run

```powershell
# API
cd backend
go run ./cmd/server

# Frontend (Node 18+)
nvm use 24.21.0
cd frontend
npm run dev
```

- API: http://localhost:8080  
- App: http://localhost:5173  

---

## Explicitly out of Phase 1

- Customer QR / self-order  
- Takeout + ready-for-pickup  
- Reservations, loyalty / points  
- Real payment providers / webhooks  
- Kitchen statuses (`preparing` / `served`) as a board  
- Polished branded UI  

---

## Suggested next

**Phase 2 — Customer QR dine-in** — **done.** See [phase-2-summary.md](./phase-2-summary.md).

## Doc map

| Doc | When |
|-----|------|
| [This file](./phase-1-summary.md) | Phase 1 complete |
| [phase-2-summary.md](./phase-2-summary.md) | Phase 2 complete |
| [phase-2.5-summary.md](./phase-2.5-summary.md) | Product snapshot (v2.5) |
| [phase-2.6-summary.md](./phase-2.6-summary.md) | **Current — categories CRUD** |
| [order-flow-decisions.md](./order-flow-decisions.md) | Living BA / status rules |
| Requirements / product brief | Prefer living decisions; expand when scoping a bigger phase |
