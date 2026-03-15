package store

import (
	"github.com/goozt/seashell/model"
)

const (
	prefixKYC          = "kyc:"
	prefixKYCNationalID = "kyc:national_id:"
	prefixKYCTaxID      = "kyc:tax_id:"
)

// SaveKYCRecord persists a KYC record for a user.
func (d *DB) SaveKYCRecord(r *model.KYCRecord) error {
	return d.set(prefixKYC+r.UserID, r)
}

// GetKYCRecord retrieves the KYC record for a user.
func (d *DB) GetKYCRecord(userID string) (*model.KYCRecord, error) {
	var r model.KYCRecord
	if err := d.get(prefixKYC+userID, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// SetNationalIDLink stores the HMAC-keyed dedup index: hmac(nationalID) → userID.
// The raw National ID is never stored.
func (d *DB) SetNationalIDLink(hmacKey, userID string) error {
	return d.set(prefixKYCNationalID+hmacKey, userID)
}

// NationalIDLinked returns (linked, linkedUserID, error).
// linked is true if the HMAC key already maps to a user.
func (d *DB) NationalIDLinked(hmacKey string) (bool, string, error) {
	var uid string
	err := d.get(prefixKYCNationalID+hmacKey, &uid)
	if err != nil {
		if IsNotFound(err) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, uid, nil
}

// SetTaxIDLink stores the HMAC-keyed dedup index for Tax ID.
func (d *DB) SetTaxIDLink(hmacKey, userID string) error {
	return d.set(prefixKYCTaxID+hmacKey, userID)
}

// TaxIDLinked returns (linked, linkedUserID, error).
func (d *DB) TaxIDLinked(hmacKey string) (bool, string, error) {
	var uid string
	err := d.get(prefixKYCTaxID+hmacKey, &uid)
	if err != nil {
		if IsNotFound(err) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, uid, nil
}

// ListKYCRecords returns all KYC records, optionally filtered by status (empty = all).
func (d *DB) ListKYCRecords(status string) ([]*model.KYCRecord, error) {
	var all []*model.KYCRecord
	err := d.iterPrefix(prefixKYC, func(val []byte) error {
		// Skip index entries (national_id: and tax_id: sub-prefixes).
		var r model.KYCRecord
		if err := jsonUnmarshal(val, &r); err != nil {
			return nil // skip non-record entries (index values are plain strings)
		}
		if r.UserID == "" {
			return nil
		}
		if status == "" || r.Status == status {
			all = append(all, &r)
		}
		return nil
	})
	return all, err
}

// KYCSummary returns aggregate KYC counts for this node.
func (d *DB) KYCSummary(nodeID string) (*model.KYCSummary, error) {
	summary := &model.KYCSummary{NodeID: nodeID}
	err := d.iterPrefix(prefixKYC, func(val []byte) error {
		var r model.KYCRecord
		if err := jsonUnmarshal(val, &r); err != nil || r.UserID == "" {
			return nil
		}
		summary.Total++
		switch r.Status {
		case model.KYCVerified:
			summary.Verified++
		case model.KYCPending:
			summary.Pending++
		case model.KYCRejected:
			summary.Rejected++
		default:
			summary.Unverified++
		}
		return nil
	})
	return summary, err
}

