package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

// VerificationService handles the modular user verification system.
type VerificationService struct {
	db *store.DB
}

// NewVerificationService creates a VerificationService.
func NewVerificationService(db *store.DB) *VerificationService {
	return &VerificationService{db: db}
}

// SaveConfig validates and persists a verification config for an authority.
func (s *VerificationService) SaveConfig(authorityID string, req *model.SaveVerificationConfigRequest) (*model.VerificationConfig, error) {
	if msg := req.Validate(); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}

	now := time.Now()

	// Check if config already exists (update vs create).
	existing, err := s.db.GetVerificationConfig(authorityID)
	if err != nil && !store.IsNotFound(err) {
		return nil, fmt.Errorf("check existing config: %w", err)
	}

	cfg := &model.VerificationConfig{
		AuthorityID: authorityID,
		Method:      req.Method,
		Fields:      req.Fields,
		Enabled:     true,
		UpdatedAt:   now,
	}

	if existing != nil {
		cfg.CreatedAt = existing.CreatedAt
	} else {
		cfg.CreatedAt = now
	}

	if err := s.db.SaveVerificationConfig(cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}

// GetConfig returns the verification config for an authority, or nil if not found.
func (s *VerificationService) GetConfig(authorityID string) (*model.VerificationConfig, error) {
	return s.db.GetVerificationConfig(authorityID)
}

// Submit validates field values against the config and creates a pending submission.
func (s *VerificationService) Submit(userID, authorityID string, req *model.SubmitVerificationRequest) (*model.VerificationSubmission, error) {
	// Fetch config.
	cfg, err := s.db.GetVerificationConfig(authorityID)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, fmt.Errorf("verification not configured for this authority")
		}
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("verification is currently disabled for this authority")
	}

	// Check existing submission.
	existing, err := s.db.GetVerificationSubmissionByUser(userID, authorityID)
	if err != nil && !store.IsNotFound(err) {
		return nil, fmt.Errorf("check existing submission: %w", err)
	}
	if existing != nil {
		switch existing.Status {
		case model.VerifyStatusPending:
			return nil, fmt.Errorf("a verification submission is already pending review")
		case model.VerifyStatusApproved:
			return nil, fmt.Errorf("you are already verified")
		// VerifyStatusRejected: allow resubmission
		}
	}

	// Validate field values against config fields.
	if err := validateFieldValues(cfg.Fields, req.FieldValues); err != nil {
		return nil, err
	}

	now := time.Now()
	sub := &model.VerificationSubmission{
		ID:          uuid.New().String(),
		AuthorityID: authorityID,
		UserID:      userID,
		Method:      cfg.Method,
		FieldValues: req.FieldValues,
		Status:      model.VerifyStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.SaveVerificationSubmission(sub); err != nil {
		return nil, fmt.Errorf("save submission: %w", err)
	}
	return sub, nil
}

// Review approves or rejects a pending submission.
func (s *VerificationService) Review(submissionID, reviewerID string, req *model.ReviewVerificationRequest) (*model.VerificationSubmission, error) {
	if msg := req.Validate(); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}

	sub, err := s.db.GetVerificationSubmission(submissionID)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, fmt.Errorf("submission not found")
		}
		return nil, fmt.Errorf("load submission: %w", err)
	}

	if sub.Status != model.VerifyStatusPending {
		return nil, fmt.Errorf("submission is not pending (current status: %s)", sub.Status)
	}

	now := time.Now()
	if req.Action == "approve" {
		sub.Status = model.VerifyStatusApproved
	} else {
		sub.Status = model.VerifyStatusRejected
	}
	sub.ReviewedBy = reviewerID
	sub.ReviewedAt = &now
	sub.Remarks = req.Remarks
	sub.UpdatedAt = now

	if err := s.db.SaveVerificationSubmission(sub); err != nil {
		return nil, fmt.Errorf("save reviewed submission: %w", err)
	}
	return sub, nil
}

// GetUserStatus returns the latest submission for a user under their authority.
// Returns nil (not an error) if no submission exists.
func (s *VerificationService) GetUserStatus(userID, authorityID string) (*model.VerificationSubmission, error) {
	sub, err := s.db.GetVerificationSubmissionByUser(userID, authorityID)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return sub, nil
}

// IssueIdentityProof creates and stores an authority-signed identity proof that
// binds walletAddress to the verified user. Called after wallet creation.
// The proof is signed with the authority's validator private key.
func (s *VerificationService) IssueIdentityProof(userID, authorityID, walletAddress string) (*model.IdentityProof, error) {
	authority, err := s.db.GetAuthorityByID(authorityID)
	if err != nil {
		return nil, fmt.Errorf("load authority: %w", err)
	}

	dBytes, err := s.db.GetAuthorityPrivKey(authorityID)
	if err != nil {
		return nil, fmt.Errorf("load authority key: %w", err)
	}

	// Reconstruct ECDSA private key from D scalar bytes.
	curve := elliptic.P256()
	priv := new(ecdsa.PrivateKey)
	priv.D = new(big.Int).SetBytes(dBytes)
	priv.PublicKey.Curve = curve
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(dBytes)

	// UserIDHash: hex(SHA256(userID + ":" + authorityID)) — privacy-preserving on-chain identity.
	rawHash := sha256.Sum256([]byte(userID + ":" + authorityID))
	userIDHash := hex.EncodeToString(rawHash[:])

	// Sign: SHA256(walletAddress + ":" + userIDHash + ":" + authorityID)
	sigInput := sha256.Sum256([]byte(walletAddress + ":" + userIDHash + ":" + authorityID))
	r, sVal, err := ecdsa.Sign(rand.Reader, priv, sigInput[:])
	if err != nil {
		return nil, fmt.Errorf("sign identity proof: %w", err)
	}

	// Fixed-width 64-byte encoding (32 bytes r || 32 bytes s), zero-padded big-endian.
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	sVal.FillBytes(sBytes)
	sig := hex.EncodeToString(append(rBytes, sBytes...))

	pubKeyHex := authority.ValidatorPubKey

	proof := &model.IdentityProof{
		WalletAddress:   walletAddress,
		UserIDHash:      userIDHash,
		AuthorityID:     authorityID,
		AuthorityName:   authority.Name,
		AuthorityPubKey: pubKeyHex,
		Sig:             sig,
	}

	if err := s.db.SaveIdentityProof(userID, proof); err != nil {
		return nil, fmt.Errorf("save identity proof: %w", err)
	}
	return proof, nil
}

// VerifyIdentityProof checks that an IdentityProof's signature is valid.
// Returns nil if the proof is valid.
func VerifyIdentityProof(proof *model.IdentityProof) error {
	pubKeyBytes, err := hex.DecodeString(proof.AuthorityPubKey)
	if err != nil || len(pubKeyBytes) != 64 {
		return fmt.Errorf("invalid authority public key")
	}

	sigBytes, err := hex.DecodeString(proof.Sig)
	if err != nil || len(sigBytes) != 64 {
		return fmt.Errorf("invalid signature encoding")
	}

	curve := elliptic.P256()
	x := new(big.Int).SetBytes(pubKeyBytes[:32])
	y := new(big.Int).SetBytes(pubKeyBytes[32:])
	pubKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	sigInput := sha256.Sum256([]byte(proof.WalletAddress + ":" + proof.UserIDHash + ":" + proof.AuthorityID))
	if !ecdsa.Verify(&pubKey, sigInput[:], r, s) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// IsUserVerified returns true if the user has an approved submission for the given authority.
func (s *VerificationService) IsUserVerified(userID, authorityID string) (bool, error) {
	sub, err := s.GetUserStatus(userID, authorityID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, nil
	}
	return sub.Status == model.VerifyStatusApproved, nil
}

// validateFieldValues checks each submitted value against the config field definitions.
func validateFieldValues(fields []model.FieldDefinition, values map[string]string) error {
	for _, f := range fields {
		val, provided := values[f.Name]

		if f.Required && (!provided || val == "") {
			return fmt.Errorf("field '%s' is required", f.Label)
		}

		if !provided || val == "" {
			continue
		}

		if f.MinLength > 0 && len(val) < f.MinLength {
			return fmt.Errorf("field '%s' must be at least %d characters", f.Label, f.MinLength)
		}

		if f.MaxLength > 0 && len(val) > f.MaxLength {
			return fmt.Errorf("field '%s' must be at most %d characters", f.Label, f.MaxLength)
		}

		if f.Pattern != "" {
			matched, err := regexp.MatchString(f.Pattern, val)
			if err != nil {
				return fmt.Errorf("invalid pattern for field '%s': %w", f.Label, err)
			}
			if !matched {
				return fmt.Errorf("field '%s' does not match the required format", f.Label)
			}
		}

		if f.InputType == "select" && len(f.Options) > 0 {
			valid := false
			for _, opt := range f.Options {
				if val == opt {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field '%s' has an invalid selection", f.Label)
			}
		}
	}
	return nil
}
