package store

import (
	"fmt"

	"github.com/goozt/seashell/model"
)

const prefixAuthority = "authority:"

// SaveAuthority persists an authority record.
func (d *DB) SaveAuthority(a *model.Authority) error {
	return d.set(prefixAuthority+a.ID, a)
}

// GetAuthorityByID retrieves an authority by ID.
func (d *DB) GetAuthorityByID(id string) (*model.Authority, error) {
	var a model.Authority
	if err := d.get(prefixAuthority+id, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAuthorities returns all authority records.
func (d *DB) ListAuthorities() ([]*model.Authority, error) {
	var all []*model.Authority
	err := d.iterPrefix(prefixAuthority, func(val []byte) error {
		var a model.Authority
		if err := jsonUnmarshal(val, &a); err != nil {
			return err
		}
		all = append(all, &a)
		return nil
	})
	return all, err
}

// ListPendingAuthorities returns authorities with status=pending.
func (d *DB) ListPendingAuthorities() ([]*model.Authority, error) {
	all, err := d.ListAuthorities()
	if err != nil {
		return nil, err
	}
	var result []*model.Authority
	for _, a := range all {
		if a.Status == model.AuthorityStatusPending {
			result = append(result, a)
		}
	}
	return result, nil
}

// ListActiveAuthorities returns authorities with status=active.
func (d *DB) ListActiveAuthorities() ([]*model.Authority, error) {
	all, err := d.ListAuthorities()
	if err != nil {
		return nil, err
	}
	var result []*model.Authority
	for _, a := range all {
		if a.Status == model.AuthorityStatusActive {
			result = append(result, a)
		}
	}
	return result, nil
}

// StoreAuthorityPrivKey stores the raw D bytes of an authority validator private key.
// This key is NEVER returned in API responses.
func (d *DB) StoreAuthorityPrivKey(authorityID string, dBytes []byte) error {
	return d.setRaw(fmt.Sprintf("authority_privkey:%s", authorityID), dBytes)
}

// GetAuthorityPrivKey retrieves the raw D bytes of an authority validator private key.
func (d *DB) GetAuthorityPrivKey(authorityID string) ([]byte, error) {
	return d.getRaw(fmt.Sprintf("authority_privkey:%s", authorityID))
}
