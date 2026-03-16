package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/goozt/seashell/service"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

// wsAuthMessage is the first message the client must send.
type wsAuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// wsAuthAck is sent back to confirm successful authentication.
type wsAuthAck struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// WSHandler serves GET /api/v1/ws.
type WSHandler struct {
	hub     *service.WSHub
	authSvc *service.AuthService
}

func NewWSHandler(hub *service.WSHub, authSvc *service.AuthService) *WSHandler {
	return &WSHandler{hub: hub, authSvc: authSvc}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Phase 1: authenticate via first message (5s deadline).
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(4001, "auth read failed"))
		conn.Close()
		return
	}

	var authMsg wsAuthMessage
	if err := json.Unmarshal(raw, &authMsg); err != nil || authMsg.Type != "auth" || authMsg.Token == "" {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(4001, "auth message required"))
		conn.Close()
		return
	}

	claims, err := h.authSvc.ValidateAccessToken(authMsg.Token)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(4001, "invalid token"))
		conn.Close()
		return
	}

	// Auth confirmed — send ack, then enter normal operation.
	ack, _ := json.Marshal(wsAuthAck{Type: "auth", Status: "ok"})
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
		conn.Close()
		return
	}

	// Phase 2: normal operation.
	defer conn.Close()

	connID, ch := h.hub.Subscribe(claims.UserID)
	defer h.hub.Unsubscribe(claims.UserID, connID)

	// Reset to keep-alive deadlines.
	conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	// Read pump — detects client disconnect; discards any client frames.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.hub.Unsubscribe(claims.UserID, connID)
				return
			}
		}
	}()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
