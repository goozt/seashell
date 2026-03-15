package store

import (
	"fmt"

	"github.com/goozt/seashell/model"
)

// SaveValueRecord persists a value record keyed by authority+height.
// It also updates the "latest" pointer for the authority.
func (d *DB) SaveValueRecord(v *model.ValueRecord) error {
	key := fmt.Sprintf("value:%s:%020d", v.AuthorityID, v.BlockHeight)
	if err := d.set(key, v); err != nil {
		return err
	}
	return d.set(fmt.Sprintf("value_latest:%s", v.AuthorityID), v)
}

// GetLatestValue returns the most recent value record for an authority.
func (d *DB) GetLatestValue(authorityID string) (*model.ValueRecord, error) {
	var v model.ValueRecord
	if err := d.get(fmt.Sprintf("value_latest:%s", authorityID), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListValueHistory returns all value records for an authority, most recent first.
func (d *DB) ListValueHistory(authorityID string) ([]*model.ValueRecord, error) {
	prefix := fmt.Sprintf("value:%s:", authorityID)
	var records []*model.ValueRecord
	err := d.iterPrefix(prefix, func(val []byte) error {
		var v model.ValueRecord
		if err := jsonUnmarshal(val, &v); err != nil {
			return err
		}
		records = append(records, &v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Reverse for most-recent-first
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	return records, nil
}
