package service

import (
	"encoding/json"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

// PushService sends Web Push notifications to subscribed devices.
type PushService struct {
	db              *store.DB
	vapidPublicKey  string
	vapidPrivateKey string
	vapidEmail      string
}

func NewPushService(db *store.DB, publicKey, privateKey, email string) *PushService {
	return &PushService{
		db:              db,
		vapidPublicKey:  publicKey,
		vapidPrivateKey: privateKey,
		vapidEmail:      email,
	}
}

// Enabled returns true if VAPID keys are configured.
func (s *PushService) Enabled() bool {
	return s.vapidPublicKey != "" && s.vapidPrivateKey != ""
}

// SendToUser sends a push notification to all subscribed devices of a user.
// Silently skips if push is not configured.
func (s *PushService) SendToUser(userID string, payload model.PushPayload) {
	if !s.Enabled() {
		return
	}
	subs, err := s.db.ListPushSubscriptions(userID)
	if err != nil || len(subs) == 0 {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, sub := range subs {
		wsSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.Keys.P256dh,
				Auth:   sub.Keys.Auth,
			},
		}
		resp, err := webpush.SendNotification(data, wsSub, &webpush.Options{
			Subscriber:      s.vapidEmail,
			VAPIDPublicKey:  s.vapidPublicKey,
			VAPIDPrivateKey: s.vapidPrivateKey,
			TTL:             86400, // 24 hours
		})
		if err != nil {
			log.Printf("push: send to %s failed: %v", userID, err)
			continue
		}
		resp.Body.Close()
		// 410 Gone = subscription expired; remove it.
		if resp.StatusCode == 410 {
			_ = s.db.DeletePushSubscription(userID, sub.Endpoint)
		}
	}
}
