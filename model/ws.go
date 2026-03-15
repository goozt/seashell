package model

import "encoding/json"

const (
	WSTypeNotification = "notification"
	WSTypeAlert        = "alert"
	WSTypeSupport      = "support"
)

// WSMessage is the envelope sent over the WebSocket connection.
type WSMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

// NotificationPayload is used for WSTypeNotification messages.
type NotificationPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Link  string `json:"link,omitempty"`
}

// AlertPayload is used for WSTypeAlert messages (toast / device push).
type AlertPayload struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Severity string `json:"severity"` // "info" | "warning" | "error"
}

// SupportPayload is used for WSTypeSupport messages (ticket reply).
type SupportPayload struct {
	TicketID string      `json:"ticket_id"`
	Reply    TicketReply `json:"reply"`
}
