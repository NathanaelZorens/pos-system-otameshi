# How to run FE & BE

Quick reference for local dev. Use **two terminals** (API + frontend).

Paths assume the repo root: `pos-system-otameshi`.

---

## Backend (Go API)

```powershell
cd backend
go run ./cmd/server
```

- URL: http://localhost:8080  
- Health: http://localhost:8080/api/health  
- SQLite DB: `backend/data/pos.db` (created on first run)

Optional env:

| Variable | Default | Meaning |
|----------|---------|---------|
| `ADDR` | `:8080` | Listen address |
| `DB_PATH` | `data/pos.db` | SQLite file path |
| `JWT_SECRET` | `dev-secret-change-me` | JWT signing secret |

---

## Frontend (Vue + Vite)

Needs **Node 18+** (this project has used nvm `24.21.0`).

```powershell
cd frontend
nvm use 24.21.0
npm install
npm run dev
```

- App: http://localhost:5173  
- Guest takeout intro: http://localhost:5173/takeout → Start → `/takeout/order`  
- Member home: http://localhost:5173/member/login → `/member`  
- Staff takeout queue: http://localhost:5173/takeout-queue (after login)

`npm install` only needed the first time, or after dependency changes.

Optional: set API base if not localhost:8080:

```powershell
$env:VITE_API_BASE = "http://localhost:8080"
npm run dev
```

---

## Demo logins

| Email | Password | Role |
|-------|----------|------|
| `admin@pos.local` | `admin123` | admin (staff) |
| `waiter@pos.local` | `waiter123` | waiter (staff) |
| `member@pos.local` | `member123` | member (customer takeout) |

Member UI: http://localhost:5173/member/login

---

## Typical daily start

**Terminal 1 — API**

```powershell
cd d:\exceptions\Other\pos-system-otameshi\backend
go run ./cmd/server
```

**Terminal 2 — UI**

```powershell
cd d:\exceptions\Other\pos-system-otameshi\frontend
nvm use 24.21.0
npm run dev
```

Then open http://localhost:5173 and log in.
