# Architecture

SeaShell is a Proof of Authority cryptocurrency with a UTXO transaction model, a multi-role REST API, a P2P node network, and a Next.js management UI. This document describes how all the pieces fit together.

---

## System Overview

```
                     ┌────────────────────────────────────────────────┐
                     │                 PRIMARY NODE                   │
                     │                                                │
                     │   ┌──────────┐  ┌────────────┐  ┌───────────┐  │
                     │   │ REST API │  │ Blockchain │  │   Store   │  │
                     │   │  (chi)   │──│ (BadgerDB) │  │ (BadgerDB)│  │
                     │   └────┬─────┘  └────────────┘  └───────────┘  │
                     │        │   Node Registry (source of truth)     │
                     └────────┼───────────────────────────────────────┘
                  ┌───────────┴─────────────┐
                  │    P2P HTTP Layer       │
                  │  (block broadcast,      │
                  │   peer discovery,       │
                  │   chain sync)           │
        ┌─────────┴───────────┐   ┌─────────┴───────────┐
        │     NODE A          │   │     NODE B          │
        │  (Authority 1)      │   │  (Authority 2)      │
        │                     │   │                     │
        │  ┌───────────────┐  │   │  ┌───────────────┐  │
        │  │ REST API      │  │   │  │ REST API      │  │
        │  │ + local users │  │   │  │ + local users │  │
        │  └───────────────┘  │   │  └───────────────┘  │
        │  ┌───────────────┐  │   │  ┌───────────────┐  │
        │  │  Blockchain   │  │   │  │  Blockchain   │  │
        │  │  (synced)     │  │   │  │  (synced)     │  │
        │  └───────────────┘  │   │  └───────────────┘  │
        └─────────────────────┘   └─────────────────────┘
```

**Key principles:**

- The blockchain is replicated identically across every active node
- User accounts, wallets, and tickets are local to each node
- The primary node owns the node registry and approves new members
- P2P communication is plain HTTP REST authenticated by a shared secret
- New blocks are pushed to all peers; new nodes pull the full chain on join

---

## Directory Layout

```
seashell/
├── main.go                  # Entry point: --api flag or CLI mode
├── config/
│   └── config.go            # Environment-based configuration
├── model/                   # Data structs (no logic)
│   ├── user.go
│   ├── authority.go
│   ├── node.go
│   ├── ticket.go
│   ├── invitation.go
│   ├── verification.go
│   └── value.go
├── store/                   # BadgerDB persistence (API database)
│   ├── db.go                # Open/close, generic get/set/iterate
│   ├── helpers.go           # Key-building utilities
│   ├── user_store.go
│   ├── authority_store.go
│   ├── node_store.go
│   ├── ticket_store.go
│   ├── invitation_store.go
│   ├── value_store.go
│   ├── verification_store.go
│   └── refresh_store.go
├── blockchain/              # Core chain: blocks, transactions, PoA
│   ├── block.go             # Block struct, GOB serialization
│   ├── chain.go             # Chain operations, UTXO queries
│   ├── chain_ext.go         # Extended methods (external blocks, height queries)
│   ├── transaction.go       # Transaction creation, signing, verification
│   ├── txio.go              # TxInput / TxOutput structs
│   ├── authority.go         # Validator management, round-robin selection
│   ├── proof.go             # Legacy PoW (unused in PoA mode)
│   └── utils.go             # Signature encoding, error handling
├── wallet/                  # ECDSA key pairs and address generation
│   ├── wallet.go            # Key generation (P256)
│   ├── address.go           # Address format, Base58, checksums
│   ├── db.go                # GOB-file wallet persistence (CLI mode)
│   └── utils.go
├── service/                 # Business logic
│   ├── auth_service.go      # JWT tokens, password hashing, superadmin bootstrap
│   ├── chain_service.go     # Thread-safe blockchain operations
│   ├── node_service.go      # Node lifecycle, P2P orchestration
│   ├── value_service.go     # Velocity-of-Money price engine
│   └── verification_service.go # Modular user verification system
├── api/
│   ├── server.go            # chi router, service wiring, server bootstrap
│   ├── response/
│   │   └── response.go      # Standardized JSON envelope
│   ├── middleware/
│   │   ├── auth.go          # JWT extraction, role enforcement
│   │   └── node.go          # X-Node-Secret validation
│   └── handlers/
│       ├── auth.go          # Register, login, refresh, logout
│       ├── public.go        # Blocks, health, value
│       ├── user.go          # Wallet, transactions, tickets, authority join
│       ├── authority_owner.go # Members, invitations, authority stats
│       ├── admin.go         # Authority requests, user list, ticket management
│       ├── superadmin.go    # Admin CRUD, authority suspension
│       ├── node.go          # P2P endpoints, node request management
│       └── verification.go  # Modular verification config + submission handlers
├── node/                    # P2P networking
│   ├── peer.go              # HTTP client for talking to peers
│   ├── sync.go              # Full and incremental chain sync
│   ├── broadcast.go         # Block and validator fan-out
│   ├── healthchecker.go     # Background peer pinger
│   └── block_wire.go        # JSON serialization for blocks over HTTP
├── cli/                     # Command-line interface
│   ├── cli.go               # Arg parsing, command dispatch
│   ├── chainCmd.go          # create, balance, send, list, addvalidator
│   └── walletCmd.go         # wallet, walletlist
├── example/                 # Next.js management UI
│   ├── src/app/             # App Router pages
│   ├── src/lib/api.ts       # API client with auto-refresh
│   ├── src/types/api.ts     # TypeScript type definitions
│   ├── src/store/auth.ts    # Zustand auth state
│   └── src/components/      # Shared UI components
├── docs/
│   └── DEPLOYMENT.md
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── init-env.sh
```

---

## Entry Points

The binary runs in one of two modes:

```
bin/seashell              # CLI mode (default)
bin/seashell --api        # API server mode
```

**CLI mode** provides direct blockchain interaction: create chains, generate wallets, send transactions, list blocks, register validators.

**API mode** starts a full HTTP server with JWT authentication, role-based access control, P2P networking, and serves as the backend for the Next.js UI.

Both modes operate on the same BadgerDB at `$DB_PATH/blocks`. They must never run concurrently — BadgerDB does not support multi-process access.

---

## Configuration

All configuration is loaded from environment variables by `config.Load()`.

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `SUPERADMIN_PASSWORD` | — | **Yes** | Bootstrap password (panics if missing) |
| `SUPERADMIN_EMAIL` | `superadmin@seashell.local` | No | Bootstrap email |
| `PORT` | `8080` | No | HTTP listen port |
| `JWT_SECRET` | insecure default | No | HS256 signing key |
| `DB_PATH` | `./db` | No | Base directory for all databases |
| `ACCESS_TOKEN_MINUTES` | `15` | No | JWT access token lifetime |
| `REFRESH_TOKEN_DAYS` | `7` | No | Refresh token lifetime |
| `NODE_URL` | `` | No | This node's publicly reachable URL |
| `PRIMARY_NODE_URL` | `` | No | Primary node URL (omit if this IS primary) |
| `IS_PRIMARY` | `false` | No | Set `true` on the genesis node |
| `NODE_SECRET` | insecure default | No | Shared P2P authentication secret |

The database directory layout under `$DB_PATH`:

```
$DB_PATH/
├── api/             # User accounts, authorities, tickets, etc. (BadgerDB)
├── blocks/          # Blockchain data (BadgerDB)
└── wallets.data     # CLI wallet file (GOB-encoded)
```

---

## Blockchain Layer

### Block Structure

```go
type Block struct {
    Timestamp    uint
    PrevHash     []byte
    Transactions []*Transaction
    Hash         []byte          // SHA256(PrevHash || TxRoot || Timestamp || Height)
    Height       uint64
    Validator    []byte          // 64-byte ECDSA P256 public key
    Signature    []byte          // 64-byte ECDSA signature (r || s)
}
```

Blocks are GOB-encoded for storage in BadgerDB.

**Storage keys:**

| Key | Value |
|-----|-------|
| `{blockHash}` | GOB-encoded Block |
| `"lh"` | Last block hash |
| `"bh"` | Current chain height (8-byte big-endian uint64) |
| `"va"` | Validator list (GOB-encoded `[][]byte`) |

### Consensus: Proof of Authority

Blocks are not mined. Instead, a registered validator signs each block.

**Validator selection** is deterministic round-robin by block height:

```go
validator = validators[height % len(validators)]
```

**Block signing** uses ECDSA P256 over `SHA256(PrevHash || TxRoot || Timestamp || Height)`. The signature is stored as a fixed 64-byte value: 32 bytes for `r`, 32 bytes for `s`, both zero-padded big-endian.

**Validation** (`ValidateBlock`) checks that:
1. The block's `Validator` field matches the expected round-robin selection
2. The ECDSA signature is valid for the block hash

Validators are added to the chain when an admin approves an authority request. The primary node broadcasts new validator keys to all peers.

### UTXO Transaction Model

Coin ownership is tracked by unspent transaction outputs, not account balances.

```go
type Transaction struct {
    Id      []byte
    Inputs  []TxInput
    Outputs []TxOutput
}

type TxInput struct {
    Id        []byte    // Hash of the referenced transaction
    Out       int       // Index of the output being spent
    Signature []byte    // 64-byte ECDSA signature
    PubKey    []byte    // 64-byte sender public key
}

type TxOutput struct {
    Value      int      // Number of SHELLs
    PubKeyHash []byte   // 20-byte recipient public key hash
}
```

A `TxOutput` is considered spent when it appears as a `TxInput.Id` + `TxInput.Out` reference elsewhere in the chain.

**Transaction types:**

- **Coinbase** — Block reward (100 SHELLs). Has an empty input ID and `Out = -1`.
- **Regular** — Spends existing UTXOs. Finds enough spendable outputs via `FindSpendableOutputs`, creates a change output back to the sender if the total exceeds the amount.

**Signing and verification** follow a trimmed-copy pattern: for each input, the transaction is copied with all signatures cleared, the input's PubKey is replaced with the referenced output's PubKeyHash, and that modified copy is hashed and signed with ECDSA P256.

### Wallet and Address Format

Key pairs use the P256 elliptic curve. The address derivation:

```
Public Key (64 bytes, X || Y)
  -> SHA256
  -> SHA3-256 (20-byte output)
  = Public Key Hash

Version (0x00) + PubKeyHash + Checksum (first 4 bytes of double-SHA256)
  -> Base58 Encode
  = Address string
```

In CLI mode, wallets are persisted as a GOB-encoded map in `$DB_PATH/wallets.data`. In API mode, private key bytes are stored in the API database under `wallet_privkey:{userID}`.

---

## Store Layer

The API database is a separate BadgerDB instance at `$DB_PATH/api`. All values are JSON-marshaled. The store provides typed CRUD methods built on generic helpers:

```go
set(key, value)       // JSON marshal and store
get(key, &target)     // Retrieve and unmarshal
del(key)              // Delete
iterPrefix(prefix, fn) // Scan all keys with a given prefix
```

### Key Prefixes

| Prefix | Entity |
|--------|--------|
| `user:` | User records |
| `idx:user:email:` | Email -> user ID index |
| `idx:user:username:` | Username -> user ID index |
| `authority:` | Authority records |
| `node:` | Node records |
| `nodereq:` | Node join requests |
| `ticket:` | Support tickets |
| `invitation:` | Invitation codes |
| `value:` | Price history records |
| `value_latest:` | Latest price per authority |
| `refresh:` | Refresh tokens |
| `wallet_privkey:` | User wallet private keys (raw D bytes) |
| `authority_privkey:` | Authority validator private keys (raw D bytes) |
| `verifyconfig:` | Authority verification configurations |
| `verifysub:` | Verification submissions (by UUID) |
| `idx:verifysub:user:` | User+Authority → latest submission index |

---

## Data Models

### User

```go
type User struct {
    ID            string     // UUID
    Username      string
    Email         string
    PasswordHash  string     // bcrypt, cost 12
    Role          string     // "superadmin" | "admin" | "user"
    AuthorityID   string     // Empty if unaffiliated
    AuthorityRole string     // "owner" | "member"
    WalletAddress string     // Blockchain address (empty until created)
    Active        bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Authority

Represents an organization that operates a node and has a validator key on the chain.

```go
type Authority struct {
    ID              string
    Name            string
    Description     string
    OwnerID         string      // User who created it
    Status          string      // "pending" | "active" | "suspended" | "rejected"
    ValidatorPubKey string      // Hex-encoded 64-byte ECDSA public key
    BasePrice       float64     // Credits per SHELL at genesis
    SensitivityK    float64     // Velocity multiplier
    CreatedAt       time.Time
    ApprovedAt      *time.Time
    ApprovedBy      string
    RejectionReason string
}
```

### Node

```go
type Node struct {
    ID              string
    AuthorityID     string
    AuthorityName   string
    NodeURL         string      // Public URL
    ValidatorPubKey string
    Status          string      // "pending" | "active" | "offline" | "suspended" | "rejected"
    IsPrimary       bool
    BlockHeight     uint64
    Version         string
    LastSeenAt      *time.Time
    RegisteredAt    time.Time
    ApprovedAt      *time.Time
    ApprovedBy      string
    RejectionReason string
}
```

### Other Models

- **Ticket** — Support ticket with replies, statuses (`open`, `in_progress`, `resolved`, `closed`), scoped to an authority.
- **Invitation** — One-time-use code for joining an authority, with expiration.
- **ValueRecord** — Snapshot of an authority's token price at a given block height: price, transaction volume, circulating supply, and velocity.
- **VerificationConfig** — Per-authority configuration for the modular verification system. Contains `Method` (currently `"manual"`) and `Fields` (array of `FieldDefinition` with name, label, input type, and validation rules). Extensible for future 3rd-party API and OAuth-based methods.
- **VerificationSubmission** — A user's verification attempt. Contains submitted field values (`map[string]string`), status (`pending`/`approved`/`rejected`), reviewer remarks, and timestamps.

---

## Service Layer

### Auth Service

Handles passwords (bcrypt cost 12), JWT tokens (HS256), and refresh tokens.

- **Access tokens** — 15-minute JWTs containing `uid`, `role`, `aid` (authority ID), and `ar` (authority role).
- **Refresh tokens** — Random hex strings stored in the API database with a 7-day TTL. Deleted on logout.
- **Bootstrap** — On first start, creates the superadmin account if none exists.

### Chain Service

Thread-safe wrapper around the blockchain. All chain operations acquire a mutex.

Key operations:
- `CreateAndSubmit(from, to, amount, privKey, pubKey)` — Builds transaction, signs it, creates and signs a block, stores it, triggers the `onBlock` callback for P2P broadcast.
- `AddExternalBlock(block)` — Accepts a peer-received block after validating PoA signature, prevHash linkage, and height sequence.
- `GetBalance(address)` — Sums all unspent outputs for an address.
- `GetBlocks(limit)` / `GetBlocksFromHeight(from, limit)` — Block retrieval for the UI and P2P sync.

The `onBlock` callback is wired to `NodeService.OnBlockMined` during server startup, which triggers broadcast to all active peers.

### Value Service

Implements a Velocity of Money pricing model:

```
velocity = transaction_volume / circulating_supply
price    = base_price * (1 + sensitivity_k * velocity)
```

- **transaction_volume** — Total SHELLs moved in the last 50 blocks by authority members.
- **circulating_supply** — Sum of all UTXO balances held by authority members.

Recalculated after each block that involves authority members.

### Node Service

Orchestrates the node lifecycle:

**Primary bootstrap:**
1. Create a self-referencing Node record (status=active, is_primary=true)
2. Start the health checker

**Secondary bootstrap:**
1. Check if already approved locally
2. If not, log a waiting message and return (approval happens via the primary's superadmin UI)
3. Once approved: full-sync the chain from primary, start the health checker

**Ongoing:**
- `OnBlockMined(blockHash)` — Broadcasts the new block to all active peers. Also broadcasts any new validator keys if an authority was just approved.

---

## API Layer

### Server Bootstrap

`api.Start(cfg)` runs through these steps:

1. Open the API database (`$DB_PATH/api`)
2. Create services: auth, chain, value, node
3. Wire the block-broadcast callback: `chainSvc.SetOnBlock(nodeSvc.OnBlockMined)`
4. Bootstrap the superadmin account
5. Bootstrap the node identity (register self or wait for approval)
6. Build the chi router with all routes and middleware
7. Start listening on `$PORT`

### Response Envelope

Every response uses a standard JSON envelope:

```json
{ "success": true,  "data": { ... } }
{ "success": false, "error": "message" }
```

### Middleware

- **JWT** — Extracts `Authorization: Bearer {token}`, validates the JWT, injects claims into the request context. On 401, attempts an automatic refresh.
- **RequireRole(roles...)** — Checks the JWT claim's role against allowed values.
- **RequireAuthorityOwner** — Checks `authority_role == "owner"` in JWT claims.
- **RequireNodeSecret** — Validates the `X-Node-Secret` header for P2P routes.

### Route Map

**Public (no auth):**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/blocks` | List blocks |
| GET | `/api/v1/blocks/{hash}` | Get block by hash |
| GET | `/api/v1/value` | Current prices |
| POST | `/api/v1/nodes/register` | Node join request (from secondary) |

**Auth:**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Create account |
| POST | `/api/v1/auth/login` | Get token pair |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| GET | `/api/v1/auth/me` | Current user (JWT) |

**User (JWT required):**

| Method | Path | Description |
|--------|------|-------------|
| GET/PUT | `/api/v1/user/me` | Profile |
| GET/POST | `/api/v1/user/wallet` | Wallet management |
| GET/POST | `/api/v1/user/transactions` | Transactions |
| GET/POST | `/api/v1/user/authority` | Create authority request |
| POST | `/api/v1/user/authority/join` | Join via invitation code |
| GET/POST | `/api/v1/user/tickets` | Support tickets |
| GET | `/api/v1/user/tickets/{id}` | Ticket detail |
| POST | `/api/v1/user/tickets/{id}/reply` | Reply to ticket |
| GET | `/api/v1/user/verification` | Verification status + config |
| POST | `/api/v1/user/verification` | Submit verification form |

**Authority Owner (JWT + owner role):**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/authority/members` | List members |
| DELETE | `/api/v1/authority/members/{userID}` | Remove member |
| GET/POST | `/api/v1/authority/invitations` | Manage invitations |
| DELETE | `/api/v1/authority/invitations/{code}` | Revoke invitation |
| GET | `/api/v1/authority/transactions` | Authority transactions |
| GET | `/api/v1/authority/value` | Token price |
| GET | `/api/v1/authority/stats` | Authority statistics |
| GET/PUT | `/api/v1/authority/verification/config` | Verification field config |
| GET | `/api/v1/authority/verification/submissions` | List member submissions |
| GET | `/api/v1/authority/verification/submissions/{id}` | Submission detail |
| POST | `/api/v1/authority/verification/submissions/{id}/review` | Approve/reject submission |

**Admin (JWT + admin/superadmin):**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/authority-requests` | Pending requests |
| POST | `/api/v1/admin/authority-requests/{id}/approve` | Approve authority |
| POST | `/api/v1/admin/authority-requests/{id}/reject` | Reject authority |
| GET | `/api/v1/admin/tickets` | All tickets |
| GET/PUT | `/api/v1/admin/tickets/{id}` | Manage ticket |
| POST | `/api/v1/admin/tickets/{id}/reply` | Admin reply |
| GET | `/api/v1/admin/authorities` | All authorities |
| GET | `/api/v1/admin/users` | All users |
| GET | `/api/v1/admin/stats` | Platform statistics |
| GET | `/api/v1/admin/nodes` | Network nodes |
| GET | `/api/v1/admin/nodes/{id}` | Node detail |

**SuperAdmin (JWT + superadmin):**

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/superadmin/admins` | Admin CRUD |
| DELETE | `/api/v1/superadmin/admins/{id}` | Demote admin |
| POST | `/api/v1/superadmin/authorities/{id}/suspend` | Suspend authority |
| POST | `/api/v1/superadmin/authorities/{id}/reinstate` | Reinstate authority |
| GET | `/api/v1/superadmin/stats` | Global statistics |
| GET | `/api/v1/superadmin/node-requests` | Pending node joins |
| POST | `/api/v1/superadmin/node-requests/{id}/approve` | Approve node |
| POST | `/api/v1/superadmin/node-requests/{id}/reject` | Reject node |
| POST | `/api/v1/superadmin/nodes/{id}/suspend` | Suspend node |
| POST | `/api/v1/superadmin/nodes/{id}/reinstate` | Reinstate node |

**P2P (X-Node-Secret header):**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/p2p/v1/ping` | Health check between peers |
| GET | `/p2p/v1/peers` | Active peer list |
| GET | `/p2p/v1/chain/sync` | Download blocks (batch) |
| POST | `/p2p/v1/blocks` | Receive broadcast block |
| GET | `/p2p/v1/validators` | Get validator list |
| POST | `/p2p/v1/validators` | Receive new validator |

---

## P2P Network

### Peer Client

Each node maintains HTTP clients for communicating with peers. All requests include the shared `X-Node-Secret` header and have a 10-second timeout.

### Chain Sync

**Full sync** — Used when a newly approved node joins the network. Downloads the entire chain from a peer in batches of 100 blocks, validating each block's PoA signature and chain linkage before storing.

**Incremental sync** — Fetches blocks after the local chain tip. Used for catch-up after brief disconnections.

### Block Broadcasting

When a node creates a new block, the `onBlock` callback fires:

1. Retrieve the list of active peers (excluding self)
2. Spawn a goroutine per peer to POST the block
3. Wait for all to complete; log errors but don't fail

The same fan-out pattern is used to broadcast new validator keys.

### Health Checker

A background goroutine runs on each node:

- Every 30 seconds, pings all known active peers via `POST /p2p/v1/ping`
- On success: updates `LastSeenAt`, `BlockHeight`, and `Status`
- On failure: marks the node as `offline`
- Updates are persisted to the local store

---

## Frontend

The Next.js app lives in `example/` and communicates with the Go API via a typed client.

### Stack

- **Next.js 16** with App Router
- **React 19** with Server Components
- **Tailwind CSS 3** for styling
- **shadcn/ui** for component primitives
- **React Query** for server state (polling, caching, auto-refresh)
- **Zustand** for client state (auth session, persisted to localStorage)

### API Client

`src/lib/api.ts` provides a `request<T>()` helper that:

1. Adds `Authorization: Bearer {token}` for authenticated requests
2. On 401, attempts a refresh using the stored refresh token
3. If refresh succeeds, retries the original request transparently
4. If refresh fails, clears tokens and throws

### Pages

| Route | Role | Description |
|-------|------|-------------|
| `/` | Public | Landing page with live block explorer |
| `/login` | Public | Login form |
| `/register` | Public | Registration form |
| `/dashboard` | User | Wallet balance, recent transactions |
| `/dashboard/wallet` | User | Create wallet, view address and balance |
| `/dashboard/transactions` | User | Transaction history and send form |
| `/dashboard/tickets` | User | Support tickets |
| `/dashboard/authority` | Owner | Members, invitations, value, stats, verification setup |
| `/dashboard/verification` | User | Dynamic verification form and status |
| `/admin` | Admin | Dashboard with network status card |
| `/admin/requests` | Admin | Authority approval queue |
| `/admin/users` | Admin | User list |
| `/admin/tickets` | Admin | Ticket management |
| `/admin/network` | Admin | Node list with status, height, last-seen |
| `/admin/network/[id]` | Admin | Node detail with suspend/reinstate actions |
| `/superadmin` | SuperAdmin | Admin management, node request approval |

---

## Key Data Flows

### Transaction

```
User -> POST /user/transactions { to_address, amount }
  -> Load user's wallet private key from store
  -> ChainService.CreateAndSubmit():
       Find spendable UTXOs
       Build transaction (inputs referencing UTXOs, outputs to recipient + change)
       Sign each input with sender's private key
       Create block with coinbase + transaction
       Sign block with authority validator key
       Store block in BadgerDB
       Fire onBlock callback
  -> NodeService.OnBlockMined():
       Broadcast block to all active peers
  -> ValueService.RecalculateForAuthority():
       Recompute token price using Velocity of Money formula
  -> Return transaction result to user
```

### Authority Approval

```
User -> POST /user/authority { name, description }
  -> Authority created with status=pending

Admin -> POST /admin/authority-requests/{id}/approve { base_price, sensitivity_k }
  -> Generate ECDSA P256 key pair
  -> Register public key as blockchain validator
  -> Store private key in API database
  -> Set authority status=active
  -> Broadcast new validator key to all peers
```

### Node Join

```
New node starts with NODE_URL + PRIMARY_NODE_URL configured
  -> POST primary/api/v1/nodes/register { node_url, authority_name, ... }
  -> Primary stores NodeJoinRequest (status=pending)

Primary superadmin -> POST /superadmin/node-requests/{id}/approve
  -> Create Node record (status=active)
  -> Return peer list

New node:
  -> FullSync from primary (batches of 100 blocks)
  -> Fetch peer list
  -> Start health checker
  -> Begin normal operation
```

### Modular Verification

```
Authority owner -> PUT /authority/verification/config { method: "manual", fields: [...] }
  -> Validate field definitions (unique names, valid types)
  -> Store VerificationConfig for authority
  -> Members can now see the verification form

Member -> GET /user/verification
  -> If authority has no config: { config_status: "not_configured" }
  -> If configured: return config fields + latest submission (if any)

Member -> POST /user/verification { field_values: { ... } }
  -> Validate field values against config (required, pattern, length)
  -> Check no pending/approved submission exists
  -> Create VerificationSubmission (status=pending)

Owner -> GET /authority/verification/submissions?status=pending
  -> List pending submissions with user info

Owner -> POST /authority/verification/submissions/{id}/review { action: "approve" }
  -> Set submission status=approved
  -> User can now create a wallet

Owner -> POST /authority/verification/submissions/{id}/review { action: "reject", remarks: "..." }
  -> Set submission status=rejected with remarks
  -> User sees remarks and can resubmit

Wallet gate:
  POST /user/wallet
  -> If user is affiliated: check IsUserVerified(userID, authorityID)
  -> If not verified: return 403 "verification required"
  -> If verified: proceed with wallet creation
```

---

## Constants

| Constant | Value | Location | Purpose |
|----------|-------|----------|---------|
| Bcrypt cost | 12 | auth_service.go | Password hashing |
| Sync batch size | 100 | node/sync.go | Blocks per P2P request |
| P2P timeout | 10s | node/peer.go | HTTP client timeout |
| Health interval | 30s | node/healthchecker.go | Peer ping frequency |
| Velocity window | 50 blocks | value_service.go | Transaction volume lookback |
| Coinbase reward | 100 SHELL | blockchain/transaction.go | Block reward |
| Checksum length | 4 bytes | wallet/address.go | Address checksum |
| Address version | 0x00 | wallet/address.go | Version byte |

---

## Security

- **Cryptography** — ECDSA P256 for all signing (blocks, transactions). Fixed 64-byte signature encoding prevents variable-length parsing bugs.
- **Passwords** — bcrypt with cost 12.
- **JWT** — HS256 with configurable secret. 15-minute access tokens limit the blast radius of a leaked token.
- **P2P authentication** — Shared secret in `X-Node-Secret` header. All nodes in the network must have the same value.
- **Private keys at rest** — Stored as raw D bytes in BadgerDB, unencrypted. In production, use disk encryption or a KMS.
- **Chain validation** — Every block received via P2P is independently validated: PoA signature, prevHash linkage, height sequence.
- **No rate limiting** — Must be handled at the reverse proxy layer (nginx, Caddy, etc.).
- **TLS** — Not terminated by the Go server; expected to be handled by a reverse proxy.

---

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for systemd, nginx, and Docker setup.

**Docker Compose** defines two services:
- `api` — Go binary with health check at `/api/v1/health`, volume-mounted database
- `ui` — Next.js frontend (optional profile), configured with `NEXT_PUBLIC_API_URL`

**Primary node** requires `IS_PRIMARY=true` and `SUPERADMIN_PASSWORD` set.
**Secondary nodes** require `PRIMARY_NODE_URL` pointing to the primary and the same `NODE_SECRET`.
