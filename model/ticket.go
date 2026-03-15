package model

import "time"

const (
	TicketStatusOpen       = "open"
	TicketStatusInProgress = "in_progress"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"
)

// Ticket is a support request created by an authority owner.
type Ticket struct {
	ID          string        `json:"id"`
	AuthorityID string        `json:"authority_id"`
	CreatedBy   string        `json:"created_by"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	Replies     []TicketReply `json:"replies"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// TicketReply is a response within a ticket thread.
type TicketReply struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	AuthorRole string   `json:"author_role"` // for UI colour-coding
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
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
