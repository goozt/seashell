package model

import "time"

const (
	AuthorityStatusPending   = "pending"
	AuthorityStatusActive    = "active"
	AuthorityStatusSuspended = "suspended"
	AuthorityStatusRejected  = "rejected"
)

// Authority represents an organizational entity that acts as a PoA validator.
type Authority struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	OwnerID         string     `json:"owner_id"`
	Status          string     `json:"status"`
	ValidatorPubKey string     `json:"validator_pubkey"` // hex-encoded X||Y bytes; empty until approved
	BasePrice       float64    `json:"base_price"`       // initial credits per SHELL
	SensitivityK    float64    `json:"sensitivity_k"`    // velocity multiplier for value engine
	CreatedAt       time.Time  `json:"created_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}

// IsActive returns true if the authority can participate in the blockchain.
func (a *Authority) IsActive() bool {
	return a.Status == AuthorityStatusActive
}

// CreateAuthorityRequest is the payload for POST /user/authority.
type CreateAuthorityRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ApproveAuthorityRequest is the payload for POST /admin/authority-requests/:id/approve.
type ApproveAuthorityRequest struct {
	BasePrice    float64 `json:"base_price"`    // credits per SHELL at genesis
	SensitivityK float64 `json:"sensitivity_k"` // 0.0–1.0 recommended
}

// RejectAuthorityRequest is the payload for POST /admin/authority-requests/:id/reject.
type RejectAuthorityRequest struct {
	Reason string `json:"reason"`
}
