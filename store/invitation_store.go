package store

import "github.com/goozt/seashell/model"

const prefixInvitation = "invitation:"

// SaveInvitation persists an invitation.
func (d *DB) SaveInvitation(inv *model.Invitation) error {
	return d.set(prefixInvitation+inv.Code, inv)
}

// GetInvitationByCode retrieves an invitation by its code.
func (d *DB) GetInvitationByCode(code string) (*model.Invitation, error) {
	var inv model.Invitation
	if err := d.get(prefixInvitation+code, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListInvitationsByAuthority returns all invitations for the given authority.
func (d *DB) ListInvitationsByAuthority(authorityID string) ([]*model.Invitation, error) {
	var result []*model.Invitation
	err := d.iterPrefix(prefixInvitation, func(val []byte) error {
		var inv model.Invitation
		if err := jsonUnmarshal(val, &inv); err != nil {
			return err
		}
		if inv.AuthorityID == authorityID {
			result = append(result, &inv)
		}
		return nil
	})
	return result, err
}

// DeleteInvitation removes an invitation by code.
func (d *DB) DeleteInvitation(code string) error {
	return d.del(prefixInvitation + code)
}
