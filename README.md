# POS System (practice)

Phase 1: dine-in POS with Go API + Vue 3 frontend.

**Progress write-up:** [docs/phase-1-summary.md](docs/phase-1-summary.md) · [docs/phase-2-summary.md](docs/phase-2-summary.md)

## Demo accounts

| Email | Password | Role |
|-------|----------|------|
| `admin@pos.local` | `admin123` | admin |
| `waiter@pos.local` | `waiter123` | waiter |

## Run backend

```powershell
cd backend
go run ./cmd/server
```

API: `http://localhost:8080`  
SQLite DB: `backend/data/pos.db` (created on first run, seeded with tables 1–8 and sample menu).

## Run frontend

```powershell
cd frontend
npm install
npm run dev
```

App: `http://localhost:5173`

Optional: set `VITE_API_BASE` if the API is not on `http://localhost:8080`.

## Phase 1 flow

1. Login as waiter → **Tables** → pick a table → add menu items → **Pay** (mock cash/card/qr/wallet).
2. View **Receipt** (print via browser).
3. Login as admin → **Menu** / **Reports**.

Payments use a `PaymentGateway` interface with a mock provider (`provider=mock` in the database).
