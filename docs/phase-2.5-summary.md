# Version 2.5 — Progress Summary

**Product version:** 2.5  
**Status:** Done (dine-in staff + guest QR with eat-then-pay, fulfillment, guest claim)  
**Stack:** Go API + Vue 3 + SQLite  
**Date snapshot:** 2026-10-05  

**Builds on:** [Phase 1](./phase-1-summary.md) · [Phase 2 guest QR baseline](./phase-2-summary.md)  
**Living rules:** [order-flow-decisions.md](./order-flow-decisions.md)

---

## What “2.5” means

Phase 2 delivered **guest QR ordering** on free tables. **Version 2.5** is the current learning POS: same stack, plus the **order lifecycle** you designed after Phase 2 (confirm, quit, force cancel, kitchen-style order status, guest resume vs block).

Think of it as **Phase 2 + order-flow hardening**, not a new venue (still dine-in only).

---

## Goal (v2.5)

One coherent **eat-then-pay** dine-in flow for **staff and guest**, with explicit line vs order status, safe guest access to an open check, and staff as fallback when the guest loses their browser session.

---

## What shipped since Phase 2

### Eat-then-pay & line status
- Lines: **`pending`** → **`confirmed`** (batch **Confirm** on guest + waiter)
- **Pay blocked** while any line is still pending
- After confirm: lines locked (no qty change / remove); same dish again → **new pending line**
- Mid-meal: add more items → confirm again

### Cancel policy
- **Quit** (guest + waiter): only if **no confirmed lines** yet
- **Force cancel** (waiter/admin only): open / preparing / served, not paid; strong confirm in UI
- DB status `cancelled`; table frees for a new check

### Order-level fulfillment (not per dish)
- **`preparing`:** set automatically on **first Confirm**
- **`served`:** waiter **Mark served** on order screen
- Tables list shows order status badges (open / preparing / served)
- Guest can pay without served (policy choice documented in order-flow-decisions)

### Guest UX sync
- Guest view **polls** (~2.5s) while order is open / preparing / served so **Mark served** updates without manual refresh

### Guest claim (resume vs block)
- Permanent table **`guest_token`** in QR (which table)
- Per-order **`guest_claim`** minted on guest create; stored in **`localStorage`**, sent as **`X-Guest-Claim`**
- Matching claim → **resume** after refresh / reopen tab
- Open order, no/wrong claim → **blocked** (“ask staff”)
- Claim cleared on pay / quit / cancel; **paid receipt** still readable without claim
- Lost claim (incognito closed, other phone, cleared data) → **staff completes** on waiter POS

---

## Already in Phase 1 + 2 (still true at 2.5)

| Area | Summary |
|------|---------|
| Staff | JWT, waiter vs admin, tables, orders, mock checkout, menu, reports |
| Guest | `/t/{guest_token}`, create-on-first-item, cashless mock pay, receipt route |
| Tables | One active dining order per table (`open` / `preparing` / `served`) |
| Money | JPY whole yen, line snapshots, mock `PaymentGateway` |

---

## Doc map (BA / design)

| Doc | Role |
|-----|------|
| [phase-1-summary.md](./phase-1-summary.md) | Staff POS foundation |
| [phase-2-summary.md](./phase-2-summary.md) | Guest QR entry (baseline) |
| **This file** | **Current product snapshot (v2.5)** |
| [order-flow-decisions.md](./order-flow-decisions.md) | Why: payment model, statuses, quit vs force, guest claim — update as you decide |

The decisions doc is intentional: minute rules (who can press what, when) belong there; this file is **what is built**.

---

## How to try v2.5 (smoke path)

1. Staff: table → open order → add items → **Confirm** → table shows **preparing**  
2. Staff: **Mark served** → guest tab (same browser) shows **served** within a few seconds  
3. Guest: QR on free table → add → **Confirm** → refresh → still on same order  
4. Guest: same QR in another browser → **blocked**  
5. Waiter: **Force cancel** on a preparing order → table free  
6. Pay (guest or staff) → next visit same QR → **new** order (paid check closed)

---

## Explicitly not in 2.5

- Staff **re-issue** guest claim
- Guest join staff-open order
- Item-level preparing/served, chef/KDS
- Void single confirmed line
- Takeout, real payments, UI redesign

**Built after 2.5 snapshot (ops):** admin **Cancelled** list — see [Cancelled ops](#cancelled-ops-admin) below if you extend this doc, or try `/admin/cancelled`.

---

## Cancelled ops (admin)

- Nav: **Cancelled** (admin only) → `/admin/cancelled`
- API: `GET /api/reports/cancelled?from=&to=`
- **Quit** / **Force cancel** require a free-text `reason` (stored as `cancel_reason`)
- Emptying the last line still cancels; reason may be blank
- Separate from **Reports** (paid sales only)

---

## Suggested next (v2.6+ ideas)

1. Staff re-issue guest claim / resume link  
2. Phase 3 takeout or kitchen board  
3. UI polish when you have design direction  

---

## Run

Same as Phase 1 — see [phase-1-summary.md](./phase-1-summary.md#run).
