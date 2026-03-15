package model

import (
	"strings"
	"time"
)

const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleUser       = "user"

	AuthorityRoleOwner  = "owner"
	AuthorityRoleMember = "member"
)

// User represents a registered account in the system.
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"password_hash,omitempty"`
	Role          string    `json:"role"`
	AuthorityID   string    `json:"authority_id"`
	AuthorityRole string    `json:"authority_role"`
	WalletAddress string    `json:"wallet_address"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// IsAffiliated returns true if the user belongs to an authority.
func (u *User) IsAffiliated() bool {
	return u.AuthorityID != ""
}

// IsAuthorityOwner returns true if the user owns their authority.
func (u *User) IsAuthorityOwner() bool {
	return u.AuthorityRole == AuthorityRoleOwner
}

// HasRole returns true if the user has one of the given roles.
func (u *User) HasRole(roles ...string) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

// SafeCopy returns a copy safe for API responses (no password hash).
func (u *User) SafeCopy() User {
	c := *u
	c.PasswordHash = ""
	return c
}

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate returns a non-nil error string if the request is invalid.
func (r *RegisterRequest) Validate() string {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if len(r.Username) < 3 || len(r.Username) > 50 {
		return "username must be 3–50 characters"
	}
	if !strings.Contains(r.Email, "@") {
		return "invalid email address"
	}
	if len(r.Password) < 8 {
		return "password must be at least 8 characters"
	}
	return ""
}

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateProfileRequest is the payload for PUT /user/me.
type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
