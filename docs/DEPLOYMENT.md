# SeaShell Deployment Guide

This guide covers deploying SeaShell nodes and onboarding a new authority into the network.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.21+ |
| Linux / macOS | Any modern version |
| Docker + Compose | Optional but recommended |
| Disk | 512 MB minimum for chain data |
| Network | Public HTTPS URL required for branch nodes |

---

## Network Topology

```
                    ┌─────────────────────────────┐
                    │        PRIMARY NODE          │
                    │  IS_PRIMARY=true             │
                    │  Owns node registry          │
                    │  Approves nodes & authorities │
                    └──────┬──────────────┬─────────┘
                           │              │
                   P2P (flat broadcast — all peers equal)
                           │              │
              ┌────────────▼───┐  ┌───────▼────────────┐
              │  BRANCH NODE A │  │  BRANCH NODE B     │
              │  Authority A   │  │  Authority B       │
              │  IS_PRIMARY=   │  │  IS_PRIMARY=false  │
              │  false         │  │                    │
              └────────────────┘  └────────────────────┘
```

All nodes share the same blockchain. User data (accounts, wallets, tickets) lives on the authority's own node.

---

## Environment Variables

### Primary node

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SUPERADMIN_PASSWORD` | **Yes** | — | Bootstrap superadmin password. Server panics if unset. |
| `SUPERADMIN_EMAIL` | No | `superadmin@seashell.local` | Superadmin email |
| `PORT` | No | `8080` | HTTP listen port |
| `JWT_SECRET` | No | insecure default | HS256 signing key — **must change in production** |
| `DB_PATH` | No | `./db` | Base path for BadgerDB files |
| `ACCESS_TOKEN_MINUTES` | No | `15` | JWT access token lifetime |
| `REFRESH_TOKEN_DAYS` | No | `7` | Refresh token lifetime |
| `NODE_URL` | No | `` | This node's public base URL |
| `IS_PRIMARY` | **Yes** | `false` | Set `true` on the genesis node only |
| `NODE_SECRET` | No | insecure default | Shared P2P secret — **must change in production** |
| `KEY_ENCRYPTION_KEY` | No | `` | 32-byte hex AES-256-GCM key to encrypt private keys at rest |

### Branch node (authority node)

All primary variables apply, plus:

| Variable | Required | Description |
|----------|----------|-------------|
| `IS_PRIMARY` | **Yes** | Set `false` |
| `PRIMARY_NODE_URL` | **Yes** | Full URL of the primary node, e.g. `https://primary.seashell.network` |
| `NODE_URL` | **Yes** | This node's own public URL, e.g. `https://node.authority-a.com` |
| `NODE_SECRET` | **Yes** | Must match the value on every other node in the network |

---

## Build

```bash
git clone <repo-url>
cd seashell
make build
# Binary at: bin/seashell
```

---

## Deploying the Primary Node

The primary node is the genesis node. Deploy it first; all branch nodes connect to it.

### `.env` (primary)

```bash
SUPERADMIN_PASSWORD=<strong-random-password>
SUPERADMIN_EMAIL=admin@primary.seashell.network
JWT_SECRET=<64-char-random-hex>
PORT=8080
DB_PATH=/var/lib/seashell/db

NODE_URL=https://primary.seashell.network
IS_PRIMARY=true
NODE_SECRET=<shared-secret-distribute-to-all-nodes>
KEY_ENCRYPTION_KEY=<32-byte-hex>   # optional but recommended in production
```

```bash
docker compose up -d
```

On first start the primary node:
1. Creates `$DB_PATH/api` (user/authority store) and `$DB_PATH/blocks` (blockchain)
2. Creates a genesis block
3. Creates the superadmin account
4. Self-registers as the primary node
5. Starts the health checker

Verify it is running:

```bash
curl https://primary.seashell.network/api/v1/health
# → {"success":true,"data":{"status":"ok"}}
```

---

## Deploying a Branch Node (New Authority)

### What you need from the primary operator

Before you start, obtain from the primary operator:
- The primary node's public URL (`PRIMARY_NODE_URL`)
- The shared `NODE_SECRET`

These are given to you out-of-band (secure channel). Every node in the network uses the same `NODE_SECRET`.

### Step 1 — Provision a server with a public HTTPS URL

Your node must be publicly reachable for P2P communication. Example: `https://node.authority-a.com`.

> **TLS note:** SeaShell does not terminate TLS itself. Put nginx or Caddy in front. See the Reverse Proxy section below.

### Step 2 — Create your `.env`

```bash
SUPERADMIN_PASSWORD=<strong-password-for-your-node>
SUPERADMIN_EMAIL=admin@authority-a.com
JWT_SECRET=<64-char-random-hex>     # independent from primary's JWT secret
PORT=8080
DB_PATH=/var/lib/seashell/db

# P2P — these MUST match across all nodes
NODE_URL=https://node.authority-a.com
PRIMARY_NODE_URL=https://primary.seashell.network
IS_PRIMARY=false
NODE_SECRET=<shared-secret-from-primary-operator>
KEY_ENCRYPTION_KEY=<32-byte-hex>    # recommended
```

### Step 3 — Start the node

```bash
docker compose up -d
```

On first start, the node:
1. Sends `POST /api/v1/nodes/register` to the primary with your `NODE_URL` and authority details
2. Logs: `node: branch node not yet approved. Waiting for approval via API.`
3. Is running but not yet connected to the blockchain — it will not broadcast or receive blocks until approved

### Step 4 — Get approved by the primary superadmin

The primary superadmin goes to: **SuperAdmin → Node Requests → Approve**

This creates your Node record with `status=active`.

### Step 5 — Restart your node

```bash
docker compose restart api
```

On restart, the node detects it is approved and:
1. Syncs the full blockchain from the primary (`IncrementalSync`)
2. Starts the health checker (pings all peers every 30 seconds)
3. Logs: `node: health checker started`

Your node is now a live peer. New blocks from any node are broadcast to yours; blocks you create are broadcast to all peers.

---

## Authority Onboarding (Business Side)

After the node is live, the authority needs to be registered in the ecosystem.

### Step 1 — Create an authority request

A user on your node submits an authority request:

```
Dashboard → Authority → Request Authority
  { name: "Authority A", description: "..." }
```

Or via API:
```bash
curl -X POST https://node.authority-a.com/api/v1/user/authority \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Authority A", "description": "Regional authority for ..."}'
```

### Step 2 — Primary admin approves the authority

The primary admin approves the request and sets pricing parameters:

```
Primary Node → Admin → Authority Requests → Approve
  { base_price: 1.0, sensitivity_k: 0.5 }
```

On approval:
- An ECDSA P256 validator keypair is generated for your authority
- The public key is registered in the blockchain's validator set
- Your authority participates in round-robin block signing from the next block onward
- The new validator key is broadcast to all nodes in the network

### Step 3 — Configure user verification

As the authority owner, set up the verification form that your users must complete before getting a wallet:

```
Your Node → Dashboard → Authority → Verification → Configure Fields
  { method: "manual", fields: [
      { name: "national_id", label: "National ID Number", input_type: "text", required: true },
      { name: "dob",         label: "Date of Birth",      input_type: "text", required: true }
  ]}
```

### Step 4 — Onboard users

Generate invitation codes and distribute them to your users:

```
Your Node → Dashboard → Authority → Invitations → Create Invitation
```

Users register on **your node**, join via the invitation code, then submit the verification form. You review submissions and approve them.

### Step 5 — Users create wallets and transact

After verification approval, each user:
1. Creates a wallet → an **IdentityProof** is automatically issued (authority-signed, stored on-chain in every transaction)
2. Shares their wallet address with others
3. Sends and receives SHELL — transactions appear on every node's blockchain simultaneously

---

## Docker Compose Reference

### Primary node

```yaml
# docker-compose.yml (no changes needed — IS_PRIMARY=true in .env)
```

```bash
# .env
IS_PRIMARY=true
NODE_URL=https://primary.seashell.network
NODE_SECRET=<shared-secret>
SUPERADMIN_PASSWORD=<password>
JWT_SECRET=<hex>
```

### Branch node

```yaml
# Same docker-compose.yml
```

```bash
# .env
IS_PRIMARY=false
NODE_URL=https://node.authority-a.com
PRIMARY_NODE_URL=https://primary.seashell.network
NODE_SECRET=<same-shared-secret>
SUPERADMIN_PASSWORD=<password>
JWT_SECRET=<independent-hex>
```

---

## Running as a systemd Service

Create `/etc/systemd/system/seashell.service`:

```ini
[Unit]
Description=SeaShell Node
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
    server_name node.authority-a.com;

    ssl_certificate     /etc/letsencrypt/live/node.authority-a.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/node.authority-a.com/privkey.pem;

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

## Database Layout

```
$DB_PATH/
├── api/      # Users, authorities, tickets, invitations, identity proofs, value records
└── blocks/   # Blockchain blocks, UTXO index, validator list (replicated across all nodes)
```

**Backup:** Stop the server and copy the entire `$DB_PATH` directory. BadgerDB does not support hot backups with open write transactions.

---

## P2P Communication

All node-to-node communication is plain HTTP authenticated by the `X-Node-Secret` header:

| Endpoint | Purpose |
|----------|---------|
| `POST /p2p/v1/ping` | Health check — updates `LastSeenAt` and `BlockHeight` |
| `GET /p2p/v1/chain/sync?from=N&limit=M` | Download blocks from height N (max 500 per request) |
| `POST /p2p/v1/blocks` | Receive a newly broadcast block |
| `GET /p2p/v1/validators` | Get registered validator list |
| `POST /p2p/v1/validators` | Receive a newly registered validator key |
| `GET /p2p/v1/peers` | Get list of active peers |

> **Firewall:** Port 8080 (or your `$PORT`) must be reachable by all other nodes for P2P to work.

---

## Summary: New Authority Checklist

```
Infrastructure                              Business
──────────────────────────────────────────  ────────────────────────────────────────
[ ] Provision server with public HTTPS URL  [ ] Get authority name approved
[ ] Get NODE_SECRET from primary operator   [ ] Set up verification form
[ ] Configure .env with PRIMARY_NODE_URL    [ ] Create invitation codes
[ ] docker compose up -d                    [ ] Onboard users
[ ] Primary superadmin approves node        [ ] Review verification submissions
[ ] Restart node → blockchain syncs         [ ] Users create wallets & transact
[ ] Confirm health checker starts           [ ] Monitor via /admin/network
```

---

## Security Notes

- Change `JWT_SECRET` to a cryptographically random value (≥ 32 bytes) before production.
- `NODE_SECRET` is the only authentication between nodes — use a strong random value and distribute it securely. Treat it like a network password.
- `SUPERADMIN_PASSWORD` should be stored in a secrets manager.
- Set `KEY_ENCRYPTION_KEY` in production to encrypt all private keys at rest (AES-256-GCM).
- The server does not terminate TLS; use nginx or Caddy in front.
- Rate limiting is not built in; configure it at the reverse proxy.
- Each authority node's JWT secret is independent — tokens from one node are not valid on another. This is by design (federated model).

---

## Consensus: Proof of Authority

Blocks are sealed by registered authority validators in round-robin order (by block height):

```
Block height % len(validators) → selects which authority signs the block
```

When an admin approves an authority:
1. An ECDSA P256 keypair is generated server-side
2. The public key is added to the blockchain's validator set
3. The new validator is broadcast to all nodes
4. The private key is stored encrypted in the API database

No mining. No fees. Deterministic block selection.

---

## Value Engine

The internal SHELL price per authority uses the **Velocity of Money** formula:

```
price    = base_price × (1 + sensitivity_k × velocity)
velocity = shells_moved_last_50_blocks / circulating_supply_of_authority_members
```

- `base_price` and `sensitivity_k` are set at authority approval time.
- Recalculated after every block involving authority members.
- Query: `GET /api/v1/authority/value` (owner) or `GET /api/v1/value` (public).

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
