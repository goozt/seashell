package service

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/goozt/seashell/model"
)

// WSHub manages per-user WebSocket subscriber channels.
// Each user may have multiple connections (tabs); all receive every message.
type WSHub struct {
	mu   sync.RWMutex
	subs map[string]map[string]chan []byte // userID → connID → channel
}

func NewWSHub() *WSHub {
	return &WSHub{subs: make(map[string]map[string]chan []byte)}
}

// Subscribe registers a new connection channel for a user.
// Returns connID and the read-only channel. Call Unsubscribe when done.
func (h *WSHub) Subscribe(userID string) (connID string, ch <-chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	connID = uuid.New().String()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[string]chan []byte)
	}
	c := make(chan []byte, 32)
	h.subs[userID][connID] = c
	return connID, c
}

// Unsubscribe removes a connection channel.
func (h *WSHub) Unsubscribe(userID, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.subs[userID]; ok {
		if c, ok := m[connID]; ok {
			close(c)
			delete(m, connID)
		}
		if len(m) == 0 {
			delete(h.subs, userID)
		}
	}
}

// SendTo delivers a WSMessage to all active connections of a user.
func (h *WSHub) SendTo(userID string, msgType, id string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := model.WSMessage{Type: msgType, ID: id, Payload: json.RawMessage(raw)}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[userID] {
		select {
		case ch <- data:
		default: // drop if full
		}
	}
}
