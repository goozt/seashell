package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

type PushHandler struct {
	db      *store.DB
	pushSvc *service.PushService
	cfg     *config.Config
}

func NewPushHandler(db *store.DB, pushSvc *service.PushService, cfg *config.Config) *PushHandler {
	return &PushHandler{db: db, pushSvc: pushSvc, cfg: cfg}
}

// GetVAPIDPublicKey handles GET /api/v1/push/vapid-public-key.
// Returns the VAPID public key so the frontend can create a subscription.
func (h *PushHandler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if !h.pushSvc.Enabled() {
		response.OK(w, map[string]string{"public_key": ""})
		return
	}
	response.OK(w, map[string]string{"public_key": h.cfg.VAPIDPublicKey})
}

// Subscribe handles POST /api/v1/user/push/subscribe.
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	var sub model.PushSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		response.BadRequest(w, "invalid subscription")
		return
	}
	if sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		response.BadRequest(w, "missing subscription fields")
		return
	}
	sub.UserID = claims.UserID
	if err := h.db.SavePushSubscription(&sub); err != nil {
		response.InternalError(w, "could not save subscription")
		return
	}
	response.OK(w, map[string]bool{"subscribed": true})
}

// Unsubscribe handles DELETE /api/v1/user/push/subscribe.
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		response.BadRequest(w, "endpoint required")
		return
	}
	_ = h.db.DeletePushSubscription(claims.UserID, body.Endpoint)
	response.OK(w, map[string]bool{"unsubscribed": true})
}
