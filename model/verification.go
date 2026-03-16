package model

import "time"

// Verification method constants.
const (
	VerifyMethodManual = "manual"
	// Future: VerifyMethodAPI = "api", VerifyMethodOAuth = "oauth"
)

// Verification submission status constants.
const (
	VerifyStatusPending  = "pending"
	VerifyStatusApproved = "approved"
	VerifyStatusRejected = "rejected"
)

// FieldDefinition describes a single custom form field configured by the authority owner.
type FieldDefinition struct {
	Name        string   `json:"name"`                   // unique field key, e.g. "pan_card"
	Label       string   `json:"label"`                  // display label, e.g. "PAN Card Number"
	InputType   string   `json:"input_type"`             // "text" | "number" | "email" | "textarea" | "select"
	Required    bool     `json:"required"`
	Pattern     string   `json:"pattern,omitempty"`      // regex for validation
	MinLength   int      `json:"min_length,omitempty"`
	MaxLength   int      `json:"max_length,omitempty"`
	Options     []string `json:"options,omitempty"`       // for "select" input type
	Placeholder string   `json:"placeholder,omitempty"`
}

// VerificationConfig holds the verification setup for an authority.
type VerificationConfig struct {
	AuthorityID string            `json:"authority_id"`
	Method      string            `json:"method"`  // "manual" (extensible)
	Fields      []FieldDefinition `json:"fields"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	// Future: ProviderConfig map[string]string `json:"provider_config,omitempty"`
}

// VerificationSubmission holds one user's verification attempt.
type VerificationSubmission struct {
	ID          string            `json:"id"` // UUID
	AuthorityID string            `json:"authority_id"`
	UserID      string            `json:"user_id"`
	Method      string            `json:"method"`
	FieldValues map[string]string `json:"field_values"` // field name -> user-entered value
	Status      string            `json:"status"`       // pending | approved | rejected
	ReviewedBy  string            `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time        `json:"reviewed_at,omitempty"`
	Remarks     string            `json:"remarks,omitempty"` // rejection reason / owner remarks
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SaveVerificationConfigRequest is the payload for PUT /authority/verification/config.
type SaveVerificationConfigRequest struct {
	Method string            `json:"method"`
	Fields []FieldDefinition `json:"fields"`
}

// Validate returns a non-empty string describing the error, or "" if valid.
func (r *SaveVerificationConfigRequest) Validate() string {
	if r.Method != VerifyMethodManual {
		return "unsupported verification method; currently only 'manual' is supported"
	}
	if len(r.Fields) == 0 {
		return "at least one field is required"
	}
	names := make(map[string]bool, len(r.Fields))
	validTypes := map[string]bool{"text": true, "number": true, "email": true, "textarea": true, "select": true}
	for _, f := range r.Fields {
		if f.Name == "" || f.Label == "" {
			return "each field must have a name and label"
		}
		if !validTypes[f.InputType] {
			return "invalid input type: " + f.InputType + "; must be text, number, email, textarea, or select"
		}
		if names[f.Name] {
			return "duplicate field name: " + f.Name
		}
		if f.InputType == "select" && len(f.Options) == 0 {
			return "select field '" + f.Label + "' must have at least one option"
		}
		names[f.Name] = true
	}
	return ""
}

// SubmitVerificationRequest is the payload for POST /user/verification.
type SubmitVerificationRequest struct {
	FieldValues map[string]string `json:"field_values"`
}

// ReviewVerificationRequest is the payload for POST /authority/verification/submissions/{id}/review.
type ReviewVerificationRequest struct {
	Action  string `json:"action"`            // "approve" | "reject"
	Remarks string `json:"remarks,omitempty"`
}

// Validate returns a non-empty string describing the error, or "" if valid.
func (r *ReviewVerificationRequest) Validate() string {
	if r.Action != "approve" && r.Action != "reject" {
		return "action must be 'approve' or 'reject'"
	}
	if r.Action == "reject" && r.Remarks == "" {
		return "remarks are required when rejecting"
	}
	return ""
}
