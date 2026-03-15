package store

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/goozt/seashell/model"
)

const prefixPush = "push:"

func pushKey(userID, endpoint string) string {
	h := sha256.Sum256([]byte(endpoint))
	return prefixPush + userID + ":" + hex.EncodeToString(h[:])[:16]
}

// SavePushSubscription stores or replaces a push subscription.
func (d *DB) SavePushSubscription(sub *model.PushSubscription) error {
	return d.set(pushKey(sub.UserID, sub.Endpoint), sub)
}

// DeletePushSubscription removes a push subscription by endpoint.
func (d *DB) DeletePushSubscription(userID, endpoint string) error {
	return d.del(pushKey(userID, endpoint))
}

// ListPushSubscriptions returns all push subscriptions for a user.
func (d *DB) ListPushSubscriptions(userID string) ([]*model.PushSubscription, error) {
	prefix := prefixPush + userID + ":"
	var result []*model.PushSubscription
	err := d.iterPrefix(prefix, func(val []byte) error {
		var sub model.PushSubscription
		if err := jsonUnmarshal(val, &sub); err != nil {
			return err
		}
		result = append(result, &sub)
		return nil
	})
	return result, err
}
