package model

import "time"

// KYCRecord holds the identity verification result for a user.
// The raw National ID is never stored; only a masked form and an HMAC-keyed dedup index.
type KYCRecord struct {
	UserID           string     `json:"user_id"`
	NationalIDMasked string     `json:"national_id_masked,omitempty"` // e.g. "XXXX-XXXX-1234"
	TaxIDMasked      string     `json:"tax_id_masked,omitempty"`
	Status           string     `json:"status"` // unverified|pending|verified|rejected
	VerifiedName     string     `json:"verified_name,omitempty"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	RejectedReason   string     `json:"rejected_reason,omitempty"`
	NodeID           string     `json:"node_id,omitempty"` // branch node that performed verification
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// KYCInitRequest is the payload for POST /api/v1/user/kyc.
type KYCInitRequest struct {
	NationalID string `json:"national_id"`
	TaxID      string `json:"tax_id,omitempty"`
}

// KYCRejectRequest is the payload for POST /api/v1/admin/kyc/{userID}/reject.
type KYCRejectRequest struct {
	Reason string `json:"reason"`
}

// KYCSummary is an aggregate of KYC statuses for a node.
type KYCSummary struct {
	NodeID     string `json:"node_id"`
	Total      int    `json:"total"`
	Verified   int    `json:"verified"`
	Pending    int    `json:"pending"`
	Rejected   int    `json:"rejected"`
	Unverified int    `json:"unverified"`
}
