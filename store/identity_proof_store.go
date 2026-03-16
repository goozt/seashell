package store

import (
	"github.com/goozt/seashell/model"
)

const prefixIdentityProof = "identity_proof:"

// SaveIdentityProof persists an identity proof for a user.
func (d *DB) SaveIdentityProof(userID string, proof *model.IdentityProof) error {
	return d.set(prefixIdentityProof+userID, proof)
}

// GetIdentityProof retrieves the identity proof for a user.
// Returns IsNotFound if no proof has been issued yet.
func (d *DB) GetIdentityProof(userID string) (*model.IdentityProof, error) {
	var proof model.IdentityProof
	if err := d.get(prefixIdentityProof+userID, &proof); err != nil {
		return nil, err
	}
	return &proof, nil
}
