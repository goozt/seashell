package model

import "time"

const (
	TicketStatusOpen       = "open"
	TicketStatusInProgress = "in_progress"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"

	// EscalatedTo values — which tier currently owns this ticket.
	TicketLevelNode     = "node"       // default: handled by node admin
	TicketLevelRegional = "regional"   // escalated to regional admin
	TicketLevelSuper    = "superadmin" // escalated to superadmin
)

// Ticket is a support request that can be escalated up the 3-tier hierarchy.
type Ticket struct {
	ID          string        `json:"id"`
	AuthorityID string        `json:"authority_id"`
	CreatedBy   string        `json:"created_by"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	// EscalatedTo tracks which tier currently owns the ticket.
	// Empty / "node" = node admin, "regional" = regional admin, "superadmin" = superadmin.
	EscalatedTo    string        `json:"escalated_to,omitempty"`
	EscalationNote string        `json:"escalation_note,omitempty"`
	Replies        []TicketReply `json:"replies"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// TicketReply is a response within a ticket thread.
type TicketReply struct {
	ID             string    `json:"id"`
	AuthorID       string    `json:"author_id"`
	AuthorUsername string    `json:"author_username"`
	AuthorRole     string    `json:"author_role"` // for UI colour-coding
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateTicketRequest is the payload for POST /user/tickets.
type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ReplyTicketRequest is the payload for POST /user/tickets/:id/reply.
type ReplyTicketRequest struct {
	Message string `json:"message"`
}

// UpdateTicketRequest is the payload for PUT /admin/tickets/:id.
type UpdateTicketRequest struct {
	Status string `json:"status"`
}

// EscalateTicketRequest is the payload for POST /admin/tickets/:id/escalate.
type EscalateTicketRequest struct {
	Note string `json:"note"` // optional reason for escalation
}
