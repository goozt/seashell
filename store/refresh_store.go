package store

import (
	"encoding/json"
	"fmt"
	"time"
)

type refreshRecord struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SetRefreshToken persists a refresh token record.
func (d *DB) SetRefreshToken(tokenID, userID string, expiresAt time.Time) error {
	rec := refreshRecord{UserID: userID, ExpiresAt: expiresAt}
	return d.set(fmt.Sprintf("refresh:%s", tokenID), rec)
}

// GetRefreshToken retrieves the user ID and expiry for a refresh token.
func (d *DB) GetRefreshToken(tokenID string) (string, time.Time, error) {
	var rec refreshRecord
	if err := d.get(fmt.Sprintf("refresh:%s", tokenID), &rec); err != nil {
		return "", time.Time{}, err
	}
	return rec.UserID, rec.ExpiresAt, nil
}

// DeleteRefreshToken removes a refresh token (logout / expiry cleanup).
func (d *DB) DeleteRefreshToken(tokenID string) error {
	return d.del(fmt.Sprintf("refresh:%s", tokenID))
}

// internal helper used by store methods that need JSON unmarshal by value
func unmarshalRefreshRecord(data []byte) (refreshRecord, error) {
	var rec refreshRecord
	err := json.Unmarshal(data, &rec)
	return rec, err
}
