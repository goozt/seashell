package service

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/node"
	"github.com/goozt/seashell/store"
)

// NodeService orchestrates this node's identity, registration with the primary
// node, blockchain sync from peers, and block broadcasting.
type NodeService struct {
	db           *store.DB
	chainSvc     *ChainService
	cfg          *config.Config
	selfID       string
	healthChecker *node.HealthChecker
}

// NewNodeService creates a NodeService.  Call Bootstrap() before serving requests.
func NewNodeService(db *store.DB, chainSvc *ChainService, cfg *config.Config) *NodeService {
	return &NodeService{db: db, chainSvc: chainSvc, cfg: cfg}
}

// Bootstrap performs node-startup initialisation:
//   - If primary: ensures a self-registration record exists.
//   - If secondary: sends a join request to the primary and waits for approval,
//     then syncs the blockchain.
//
// Bootstrap is called once during server startup and blocks until the node is
// ready (or an error occurs).
func (s *NodeService) Bootstrap() error {
	if s.cfg.IsPrimary {
		return s.bootstrapPrimary()
	}
	return s.bootstrapSecondary()
}

func (s *NodeService) bootstrapPrimary() error {
	// Ensure a Node record exists for ourselves.
	nodes, err := s.db.ListNodes()
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	for _, n := range nodes {
		if n.IsPrimary {
			s.selfID = n.ID
			s.startHealthChecker()
			return nil
		}
	}

	// Create the primary node record.
	now := time.Now()
	primaryNode := &model.Node{
		ID:           uuid.New().String(),
		NodeURL:      s.cfg.NodeURL,
		AuthorityID:  "",
		AuthorityName: "Primary Node",
		Status:       model.NodeStatusActive,
		IsPrimary:    true,
		RegisteredAt: now,
		ApprovedAt:   &now,
		Version:      "1.0.0",
	}
	if err := s.db.SaveNode(primaryNode); err != nil {
		return fmt.Errorf("save primary node: %w", err)
	}
	s.selfID = primaryNode.ID
	log.Printf("node: primary node registered (id=%s)", s.selfID)
	s.startHealthChecker()
	return nil
}

func (s *NodeService) bootstrapSecondary() error {
	// Check if we already have an approved node record locally.
	nodes, err := s.db.ListNodes()
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	for _, n := range nodes {
		if n.NodeURL == s.cfg.NodeURL && n.Status == model.NodeStatusActive {
			s.selfID = n.ID
			s.startHealthChecker()
			return nil
		}
	}

	// Not yet registered — we can't register automatically without an authority.
	// The node will accept incoming P2P traffic but won't broadcast until approved.
	log.Printf("node: not yet approved by primary. Waiting for approval via API.")
	return nil
}

// SelfID returns this node's ID.
func (s *NodeService) SelfID() string {
	return s.selfID
}

// OnBlockMined is called (in a goroutine) when a new block is mined locally.
// It fetches the block and broadcasts it to all active peers.
func (s *NodeService) OnBlockMined(blockHash []byte) {
	if s.cfg.NodeURL == "" {
		return
	}
	block, _, err := s.chainSvc.GetBlock(hex.EncodeToString(blockHash))
	if err != nil || block == nil {
		log.Printf("node: broadcast: could not load block %x: %v", blockHash, err)
		return
	}

	peers, err := s.db.ListActiveNodes()
	if err != nil {
		log.Printf("node: broadcast: list peers: %v", err)
		return
	}

	var filtered []model.Node
	for _, p := range peers {
		if p.NodeURL != s.cfg.NodeURL {
			filtered = append(filtered, *p)
		}
	}

	if len(filtered) == 0 {
		return
	}
	log.Printf("node: broadcasting block %x to %d peers", blockHash[:4], len(filtered))
	node.BroadcastBlock(filtered, block, s.cfg.NodeSecret)
}

// GetNetworkNodes returns the full node list.
// On the primary node, this is the authoritative registry.
// On secondary nodes, we first try to fetch from the primary, falling back to local cache.
func (s *NodeService) GetNetworkNodes() ([]*model.Node, error) {
	if s.cfg.IsPrimary || s.cfg.PrimaryNodeURL == "" {
		return s.db.ListNodes()
	}

	// Try primary node first.
	client := node.NewPeerClient(s.cfg.PrimaryNodeURL, s.cfg.NodeSecret)
	peers, err := client.GetPeers()
	if err == nil {
		result := make([]*model.Node, len(peers))
		for i := range peers {
			p := peers[i]
			result[i] = &p
		}
		return result, nil
	}
	log.Printf("node: could not fetch peer list from primary (%v), using local cache", err)
	return s.db.ListNodes()
}

// AddValidatorToChain registers pubKeyHex as a validator in the local blockchain DB
// and notifies all active peers.
func (s *NodeService) AddValidatorToChain(pubKeyHex string) error {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return fmt.Errorf("decode pubkey: %w", err)
	}

	// Open chain and add validator.
	s.chainSvc.mu.Lock()
	var addErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				addErr = fmt.Errorf("%v", r)
			}
		}()
		chain := blockchain.ContinueBlockChainAt(s.chainSvc.chainDBPath, false)
		defer chain.Close()
		blockchain.AddValidatorToDB(chain.Database, pubKey)
	}()
	s.chainSvc.mu.Unlock()
	if addErr != nil {
		return addErr
	}

	// Broadcast to peers.
	peers, err := s.db.ListActiveNodes()
	if err != nil {
		log.Printf("node: broadcast validator: list peers: %v", err)
		return nil
	}
	var filtered []model.Node
	for _, p := range peers {
		if p.NodeURL != s.cfg.NodeURL {
			filtered = append(filtered, *p)
		}
	}
	go node.BroadcastNewValidator(filtered, pubKeyHex, s.cfg.NodeSecret)
	return nil
}

// SyncFromPeer syncs this node's blockchain from the given peer URL.
func (s *NodeService) SyncFromPeer(peerURL string) error {
	s.chainSvc.mu.Lock()
	defer s.chainSvc.mu.Unlock()

	chain, err := s.chainSvc.openChain()
	if err != nil {
		return fmt.Errorf("open chain: %w", err)
	}
	defer chain.Close()

	client := node.NewPeerClient(peerURL, s.cfg.NodeSecret)
	return node.IncrementalSync(client, chain)
}

// startHealthChecker begins periodic peer pings.
func (s *NodeService) startHealthChecker() {
	s.healthChecker = node.NewHealthChecker(
		s.db, s.selfID, s.cfg.NodeURL, s.cfg.NodeSecret, 30*time.Second,
	)
	s.healthChecker.Start()
	log.Printf("node: health checker started")
}

// Stop shuts down the health checker.
func (s *NodeService) Stop() {
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}
}
