package model

import "time"

const (
	NodeStatusPending   = "pending"
	NodeStatusActive    = "active"
	NodeStatusOffline   = "offline"
	NodeStatusSuspended = "suspended"
	NodeStatusRejected  = "rejected"
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
}

// NodeRegisterRequest is the payload POSTed by a new node to /api/v1/nodes/register.
type NodeRegisterRequest struct {
	NodeURL         string `json:"node_url"`
	AuthorityName   string `json:"authority_name"`
	AdminEmail      string `json:"admin_email"`
	ValidatorPubKey string `json:"validator_pubkey"`
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
