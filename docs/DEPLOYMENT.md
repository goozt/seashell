# SeaShell Deployment Guide

This guide covers deploying the SeaShell API server. The CLI mode requires no server setup.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.18+ |
| Linux / macOS | Any modern version |
| Disk | 512 MB minimum for chain data |

---

## Build

```bash
# Clone and build
git clone <repo-url>
cd seashell
make build
# Binary: bin/seashell
```

---

## Environment Variables

The API server is configured entirely through environment variables. No configuration file is required.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SUPERADMIN_PASSWORD` | **Yes** | — | Initial superadmin account password (min 8 chars). Server panics if unset. |
| `SUPERADMIN_EMAIL` | No | `superadmin@seashell.local` | Superadmin email address |
| `PORT` | No | `8080` | HTTP listen port |
| `JWT_SECRET` | No | `changeme-secret` | HS256 signing secret. **Must be changed in production.** |
| `DB_PATH` | No | `./db` | Base directory for BadgerDB files |
| `ACCESS_TOKEN_MINUTES` | No | `15` | Access token lifetime in minutes |
| `REFRESH_TOKEN_DAYS` | No | `7` | Refresh token lifetime in days |

### Minimum production `.env`

```bash
SUPERADMIN_PASSWORD=<strong-random-password>
SUPERADMIN_EMAIL=admin@yourdomain.com
JWT_SECRET=<64-char-random-hex>
PORT=8080
DB_PATH=/var/lib/seashell/db
```

---

## Running the API Server

```bash
# Start in API mode
SUPERADMIN_PASSWORD=secret JWT_SECRET=myjwtsecret bin/seashell --api

# With a .env file (using a helper like direnv or dotenv)
dotenv bin/seashell --api
```

On first start the server:
1. Opens (or creates) BadgerDB databases at `$DB_PATH/api` and `$DB_PATH/blocks`
2. Creates the superadmin account if one does not exist
3. Starts listening on `$PORT`

---

## Database Layout

Two separate BadgerDB instances are created under `$DB_PATH`:

```
$DB_PATH/
├── api/          # Users, authorities, tickets, invitations, value records, tokens
└── blocks/       # Blockchain blocks, UTXO index, validator list
```

**Backup**: Stop the server and copy the entire `$DB_PATH` directory. BadgerDB does not support hot backups while a write transaction is open.

---

## Running as a systemd Service

Create `/etc/systemd/system/seashell.service`:

```ini
[Unit]
Description=SeaShell API Server
After=network.target

[Service]
Type=simple
User=seashell
WorkingDirectory=/opt/seashell
EnvironmentFile=/etc/seashell/env
ExecStart=/opt/seashell/bin/seashell --api
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

`/etc/seashell/env`:
```
SUPERADMIN_PASSWORD=<strong-password>
SUPERADMIN_EMAIL=admin@yourdomain.com
JWT_SECRET=<64-char-hex>
PORT=8080
DB_PATH=/var/lib/seashell/db
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now seashell
sudo journalctl -u seashell -f
```

---

## Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl;
    server_name api.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    location / {
        proxy_pass         http://127.0.0.1:8080;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }
}
```

---

## First-Run Checklist

1. Start the server — verify it logs `SeaShell API server listening on :8080`
2. Check the superadmin account was created:
   ```bash
   curl http://localhost:8080/api/v1/health
   # → {"success":true,"data":{"status":"ok",...}}
   ```
3. Log in as superadmin:
   ```bash
   curl -s -X POST http://localhost:8080/api/v1/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"superadmin@seashell.local","password":"<your-password>"}'
   ```
4. Use the returned `access_token` as `Authorization: Bearer <token>` on protected endpoints.

---

## User Onboarding Flow

```
Register  →  Login  →  Create Authority Request  →  (Admin Approves)
         or                                                ↓
         →  Join Authority via Invitation Code   →  Generate Wallet  →  Transact
```

1. **Register**: `POST /api/v1/auth/register` `{username, email, password}`
2. **Login**: `POST /api/v1/auth/login` → `{access_token, refresh_token}`
3. **Create authority**: `POST /api/v1/user/authority` `{name, description}` — awaits admin approval
4. **Or join existing**: `POST /api/v1/user/authority/join` `{code}` — immediate if code is valid
5. **Generate wallet**: `POST /api/v1/user/wallet` — creates blockchain keypair
6. **Transact**: `POST /api/v1/user/transactions` `{to_address, amount}`

---

## Admin Operations

| Task | Endpoint |
|------|----------|
| List pending authority requests | `GET /api/v1/admin/authority-requests` |
| Approve authority | `POST /api/v1/admin/authority-requests/:id/approve` `{base_price, sensitivity_k}` |
| Reject authority | `POST /api/v1/admin/authority-requests/:id/reject` `{reason}` |
| List all tickets | `GET /api/v1/admin/tickets` |
| Update ticket status | `PUT /api/v1/admin/tickets/:id` `{status}` |
| List all users | `GET /api/v1/admin/users` |

---

## SuperAdmin Operations

| Task | Endpoint |
|------|----------|
| Create admin | `POST /api/v1/superadmin/admins` `{username, email, password}` |
| Demote admin | `DELETE /api/v1/superadmin/admins/:id` |
| Suspend authority | `POST /api/v1/superadmin/authorities/:id/suspend` |
| Reinstate authority | `POST /api/v1/superadmin/authorities/:id/reinstate` |

---

## Token Lifecycle

- **Access token**: 15-minute JWT (HS256). Send as `Authorization: Bearer <token>`.
- **Refresh token**: 7-day opaque token. Use `POST /api/v1/auth/refresh {refresh_token}` to get a new access token.
- **Logout**: `POST /api/v1/auth/logout {refresh_token}` revokes the refresh token immediately.

---

## Consensus: Proof of Authority

Blocks are sealed by registered authority validators in round-robin order (by block height). When an admin approves an authority:

1. An ECDSA P256 keypair is generated server-side for the authority.
2. The public key is registered as a blockchain validator.
3. The private key (D bytes) is stored in the API database.

When a transaction is submitted, the server identifies the current validator, signs the block, and adds it to the chain — no mining required.

---

## Value Engine

The internal SHELL price is calculated per authority using the **Velocity of Money** formula:

```
price = base_price × (1 + sensitivity_k × velocity)
velocity = shells_moved_last_50_blocks / circulating_supply_of_authority_members
```

- `base_price` and `sensitivity_k` are set by the admin at authority approval.
- A new `ValueRecord` is stored after every block sealed by that authority.
- Query current price: `GET /api/v1/authority/value` (authority owner) or `GET /api/v1/value` (public, system-wide average).

---

## CLI Mode

The original blockchain CLI still works without any server:

```bash
bin/seashell wallet              # create wallet
bin/seashell create -a <addr>    # initialize chain
bin/seashell send -from F -to T -amount N
bin/seashell balance -a <addr>
bin/seashell list
```

CLI and API share the same `./db/blocks` directory. **Do not run both concurrently** — BadgerDB does not support multi-process access.

---

## Security Notes

- Change `JWT_SECRET` to a cryptographically random value (≥ 32 bytes) before production use.
- `SUPERADMIN_PASSWORD` should be a strong random password stored in a secrets manager.
- Authority validator private keys are stored as raw bytes in BadgerDB. For production, consider encrypting the database at rest.
- The server does not implement TLS directly; terminate TLS at the reverse proxy.
- Rate limiting is not built in; configure it at the reverse proxy or load balancer level.
