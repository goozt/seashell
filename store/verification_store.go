package store

import (
	"github.com/goozt/seashell/model"
)

const (
	prefixVerifyConfig = "verifyconfig:"
	prefixVerifySub    = "verifysub:"
	prefixVerifySubIdx = "idx:verifysub:user:"
)

// SaveVerificationConfig persists or updates the verification config for an authority.
func (d *DB) SaveVerificationConfig(cfg *model.VerificationConfig) error {
	return d.set(prefixVerifyConfig+cfg.AuthorityID, cfg)
}

// GetVerificationConfig retrieves the verification config for an authority.
func (d *DB) GetVerificationConfig(authorityID string) (*model.VerificationConfig, error) {
	var cfg model.VerificationConfig
	if err := d.get(prefixVerifyConfig+authorityID, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveVerificationSubmission persists a submission and updates the user+authority index.
func (d *DB) SaveVerificationSubmission(sub *model.VerificationSubmission) error {
	if err := d.set(prefixVerifySub+sub.ID, sub); err != nil {
		return err
	}
	// Update index: user+authority -> latest submission ID.
	return d.set(prefixVerifySubIdx+sub.UserID+":"+sub.AuthorityID, sub.ID)
}

// GetVerificationSubmission retrieves a submission by its UUID.
func (d *DB) GetVerificationSubmission(id string) (*model.VerificationSubmission, error) {
	var sub model.VerificationSubmission
	if err := d.get(prefixVerifySub+id, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetVerificationSubmissionByUser retrieves the latest submission for a user under an authority
// using the user+authority index. Returns IsNotFound if no submission exists.
func (d *DB) GetVerificationSubmissionByUser(userID, authorityID string) (*model.VerificationSubmission, error) {
	var subID string
	if err := d.get(prefixVerifySubIdx+userID+":"+authorityID, &subID); err != nil {
		return nil, err
	}
	return d.GetVerificationSubmission(subID)
}

// ListVerificationSubmissions returns all submissions for an authority, optionally filtered by status.
func (d *DB) ListVerificationSubmissions(authorityID, status string) ([]*model.VerificationSubmission, error) {
	var all []*model.VerificationSubmission
	err := d.iterPrefix(prefixVerifySub, func(val []byte) error {
		var sub model.VerificationSubmission
		if err := jsonUnmarshal(val, &sub); err != nil {
			return nil // skip non-record entries (index values are plain strings)
		}
		if sub.ID == "" || sub.AuthorityID == "" {
			return nil
		}
		if sub.AuthorityID != authorityID {
			return nil
		}
		if status != "" && sub.Status != status {
			return nil
		}
		all = append(all, &sub)
		return nil
	})
	return all, err
}
