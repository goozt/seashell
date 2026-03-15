# SeaShell UI — Example Web App

A **mobile-first** React/Next.js application for managing the SeaShell cryptocurrency ecosystem. Built with [shadcn/ui](https://ui.shadcn.com/) and [Tailwind CSS](https://tailwindcss.com/).

## Features

| Role | Capabilities |
|------|-------------|
| **Public** | Block explorer, live token prices |
| **User** | Wallet management, send SHELL, authority affiliation, support tickets |
| **Authority Owner** | Member management, invitation codes, price history, authority stats |
| **Admin** | Approve/reject authority requests, manage tickets, view all users |
| **Super Admin** | Create/demote admins, suspend/reinstate authorities, platform stats |

## Tech Stack

- **Framework**: Next.js 15 (App Router)
- **UI**: shadcn/ui + Tailwind CSS
- **State**: Zustand (auth) + TanStack Query (server state)
- **Auth**: JWT (access + refresh token pair, auto-refresh)
- **Design**: Mobile-first, bottom navigation bar

---

## Local Development

### 1. Prerequisites

- Node.js 18+
- The SeaShell Go backend running (see root `README.md`)

### 2. Install dependencies

```bash
cd example
npm install
```

### 3. Configure environment

```bash
cp .env.example .env.local
```

Edit `.env.local`:

```env
# Point to your running SeaShell backend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 4. Install shadcn/ui components

```bash
npm run setup
```

This runs `npx shadcn@latest init` and adds all required components to `src/components/ui/`.

> **Note**: The `src/components/ui/` directory already contains hand-written stubs that compile without shadcn. Running `npm run setup` will overwrite them with the official shadcn components (recommended for production).

### 5. Start the dev server

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

---

## Deploying to Vercel

### Option A — Vercel CLI (recommended)

```bash
# 1. Install Vercel CLI
npm i -g vercel

# 2. From the example/ directory
cd example
vercel

# 3. Follow the prompts:
#    - Link to your Vercel account
#    - Set the root directory to "example" (or deploy from inside it)
#    - Vercel auto-detects Next.js

# 4. Add environment variable in the Vercel dashboard:
#    NEXT_PUBLIC_API_URL = https://your-seashell-backend.com
```

### Option B — GitHub Integration

1. Push the repository to GitHub.
2. Go to [vercel.com/new](https://vercel.com/new) → **Import Git Repository**.
3. Set **Root Directory** to `example`.
4. Under **Environment Variables** add:
   ```
   NEXT_PUBLIC_API_URL = https://your-seashell-backend.com
   ```
5. Click **Deploy**.

### Option C — vercel.json (monorepo)

Create `example/vercel.json` if you need custom build settings:

```json
{
  "buildCommand": "npm run build",
  "outputDirectory": ".next",
  "installCommand": "npm install",
  "framework": "nextjs"
}
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | Yes | `http://localhost:8080` | Base URL of the SeaShell Go backend |

The Next.js `next.config.js` rewrites all `/api/*` requests to the backend URL, so CORS is not required on the frontend.

---

## Backend Requirements

Your SeaShell backend must have:

- **CORS** configured to allow the frontend origin (or use the proxy rewrite — no CORS needed in that case)
- The API running on the URL set in `NEXT_PUBLIC_API_URL`
- An initial superadmin account (create via the backend CLI or seed script)

---

## Project Structure

```
example/
├── src/
│   ├── app/
│   │   ├── page.tsx              # Public block explorer + token prices
│   │   ├── (auth)/
│   │   │   ├── login/page.tsx
│   │   │   └── register/page.tsx
│   │   ├── dashboard/            # Authenticated user area
│   │   │   ├── page.tsx          # Overview
│   │   │   ├── wallet/page.tsx   # Wallet balance + address
│   │   │   ├── transactions/     # Send + history
│   │   │   ├── authority/        # Affiliation + owner tools
│   │   │   └── tickets/          # Support tickets
│   │   ├── admin/                # Admin area (admin + superadmin)
│   │   │   ├── page.tsx          # Stats dashboard
│   │   │   ├── requests/         # Authority approval
│   │   │   ├── tickets/          # Ticket management
│   │   │   └── users/            # User list
│   │   └── superadmin/           # SuperAdmin area
│   │       └── page.tsx          # Admin + authority management
│   ├── components/
│   │   ├── layout/
│   │   │   ├── header.tsx        # Top bar with user info + logout
│   │   │   └── mobile-nav.tsx    # Bottom tab navigation (role-aware)
│   │   └── ui/                   # shadcn/ui components
│   ├── lib/
│   │   ├── api.ts                # Typed API client (auto token refresh)
│   │   └── utils.ts              # cn(), formatDate, truncateHash, etc.
│   ├── store/
│   │   └── auth.ts               # Zustand auth store (persisted)
│   └── types/
│       └── api.ts                # TypeScript types for all API models
├── .env.example
├── components.json               # shadcn/ui config
├── next.config.js                # API proxy rewrite
├── tailwind.config.js
└── package.json
```

---

## Authentication Flow

```
1. User registers or logs in → receives { access_token, refresh_token }
2. Tokens saved to localStorage via tokenStore
3. Every API request sends Authorization: Bearer <access_token>
4. On 401 → auto-refresh using refresh_token → retry original request
5. On refresh failure → clear tokens → redirect to /login
6. Logout → POST /auth/logout (revokes refresh token) → clear localStorage
```

---

## Mobile-First Design

- **Bottom navigation bar** fixed at the bottom with safe-area support
- Navigation items change based on user role (user / admin / superadmin)
- All pages constrained to `max-w-2xl` and centered for tablet/desktop
- Touch-friendly tap targets (minimum 44px)
- Cards and lists optimized for small screens
