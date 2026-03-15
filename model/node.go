package model

import "time"

const (
	NodeStatusPending   = "pending"
	NodeStatusActive    = "active"
	NodeStatusOffline   = "offline"
	NodeStatusSuspended = "suspended"
	NodeStatusRejected  = "rejected"

	NodeTierPrimary  = "primary"
	NodeTierRegional = "regional"
	NodeTierBranch   = "branch"
)

// Node represents a SeaShell server participating in the blockchain network.
// Each node is associated with exactly one Authority.
type Node struct {
	ID              string     `json:"id"`
	AuthorityID     string     `json:"authority_id"`
	AuthorityName   string     `json:"authority_name"`
	NodeURL         string     `json:"node_url"`
	ValidatorPubKey string     `json:"validator_pubkey"` // hex-encoded; set after approval
	Status          string     `json:"status"`
	IsPrimary       bool       `json:"is_primary"`
	BlockHeight     uint64     `json:"block_height"`  // cached from last ping
	Version         string     `json:"version"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	RegisteredAt    time.Time  `json:"registered_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`

	// 3-tier topology fields. Existing records with empty NodeTier are treated as "branch".
	NodeTier        string `json:"node_tier,omitempty"`        // primary|regional|branch
	ParentNodeID    string `json:"parent_node_id,omitempty"`
	ParentNodeURL   string `json:"parent_node_url,omitempty"`
	CertFingerprint string `json:"cert_fingerprint,omitempty"` // SHA-256 hex of TLS cert
}

// NodeJoinRequest is a request from a new node to join the network.
// Created by the new node; reviewed by the primary node's superadmin.
type NodeJoinRequest struct {
	ID              string     `json:"id"`
	NodeURL         string     `json:"node_url"`
	AuthorityName   string     `json:"authority_name"`
	AdminEmail      string     `json:"admin_email"`
	ValidatorPubKey string     `json:"validator_pubkey"` // hex-encoded public key the new node will use
	Status          string     `json:"status"`           // pending|approved|rejected
	CreatedAt       time.Time  `json:"created_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	ApprovedByTier  string     `json:"approved_by_tier,omitempty"` // tier of approving admin

	// 3-tier topology fields.
	NodeTier      string `json:"node_tier,omitempty"`
	ParentNodeID  string `json:"parent_node_id,omitempty"`
	ParentNodeURL string `json:"parent_node_url,omitempty"`
}

// NodeRegisterRequest is the payload POSTed by a new node to /api/v1/nodes/register.
type NodeRegisterRequest struct {
	NodeURL         string `json:"node_url"`
	AuthorityName   string `json:"authority_name"`
	AdminEmail      string `json:"admin_email"`
	ValidatorPubKey string `json:"validator_pubkey"`
	NodeTier        string `json:"node_tier,omitempty"`
	ParentNodeID    string `json:"parent_node_id,omitempty"`
	ParentNodeURL   string `json:"parent_node_url,omitempty"`
}

// NodeApproveResponse is returned to the new node after its request is approved.
type NodeApproveResponse struct {
	NodeID string `json:"node_id"`
	Peers  []Node `json:"peers"` // existing active nodes
}

// NodeRejectRequest is the payload for rejecting a node join request.
type NodeRejectRequest struct {
	Reason string `json:"reason"`
}

// CoSignRequest is the P2P payload for POST /p2p/v1/cosign.
type CoSignRequest struct {
	BlockHash     string `json:"block_hash"`     // hex-encoded SHA-256
	Height        uint64 `json:"height"`
	LeadValidator string `json:"lead_validator"` // hex-encoded 64-byte pubkey
}

// CoSignResponse is returned by a co-signer node.
type CoSignResponse struct {
	PubKey string `json:"pub_key"` // hex-encoded 64-byte pubkey
	Sig    string `json:"sig"`     // hex-encoded 64-byte r||s signature
}

// PingRequest is the P2P ping payload.
type PingRequest struct {
	NodeID      string `json:"node_id"`
	BlockHeight uint64 `json:"block_height"`
}

// PingResponse is the P2P ping response.
type PingResponse struct {
	NodeID      string `json:"node_id"`
	BlockHeight uint64 `json:"block_height"`
	Status      string `json:"status"`
}
