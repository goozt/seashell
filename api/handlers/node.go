package handlers

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/node"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// NodeHandler serves both P2P (/p2p/v1/*) and REST node routes.
type NodeHandler struct {
	db       *store.DB
	chainSvc *service.ChainService
	nodeSvc  *service.NodeService
	isPrimary bool
	nodeURL  string
}

// NewNodeHandler creates a NodeHandler.
func NewNodeHandler(db *store.DB, chainSvc *service.ChainService, nodeSvc *service.NodeService, isPrimary bool, nodeURL string) *NodeHandler {
	return &NodeHandler{db: db, chainSvc: chainSvc, nodeSvc: nodeSvc, isPrimary: isPrimary, nodeURL: nodeURL}
}

// -------------------------------------------------------------------
// P2P routes (authenticated by X-Node-Secret header)
// -------------------------------------------------------------------

// P2PPing handles POST /p2p/v1/ping
// Used by peers to confirm this node is alive and exchange block height.
func (h *NodeHandler) P2PPing(w http.ResponseWriter, r *http.Request) {
	var req model.PingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}

	// Update the pinging node's status and last_seen.
	if req.NodeID != "" {
		n, err := h.db.GetNodeByID(req.NodeID)
		if err == nil {
			now := time.Now()
			n.LastSeenAt = &now
			n.BlockHeight = req.BlockHeight
			if n.Status == model.NodeStatusOffline {
				n.Status = model.NodeStatusActive
			}
			_ = h.db.SaveNode(n)
		}
	}

	// Return our own height.
	summaries, err := h.chainSvc.GetBlocks(1)
	height := uint64(0)
	if err == nil && len(summaries) > 0 {
		height = summaries[0].Height
	}

	response.OK(w, model.PingResponse{
		NodeID:      h.nodeSvc.SelfID(),
		BlockHeight: height,
		Status:      "ok",
	})
}

// P2PGetPeers handles GET /p2p/v1/peers
// Returns the list of active peers known to this node.
func (h *NodeHandler) P2PGetPeers(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.db.ListActiveNodes()
	if err != nil {
		response.InternalError(w, "could not load peers")
		return
	}
	peers := make([]model.Node, 0, len(nodes))
	for _, n := range nodes {
		peers = append(peers, *n)
	}
	response.OK(w, map[string]interface{}{"peers": peers})
}

// P2PGetChainSync handles GET /p2p/v1/chain/sync?from=N&limit=M
// Returns up to limit serialized blocks starting from height N.
func (h *NodeHandler) P2PGetChainSync(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	limitStr := r.URL.Query().Get("limit")

	from := uint64(0)
	if fromStr != "" {
		if v, err := strconv.ParseUint(fromStr, 10, 64); err == nil {
			from = v
		}
	}
	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	// Open chain directly (read-only, no ChainService lock needed for reads).
	var blocks []*blockchain.Block
	var fetchErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				fetchErr = nil
				blocks = nil
			}
		}()
		chain := blockchain.ContinueBlockChainAt(h.chainSvc.ChainDBPath(), false)
		defer chain.Close()
		blocks = chain.GetBlocksFromHeight(from, limit)
	}()

	if fetchErr != nil {
		response.InternalError(w, "could not read chain")
		return
	}

	serialized := make([]*node.SerializedBlock, len(blocks))
	for i, b := range blocks {
		serialized[i] = node.EncodeBlock(b)
	}
	response.OK(w, map[string]interface{}{
		"blocks": serialized,
		"total":  len(serialized),
	})
}

// P2PReceiveBlock handles POST /p2p/v1/blocks
// Accepts a block broadcast from a peer and adds it to the local chain.
func (h *NodeHandler) P2PReceiveBlock(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Block *node.SerializedBlock `json:"block"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Block == nil {
		response.BadRequest(w, "invalid block payload")
		return
	}

	block, err := payload.Block.Decode()
	if err != nil {
		response.BadRequest(w, "decode block: "+err.Error())
		return
	}

	if err := h.chainSvc.AddExternalBlock(block); err != nil {
		response.BadRequest(w, "rejected: "+err.Error())
		return
	}

	response.OK(w, map[string]bool{"accepted": true})
}

// P2PGetValidators handles GET /p2p/v1/validators
func (h *NodeHandler) P2PGetValidators(w http.ResponseWriter, r *http.Request) {
	var validators []string
	func() {
		defer func() { recover() }()
		chain := blockchain.ContinueBlockChainAt(h.chainSvc.ChainDBPath(), false)
		defer chain.Close()
		raw := blockchain.GetValidators(chain.Database)
		for _, v := range raw {
			validators = append(validators, hex.EncodeToString(v))
		}
	}()
	response.OK(w, map[string]interface{}{"validators": validators})
}

// P2PReceiveValidator handles POST /p2p/v1/validators
// Adds a newly approved validator key to the local blockchain DB.
func (h *NodeHandler) P2PReceiveValidator(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ValidatorPubKey string `json:"validator_pubkey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.ValidatorPubKey == "" {
		response.BadRequest(w, "validator_pubkey required")
		return
	}

	pubKey, err := hex.DecodeString(payload.ValidatorPubKey)
	if err != nil {
		response.BadRequest(w, "invalid validator_pubkey hex")
		return
	}

	var addErr error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				addErr = nil // ignore; DB might not be initialised yet
			}
		}()
		chain := blockchain.ContinueBlockChainAt(h.chainSvc.ChainDBPath(), false)
		defer chain.Close()
		blockchain.AddValidatorToDB(chain.Database, pubKey)
	}()
	if addErr != nil {
		response.InternalError(w, "could not add validator")
		return
	}
	response.OK(w, map[string]bool{"accepted": true})
}

// -------------------------------------------------------------------
// Public registration route (new node → primary)
// -------------------------------------------------------------------

// RegisterNode handles POST /api/v1/nodes/register
// A new node POSTs this to the primary to request joining the network.
func (h *NodeHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	if !h.isPrimary {
		response.Forbidden(w)
		return
	}
	var req model.NodeRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if req.NodeURL == "" || req.AuthorityName == "" || req.ValidatorPubKey == "" {
		response.BadRequest(w, "node_url, authority_name, and validator_pubkey are required")
		return
	}

	joinReq := &model.NodeJoinRequest{
		ID:              uuid.New().String(),
		NodeURL:         req.NodeURL,
		AuthorityName:   req.AuthorityName,
		AdminEmail:      req.AdminEmail,
		ValidatorPubKey: req.ValidatorPubKey,
		Status:          "pending",
		CreatedAt:       time.Now(),
	}
	if err := h.db.SaveNodeJoinRequest(joinReq); err != nil {
		response.InternalError(w, "could not save join request")
		return
	}
	response.Created(w, map[string]string{
		"message":    "join request submitted, awaiting superadmin approval",
		"request_id": joinReq.ID,
	})
}

// -------------------------------------------------------------------
// Admin routes (JWT + admin/superadmin)
// -------------------------------------------------------------------

// GetNodes handles GET /api/v1/admin/nodes
func (h *NodeHandler) GetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.nodeSvc.GetNetworkNodes()
	if err != nil {
		response.InternalError(w, "could not load nodes")
		return
	}
	response.OK(w, map[string]interface{}{"nodes": nodes})
}

// GetNode handles GET /api/v1/admin/nodes/{id}
func (h *NodeHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.db.GetNodeByID(id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}
	response.OK(w, n)
}

// -------------------------------------------------------------------
// SuperAdmin routes (JWT + superadmin, primary node only)
// -------------------------------------------------------------------

// GetNodeJoinRequests handles GET /api/v1/superadmin/node-requests
func (h *NodeHandler) GetNodeJoinRequests(w http.ResponseWriter, r *http.Request) {
	if !h.isPrimary {
		response.Forbidden(w)
		return
	}
	reqs, err := h.db.ListNodeJoinRequests()
	if err != nil {
		response.InternalError(w, "could not load requests")
		return
	}
	response.OK(w, map[string]interface{}{"requests": reqs})
}

// ApproveNodeJoinRequest handles POST /api/v1/superadmin/node-requests/{id}/approve
func (h *NodeHandler) ApproveNodeJoinRequest(w http.ResponseWriter, r *http.Request) {
	if !h.isPrimary {
		response.Forbidden(w)
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	req, err := h.db.GetNodeJoinRequestByID(id)
	if err != nil || req.Status != "pending" {
		response.NotFound(w, "pending request not found")
		return
	}

	now := time.Now()
	req.Status = "approved"
	req.ReviewedAt = &now
	req.ReviewedBy = claims.UserID
	if err := h.db.SaveNodeJoinRequest(req); err != nil {
		response.InternalError(w, "could not update request")
		return
	}

	// Create the node record.
	newNode := &model.Node{
		ID:              uuid.New().String(),
		NodeURL:         req.NodeURL,
		AuthorityName:   req.AuthorityName,
		ValidatorPubKey: req.ValidatorPubKey,
		Status:          model.NodeStatusActive,
		IsPrimary:       false,
		RegisteredAt:    req.CreatedAt,
		ApprovedAt:      &now,
		ApprovedBy:      claims.UserID,
		Version:         "1.0.0",
	}
	if err := h.db.SaveNode(newNode); err != nil {
		response.InternalError(w, "could not save node")
		return
	}

	// Add validator key to local blockchain and broadcast to peers.
	if err := h.nodeSvc.AddValidatorToChain(req.ValidatorPubKey); err != nil {
		response.InternalError(w, "could not register validator: "+err.Error())
		return
	}

	// Return the new node record plus the current active peer list.
	activePeers, _ := h.db.ListActiveNodes()
	peers := make([]model.Node, 0, len(activePeers))
	for _, p := range activePeers {
		peers = append(peers, *p)
	}
	response.Created(w, model.NodeApproveResponse{
		NodeID: newNode.ID,
		Peers:  peers,
	})
}

// RejectNodeJoinRequest handles POST /api/v1/superadmin/node-requests/{id}/reject
func (h *NodeHandler) RejectNodeJoinRequest(w http.ResponseWriter, r *http.Request) {
	if !h.isPrimary {
		response.Forbidden(w)
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	req, err := h.db.GetNodeJoinRequestByID(id)
	if err != nil || req.Status != "pending" {
		response.NotFound(w, "pending request not found")
		return
	}

	var body model.NodeRejectRequest
	_ = json.NewDecoder(r.Body).Decode(&body)

	now := time.Now()
	req.Status = "rejected"
	req.ReviewedAt = &now
	req.ReviewedBy = claims.UserID
	req.RejectionReason = body.Reason
	if err := h.db.SaveNodeJoinRequest(req); err != nil {
		response.InternalError(w, "could not update request")
		return
	}
	response.OK(w, map[string]string{"message": "rejected"})
}

// SuspendNode handles POST /api/v1/superadmin/nodes/{id}/suspend
func (h *NodeHandler) SuspendNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.db.GetNodeByID(id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}
	if n.IsPrimary {
		response.BadRequest(w, "cannot suspend the primary node")
		return
	}
	n.Status = model.NodeStatusSuspended
	if err := h.db.SaveNode(n); err != nil {
		response.InternalError(w, "could not suspend node")
		return
	}
	response.OK(w, n)
}

// ReinstateNode handles POST /api/v1/superadmin/nodes/{id}/reinstate
func (h *NodeHandler) ReinstateNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.db.GetNodeByID(id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}
	n.Status = model.NodeStatusActive
	if err := h.db.SaveNode(n); err != nil {
		response.InternalError(w, "could not reinstate node")
		return
	}
	response.OK(w, n)
}
