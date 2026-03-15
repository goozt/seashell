package model

import "time"

// Invitation is a one-time-use code that allows a user to join an authority.
type Invitation struct {
	Code        string    `json:"code"`
	AuthorityID string    `json:"authority_id"`
	CreatedBy   string    `json:"created_by"`
	UsedBy      string    `json:"used_by,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
	CreatedAt   time.Time `json:"created_at"`
}

// IsValid returns true if the invitation can still be used.
func (inv *Invitation) IsValid() bool {
	return !inv.Used && inv.ExpiresAt.After(time.Now())
}

// JoinAuthorityRequest is the payload for POST /user/authority/join.
type JoinAuthorityRequest struct {
	Code string `json:"code"`
}
