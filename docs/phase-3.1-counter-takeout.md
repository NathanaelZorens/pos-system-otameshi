# Phase 3.1 — Counter-order takeout

**Status:** Built  
**Builds on:** [Phase 3 self-order takeout](./phase-3-takeout.md)  
**Date:** 2026-10-06

---

## Framing

| Door | Who | Pay | Status for customer |
|------|-----|-----|---------------------|
| **Self-order** | Guest `/takeout` | Cashless only | Phone (code + status) |
| **Counter-order** | Staff (waiter/admin) | Cash + card/QR/wallet | Ask staff / paper later |

Same kitchen queue (`preparing` → `ready` → `completed`). No new nav tab — **New takeout** on the Takeout queue page.

## Locked rules

- Pay first, then kitchen (same as self-order).
- Confirm lines before pay.
- Pickup code on create.
- `source = takeout`; staff sets `waiter_id` (self-order leaves it null).
- Receipt print behavior: **out of this slice**.

## What shipped

- `POST /api/takeout/orders` (staff JWT)
- Staff checkout via `POST /api/orders/{id}/checkout` → `preparing` (any method)
- UI: Takeout → **New takeout** → `/takeout-queue/new` → order → queue

## Explicitly out

- Auto-print / receipt UX polish  
- Guest cash / pay later  
- Logged-in remote order  

---

## One-line

**Staff taps New takeout → add → confirm → pay (any method) → same Takeout queue as QR orders.**
