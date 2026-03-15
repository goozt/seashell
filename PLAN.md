# SeaShell Decentralized Network — Implementation Plan

## Overview

Transform SeaShell from a single-process centralized system into a distributed network where each Authority runs its own node. The Primary node bootstraps the network; all subsequent nodes register through the Primary and sync the blockchain across peers.

---

## 1. Architecture

```
                        ┌─────────────────────────────────┐
                        │        PRIMARY NODE              │
                        │  (Authority 0 / First Node)      │
                        │                                  │
                        │  ┌─────────┐  ┌──────────────┐  │
                        │  │ API/UI  │  │  Blockchain   │  │
                        │  │ Server  │  │  (BadgerDB)   │  │
                        │  └────┬────┘  └──────────────┘  │
                        │       │  Node Registry           │
                        │       │  (source of truth for    │
                        │       │   network membership)    │
                        └───────┼─────────────────────────┘
                    ┌───────────┴────────────┐
                    │   P2P HTTP Layer        │
                    │  (block broadcast +     │
                    │   peer discovery)       │
          ┌─────────┴──────────┐   ┌─────────┴──────────┐
          │    NODE A           │   │    NODE B           │
          │  (Authority 1)      │   │  (Authority 2)      │
          │                     │   │                     │
          │  ┌───────────────┐  │   │  ┌───────────────┐  │
          │  │ API/UI Server │  │   │  │ API/UI Server │  │
          │  │ (local users  │  │   │  │ (local users  │  │
          │  │  + wallets)   │  │   │  │  + wallets)   │  │
          │  └───────────────┘  │   │  └───────────────┘  │
          │  ┌───────────────┐  │   │  ┌───────────────┐  │
          │  │  Blockchain   │  │   │  │  Blockchain   │  │
          │  │  (synced copy)│  │   │  │  (synced copy)│  │
          │  └───────────────┘  │   │  └───────────────┘  │
          └─────────────────────┘   └─────────────────────┘
```

### Key Principles
- **Blockchain is replicated** across all active nodes (each node has an identical copy)
- **User/wallet data is local** to each node (users register on their authority's node)
- **Node registry lives on the primary node** — other nodes fetch it from primary
- **P2P is HTTP-based REST** (no libp2p/gossip — simple and auditable)
- **Block broadcast is push** — block creator pushes to all known peers after mining
- **Initial sync is pull** — new approved node pulls full chain from primary

---

## 2. Node Lifecycle

### 2.1 Primary Node Startup
```
IS_PRIMARY=true, PRIMARY_NODE_URL not set

1. Create blockchain (genesis block)
2. Bootstrap superadmin account
3. Self-register as Node{id:"primary", status:active, is_primary:true}
4. Start background health-checker (pings peers every 30s)
5. Start API server
```

### 2.2 New Authority Node Registration
```
New authority installs SeaShell on their server
↓
Configure: PRIMARY_NODE_URL=https://primary.seashell.io
            NODE_URL=https://authority-n.example.com
↓
Node starts → detects not yet approved → sends NodeJoinRequest to primary:
  POST primary/api/v1/nodes/register
  { node_url, authority_name, admin_email, validator_pubkey, node_public_key }
↓
Primary stores as pending NodeJoinRequest
↓
Primary superadmin reviews in UI → Approves
↓
Primary:
  a. Adds node's ValidatorPubKey to blockchain DB ("va" key) on primary
  b. Broadcasts new validator key to all active peers (they update their "va")
  c. Updates NodeJoinRequest → Node record (status=active)
  d. Responds to waiting new node with: { node_id, peer_list }
↓
New node:
  a. Downloads full blockchain from primary (GET /p2p/v1/chain/sync)
  b. Fetches peer list (GET /p2p/v1/peers)
  c. Starts normal operation
```

### 2.3 Block Creation & Broadcast
```
User sends transaction on Node A
↓
Node A builds + signs transaction
↓
Node A selects validator (round-robin from "va" list, determined by block height)
↓
If selected validator is this node: sign and mine block locally
If selected validator is another node: forward signing request to that node via P2P
↓
Block created → Node A broadcasts to all known active peers:
  POST peer/p2p/v1/blocks  { block_data }
↓
Each peer validates block signature + PoA round, adds to chain
```

### 2.4 Health Monitoring
```
Each node maintains a background goroutine:
  Every 30s: POST /p2p/v1/ping to all known peers
  Peer responds with: { node_id, block_height, status }
  If ping fails 3 times: mark peer as offline in local cache
  Update primary node registry: POST primary/p2p/v1/node-status
```

---

## 3. Go Backend Changes

### 3.1 New Package: `/node/`

#### `/node/node.go`
Core node identity and config.
```go
type NodeConfig struct {
    NodeURL          string  // this node's public URL
    PrimaryNodeURL   string  // empty if this is primary
    IsPrimary        bool
    SharedSecret     string  // for authenticating P2P requests
}
```

#### `/node/registry.go`
Node registry management (full CRUD, runs on primary; other nodes cache).
```go
// CreateNodeRequest stores a pending join request
func (r *Registry) CreateNodeRequest(req *model.NodeJoinRequest) error

// ApproveNode promotes request → active node, returns peer list
func (r *Registry) ApproveNode(requestID string, approvedBy string) (*model.Node, []model.Node, error)

// RejectNode marks request as rejected
func (r *Registry) RejectNode(requestID string, reason string) error

// GetActiveNodes returns all active nodes
func (r *Registry) GetActiveNodes() ([]model.Node, error)

// GetAllNodes returns all nodes with all statuses
func (r *Registry) GetAllNodes() ([]model.Node, error)

// UpdateNodeStatus updates status + last_seen + block_height from ping
func (r *Registry) UpdateNodeStatus(nodeID string, height uint64, status string) error
```

#### `/node/peer.go`
P2P HTTP client for communicating with peer nodes.
```go
type PeerClient struct {
    baseURL      string
    sharedSecret string
    httpClient   *http.Client
}

func (c *PeerClient) Ping(height uint64) (*PingResponse, error)
func (c *PeerClient) GetChain(fromHeight uint64) ([]blockchain.Block, error)
func (c *PeerClient) BroadcastBlock(block *blockchain.Block) error
func (c *PeerClient) GetPeers() ([]model.Node, error)
func (c *PeerClient) NotifyNewValidator(pubKey []byte) error
```

#### `/node/sync.go`
Blockchain synchronization logic.
```go
// FullSync downloads all blocks from a peer starting from height 0
func FullSync(peer *PeerClient, chain *blockchain.BlockChain) error

// IncrementalSync fetches blocks from peer starting after localHeight
func IncrementalSync(peer *PeerClient, chain *blockchain.BlockChain, localHeight uint64) error
```

#### `/node/broadcast.go`
Broadcast new blocks to all active peers.
```go
// BroadcastBlock sends the block to all active peers concurrently
func BroadcastBlock(peers []model.Node, block *blockchain.Block, secret string) []error

// BroadcastNewValidator notifies all active peers of new validator key
func BroadcastNewValidator(peers []model.Node, pubKey []byte, secret string) []error
```

#### `/node/healthchecker.go`
Background goroutine for peer health monitoring.
```go
// Start launches background goroutine, pings every interval
func StartHealthChecker(registry *Registry, secret string, interval time.Duration)
```

---

### 3.2 New Model: `/model/node.go`
```go
const (
    NodeStatusPending   = "pending"
    NodeStatusActive    = "active"
    NodeStatusOffline   = "offline"
    NodeStatusSuspended = "suspended"
    NodeStatusRejected  = "rejected"
)

type Node struct {
    ID              string     `json:"id"`
    AuthorityID     string     `json:"authority_id"`
    AuthorityName   string     `json:"authority_name"`
    NodeURL         string     `json:"node_url"`
    ValidatorPubKey string     `json:"validator_pubkey"`
    Status          string     `json:"status"`
    IsPrimary       bool       `json:"is_primary"`
    BlockHeight     uint64     `json:"block_height"`
    Version         string     `json:"version"`
    LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
    RegisteredAt    time.Time  `json:"registered_at"`
    ApprovedAt      *time.Time `json:"approved_at,omitempty"`
    ApprovedBy      string     `json:"approved_by,omitempty"`
    RejectionReason string     `json:"rejection_reason,omitempty"`
}

type NodeJoinRequest struct {
    ID              string    `json:"id"`
    NodeURL         string    `json:"node_url"`
    AuthorityName   string    `json:"authority_name"`
    AdminEmail      string    `json:"admin_email"`
    ValidatorPubKey string    `json:"validator_pubkey"`
    Status          string    `json:"status"`  // pending|approved|rejected
    CreatedAt       time.Time `json:"created_at"`
    ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
    ReviewedBy      string    `json:"reviewed_by,omitempty"`
    RejectionReason string    `json:"rejection_reason,omitempty"`
}

// NodeRegisterRequest is the payload for POST /api/v1/nodes/register
type NodeRegisterRequest struct {
    NodeURL         string `json:"node_url"`
    AuthorityName   string `json:"authority_name"`
    AdminEmail      string `json:"admin_email"`
    ValidatorPubKey string `json:"validator_pubkey"`
}

// NodeApproveResponse is returned after superadmin approves a node
type NodeApproveResponse struct {
    NodeID    string  `json:"node_id"`
    Peers     []Node  `json:"peers"`  // existing active nodes to connect to
}
```

---

### 3.3 New Handler: `/api/handlers/node.go`

#### P2P Endpoints (authenticated via shared secret in `X-Node-Secret` header)
```
POST /p2p/v1/ping
  Request:  { node_id, block_height }
  Response: { node_id, block_height, status:"ok" }

GET  /p2p/v1/chain/sync?from=0&limit=100
  Response: { blocks: [...], total: N }
  (streams blocks for initial sync)

POST /p2p/v1/blocks
  Request:  { block: {...} }
  Response: { accepted: true } or { accepted: false, reason: "..." }

GET  /p2p/v1/peers
  Response: { peers: [{id, node_url, status, block_height}...] }

POST /p2p/v1/validators
  Request:  { validator_pubkey: "hex" }
  Response: { accepted: true }
```

#### Public Registration (new node calls this on primary)
```
POST /api/v1/nodes/register
  Request:  NodeRegisterRequest
  Response: { message: "join request submitted, awaiting approval" }
```

#### Admin Endpoints (JWT + admin/superadmin role)
```
GET /api/v1/admin/nodes
  Response: { nodes: [Node...] }
  (on non-primary nodes, fetches from primary node's registry)

GET /api/v1/admin/nodes/{id}
  Response: { node: Node }
```

#### SuperAdmin Endpoints (JWT + superadmin role, primary node only)
```
GET  /api/v1/superadmin/node-requests
  Response: { requests: [NodeJoinRequest...] }

POST /api/v1/superadmin/node-requests/{id}/approve
  Response: { node: Node }

POST /api/v1/superadmin/node-requests/{id}/reject
  Request:  { reason: string }
  Response: { message: "rejected" }

POST /api/v1/superadmin/nodes/{id}/suspend
  Response: { message: "suspended" }

POST /api/v1/superadmin/nodes/{id}/reinstate
  Response: { message: "reinstated" }
```

---

### 3.4 Config Changes: `/config/config.go`
Add new fields:
```go
type Config struct {
    // ... existing fields ...
    NodeURL        string  // this node's public URL (e.g. https://mynode.example.com)
    PrimaryNodeURL string  // primary node URL; empty if this IS primary
    IsPrimary      bool    // true for the first/bootstrap node
    NodeSecret     string  // shared secret for P2P authentication
}
```

New environment variables:
```
NODE_URL=https://mynode.example.com     # required
PRIMARY_NODE_URL=https://primary.io     # omit if primary
IS_PRIMARY=false                        # set true for primary node
NODE_SECRET=change-me-p2p-secret        # shared across all nodes
```

---

### 3.5 New Middleware: `/api/middleware/node.go`
```go
// RequireNodeSecret validates the X-Node-Secret header for P2P routes
func RequireNodeSecret(secret string) func(http.Handler) http.Handler
```

---

### 3.6 Service Changes: `/service/node_service.go` (new)
```go
type NodeService struct {
    db       *store.DB
    registry *node.Registry
    cfg      *config.Config
}

// RegisterSelf registers this node on the primary node at startup (non-primary nodes)
func (s *NodeService) RegisterSelf() error

// SyncFromPrimary downloads full blockchain from primary node
func (s *NodeService) SyncFromPrimary() error

// BroadcastBlock sends a newly mined block to all active peers
func (s *NodeService) BroadcastBlock(block *blockchain.Block) error

// GetNetworkNodes returns nodes: from local registry (if primary) or fetched from primary
func (s *NodeService) GetNetworkNodes() ([]model.Node, error)
```

---

### 3.7 Server Bootstrap Changes: `/api/server.go`
On startup:
1. If `IsPrimary`: self-register, start health checker
2. If not `IsPrimary`:
   - Poll primary registration endpoint until approved (with exponential backoff)
   - Once approved: run `FullSync` to download blockchain
   - Fetch peer list and start health checker
3. Add P2P routes and node routes to router
4. Pass `NodeService` into `ChainService` so new blocks are broadcast after mining

---

### 3.8 Chain Service Changes: `/service/chain_service.go`
After successfully adding a block:
```go
// After AddBlock(...) succeeds:
go nodeSvc.BroadcastBlock(block)
```

---

### 3.9 Store Keys (additions)
```
"node:{nodeID}"         → Node JSON
"nodereq:{requestID}"   → NodeJoinRequest JSON
"node:index"            → []string of node IDs (sorted)
"nodereq:index"         → []string of request IDs
```

---

## 4. Frontend Changes (Next.js)

### 4.1 New Types in `/example/src/types/api.ts`
```typescript
export type NodeStatus = 'pending' | 'active' | 'offline' | 'suspended' | 'rejected'

export interface Node {
  id: string
  authority_id: string
  authority_name: string
  node_url: string
  validator_pubkey: string
  status: NodeStatus
  is_primary: boolean
  block_height: number
  version: string
  last_seen_at?: string
  registered_at: string
  approved_at?: string
  approved_by?: string
  rejection_reason?: string
}

export interface NodeJoinRequest {
  id: string
  node_url: string
  authority_name: string
  admin_email: string
  validator_pubkey: string
  status: 'pending' | 'approved' | 'rejected'
  created_at: string
  reviewed_at?: string
  reviewed_by?: string
  rejection_reason?: string
}
```

---

### 4.2 New API Client Methods in `/example/src/lib/api.ts`

```typescript
// Admin API additions
adminApi.getNodes(): Promise<{ nodes: Node[] }>
adminApi.getNode(id: string): Promise<{ node: Node }>

// SuperAdmin API additions
superAdminApi.getNodeRequests(): Promise<{ requests: NodeJoinRequest[] }>
superAdminApi.approveNodeRequest(id: string): Promise<{ node: Node }>
superAdminApi.rejectNodeRequest(id: string, reason: string): Promise<void>
superAdminApi.suspendNode(id: string): Promise<void>
superAdminApi.reinstateNode(id: string): Promise<void>
```

---

### 4.3 New Admin Page: `/example/src/app/admin/network/page.tsx`

**Network Overview Page** — accessible to all admins.

Layout:
```
┌────────────────────────────────────────────────────────────┐
│ Network Overview                                            │
│                                                             │
│ ┌──────────┐  ┌────────────┐  ┌──────────┐  ┌──────────┐  │
│ │  Total   │  │   Active   │  │ Offline  │  │  Pending │  │
│ │  Nodes   │  │   Nodes    │  │  Nodes   │  │  Nodes   │  │
│ │    5     │  │     4      │  │    1     │  │    0     │  │
│ └──────────┘  └────────────┘  └──────────┘  └──────────┘  │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Node               │ Status  │ Height │ URL    │ Seen   │ │
│ ├────────────────────┼─────────┼────────┼────────┼────────┤ │
│ │ ★ Primary Node     │ ●active │ 1024   │ ...    │ now    │ │
│ │ Authority Alpha    │ ●active │ 1024   │ ...    │ 12s    │ │
│ │ Authority Beta     │ ●active │ 1023   │ ...    │ 45s    │ │
│ │ Authority Gamma    │ ●offline│  987   │ ...    │ 5m     │ │
│ │ Authority Delta    │ ●active │ 1024   │ ...    │ 8s     │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  Auto-refreshes every 30 seconds                           │
└────────────────────────────────────────────────────────────┘
```

Features:
- Status badge: green (active), red (offline), yellow (pending), grey (suspended)
- Block height column shows chain sync lag
- "Last Seen" relative time (e.g. "12s ago")
- Node URL links to the node's public endpoint
- Primary node marked with ★
- Auto-refresh every 30s via React Query `refetchInterval`
- Click row → opens node detail modal or navigates to detail page

---

### 4.4 New Admin Page: `/example/src/app/admin/network/[id]/page.tsx`

**Node Detail Page** — detailed view of a single node.

Layout:
```
┌────────────────────────────────────────────────────────────┐
│ ← Back to Network                                           │
│                                                             │
│ Authority Alpha                              ●active        │
│ Node ID: abc-123...                                         │
│                                                             │
│ ┌─────────────────────────┐  ┌──────────────────────────┐  │
│ │ Node URL                │  │ Block Height             │  │
│ │ https://alpha.example   │  │ 1,024 blocks             │  │
│ └─────────────────────────┘  └──────────────────────────┘  │
│ ┌─────────────────────────┐  ┌──────────────────────────┐  │
│ │ Last Seen               │  │ Joined Network           │  │
│ │ 12 seconds ago          │  │ Jan 5, 2026              │  │
│ └─────────────────────────┘  └──────────────────────────┘  │
│                                                             │
│ Validator Public Key                                        │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ 04a3f1... (truncated)                         [Copy]    │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  [Suspend Node]  (superadmin only, non-primary nodes)      │
└────────────────────────────────────────────────────────────┘
```

---

### 4.5 Update Admin Dashboard: `/example/src/app/admin/page.tsx`

Add a **Network Status** summary card to the existing stats grid:

```
┌────────────────────────────────────────────────────────────┐
│ Network Status                                              │
│ 4/5 nodes online          [View Network →]                  │
│ Latest block: #1,024                                        │
│ ● ● ● ● ○  (node health dots)                              │
└────────────────────────────────────────────────────────────┘
```

---

### 4.6 Update Admin Navigation: `/example/src/components/layout/header.tsx` (or nav)

Add "Network" link to admin navigation menu.

---

### 4.7 Update SuperAdmin Page: `/example/src/app/superadmin/page.tsx`

Add a **"Node Requests"** tab alongside existing "Admins" and "Authorities" tabs.

Node Requests tab layout:
```
┌────────────────────────────────────────────────────────────┐
│ Tabs: [Admins] [Authorities] [Node Requests]               │
│                                                             │
│ Node Requests                                               │
│                                                             │
│ ● 2 pending requests                                        │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Authority Name │ URL           │ Email    │ Requested   │ │
│ ├────────────────┼───────────────┼──────────┼─────────────┤ │
│ │ Epsilon Co     │ eps.example   │ a@b.com  │ 2h ago      │ │
│ │  [Approve]  [Reject]                                    │ │
│ │ Zeta Corp      │ zeta.example  │ c@d.com  │ 3h ago      │ │
│ │  [Approve]  [Reject]                                    │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Note: Only visible on primary node                         │
└────────────────────────────────────────────────────────────┘
```

- Approve button: calls `superAdminApi.approveNodeRequest(id)`
- Reject button: opens dialog asking for rejection reason
- Only shows on primary node (detect via `node.is_primary` flag from API)
- Refreshes every 30s

---

### 4.8 New Dashboard Page for Authority Owners (optional enhancement)

Update `/example/src/app/dashboard/authority/page.tsx` to show:
- Node status of their authority's node (online/offline/pending)
- Block height sync status
- "Your authority is running node X" information card

---

## 5. Deployment Model

### Primary Node (Authority 0)
```bash
# Environment
IS_PRIMARY=true
NODE_URL=https://primary.seashell.io
PORT=8080
JWT_SECRET=<strong-secret>
SUPERADMIN_PASSWORD=<strong-password>
SUPERADMIN_EMAIL=superadmin@authority0.com
NODE_SECRET=<shared-p2p-secret>  # distribute this to all node operators
DB_PATH=./db

# Start
./bin/seashell serve
```

### Secondary Node (Authority N)
```bash
# Environment
IS_PRIMARY=false
NODE_URL=https://authority-n.example.com
PRIMARY_NODE_URL=https://primary.seashell.io
PORT=8080
JWT_SECRET=<own-secret>  # independent JWT (users are local)
SUPERADMIN_PASSWORD=<own-password>  # own superadmin for local user mgmt
SUPERADMIN_EMAIL=admin@authority-n.com
NODE_SECRET=<same-shared-p2p-secret>  # must match primary's NODE_SECRET
DB_PATH=./db

# Start
./bin/seashell serve
# Node will self-register with primary, then wait for approval
# Once approved, syncs chain and begins normal operation
```

---

## 6. Implementation Order

### Phase 1 — Core Node Infrastructure (Go)
1. Add `model/node.go` with Node and NodeJoinRequest types
2. Add node store keys to `store/db.go`
3. Create `node/registry.go` — CRUD for node records
4. Update `config/config.go` — add NODE_URL, PRIMARY_NODE_URL, IS_PRIMARY, NODE_SECRET
5. Create `node/peer.go` — HTTP client for P2P communication
6. Create `node/sync.go` — blockchain sync logic
7. Create `node/broadcast.go` — block broadcast to peers
8. Create `node/healthchecker.go` — background peer pinger
9. Create `service/node_service.go` — orchestrates node lifecycle
10. Update `service/chain_service.go` — broadcast after block creation

### Phase 2 — API Routes (Go)
11. Create `api/handlers/node.go` — all node handlers (P2P + admin + superadmin)
12. Add `api/middleware/node.go` — RequireNodeSecret middleware
13. Update `api/server.go` — register new routes, bootstrap node on startup

### Phase 3 — Frontend Types & API Client
14. Add Node and NodeJoinRequest types to `example/src/types/api.ts`
15. Add node API methods to `example/src/lib/api.ts`

### Phase 4 — Frontend Pages & UI
16. Create `example/src/app/admin/network/page.tsx` — network overview
17. Create `example/src/app/admin/network/[id]/page.tsx` — node detail
18. Update `example/src/app/admin/page.tsx` — add network status card
19. Update admin navigation — add "Network" link
20. Update `example/src/app/superadmin/page.tsx` — add Node Requests tab

---

## 7. Security Considerations

- **P2P authentication**: All P2P endpoints require `X-Node-Secret` header matching `NODE_SECRET`
- **NODE_SECRET distribution**: Out-of-band secret sharing between node operators (similar to a consortium key)
- **Block validation**: Each node independently validates block signature and PoA round before accepting
- **Chain fork resolution**: Longest-valid-chain wins; nodes reject blocks that don't extend their known chain
- **Node URL validation**: Primary validates URL is reachable before approving (sends a ping)
- **Superadmin isolation**: Each node's superadmin only controls that node's local users; only primary node superadmin controls network membership

---

## 8. Files Changed Summary

### New Files
```
model/node.go
node/registry.go
node/peer.go
node/sync.go
node/broadcast.go
node/healthchecker.go
service/node_service.go
api/handlers/node.go
api/middleware/node.go
example/src/app/admin/network/page.tsx
example/src/app/admin/network/[id]/page.tsx
```

### Modified Files
```
config/config.go                              (+4 node config fields)
store/db.go                                   (+node store key helpers)
service/chain_service.go                      (+broadcast after mining)
api/server.go                                 (+node routes, node service bootstrap)
example/src/types/api.ts                      (+Node, NodeJoinRequest types)
example/src/lib/api.ts                        (+node API methods)
example/src/app/admin/page.tsx                (+network status card)
example/src/app/superadmin/page.tsx           (+node requests tab)
example/src/components/layout/header.tsx      (+network nav link)
```

Total: **11 new files**, **9 modified files**
