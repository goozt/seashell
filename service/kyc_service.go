package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

var nationalIDPattern = regexp.MustCompile(`^\d{12}$`)

// IdPClient is the interface satisfied by real and mock identity provider clients.
type IdPClient interface {
	// VerifyNationalID verifies a 12-digit national ID and returns the verified name.
	// Returns an error if the ID is invalid or verification fails.
	VerifyNationalID(ctx context.Context, nationalID string) (name string, err error)
}

// MockIdPClient calls the mock-idp server at BaseURL for verification.
// Suitable for development and testing.
type MockIdPClient struct {
	BaseURL    string
	httpClient *http.Client
}

// NewMockIdPClient creates a MockIdPClient targeting baseURL.
func NewMockIdPClient(baseURL string) *MockIdPClient {
	return &MockIdPClient{BaseURL: baseURL, httpClient: &http.Client{Timeout: 5 * time.Second}}
}

// VerifyNationalID calls GET $BaseURL/verify?id=<nationalID>.
func (m *MockIdPClient) VerifyNationalID(ctx context.Context, nationalID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.BaseURL+"/verify?id="+nationalID, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("idp request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		return "", fmt.Errorf("invalid national ID")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("idp returned %d: %s", resp.StatusCode, b)
	}
	// Response body is just the verified name.
	name, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(name), nil
}

// KYCService handles National ID / KYC verification for users.
type KYCService struct {
	db      *store.DB
	idp     IdPClient
	nodeID  string
	hmacKey []byte // HMAC-SHA256 key for dedup indexes (never stored raw)
}

// NewKYCService creates a KYCService.
// hmacKey should be a 32-byte random secret consistent across restarts (e.g. derived from KEK).
func NewKYCService(db *store.DB, idp IdPClient, nodeID string, hmacKey []byte) *KYCService {
	return &KYCService{db: db, idp: idp, nodeID: nodeID, hmacKey: hmacKey}
}

// InitiateKYC validates a national ID, deduplicates via HMAC index, calls the IdP,
// and stores the masked record. Returns the resulting KYCRecord.
func (s *KYCService) InitiateKYC(ctx context.Context, userID, nationalID, taxID string) (*model.KYCRecord, error) {
	if !nationalIDPattern.MatchString(nationalID) {
		return nil, fmt.Errorf("national ID must be exactly 12 digits")
	}

	// Dedup check via HMAC-keyed index (raw ID never stored).
	idHMAC := s.hmacHex(nationalID)
	linked, linkedUID, err := s.db.NationalIDLinked(idHMAC)
	if err != nil {
		return nil, fmt.Errorf("dedup check: %w", err)
	}
	if linked && linkedUID != userID {
		return nil, fmt.Errorf("national ID is already linked to another account")
	}

	var taxIDMasked string
	if taxID != "" {
		taxHMAC := s.hmacHex(taxID)
		taxLinked, taxUID, err := s.db.TaxIDLinked(taxHMAC)
		if err != nil {
			return nil, fmt.Errorf("tax id dedup check: %w", err)
		}
		if taxLinked && taxUID != userID {
			return nil, fmt.Errorf("tax ID is already linked to another account")
		}
		taxIDMasked = maskID(taxID)
		if err := s.db.SetTaxIDLink(taxHMAC, userID); err != nil {
			return nil, fmt.Errorf("store tax id link: %w", err)
		}
	}

	// Call identity provider.
	verifiedName, idpErr := s.idp.VerifyNationalID(ctx, nationalID)

	now := time.Now()
	rec := &model.KYCRecord{
		UserID:           userID,
		NationalIDMasked: maskID(nationalID),
		TaxIDMasked:      taxIDMasked,
		NodeID:           s.nodeID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if idpErr != nil {
		rec.Status = model.KYCRejected
		rec.RejectedReason = idpErr.Error()
	} else {
		rec.Status = model.KYCVerified
		rec.VerifiedName = verifiedName
		rec.VerifiedAt = &now
		// Store dedup index only on success.
		if err := s.db.SetNationalIDLink(idHMAC, userID); err != nil {
			return nil, fmt.Errorf("store national id link: %w", err)
		}
	}

	if err := s.db.SaveKYCRecord(rec); err != nil {
		return nil, fmt.Errorf("save kyc record: %w", err)
	}

	// Update user KYC fields.
	user, err := s.db.GetUserByID(userID)
	if err == nil {
		user.KYCStatus = rec.Status
		user.NationalIDMasked = rec.NationalIDMasked
		user.TaxIDMasked = rec.TaxIDMasked
		user.VerifiedName = rec.VerifiedName
		user.KYCVerifiedAt = rec.VerifiedAt
		user.KYCRejectedReason = rec.RejectedReason
		user.UpdatedAt = now
		_ = s.db.SaveUser(user)
	}

	return rec, nil
}

// GetKYCStatus returns the KYC record for a user, or a default unverified record.
func (s *KYCService) GetKYCStatus(userID string) (*model.KYCRecord, error) {
	rec, err := s.db.GetKYCRecord(userID)
	if store.IsNotFound(err) {
		return &model.KYCRecord{UserID: userID, Status: model.KYCUnverified}, nil
	}
	return rec, err
}

// ValidateUserCanTransact returns nil if the user is KYC-verified, error otherwise.
func (s *KYCService) ValidateUserCanTransact(userID string) error {
	rec, err := s.GetKYCStatus(userID)
	if err != nil {
		return fmt.Errorf("kyc check: %w", err)
	}
	if rec.Status != model.KYCVerified {
		return fmt.Errorf("identity verification required before transacting (status: %s)", rec.Status)
	}
	return nil
}

// AdminRejectKYC marks a user's KYC as rejected with a given reason.
func (s *KYCService) AdminRejectKYC(userID, reason string) error {
	rec, err := s.GetKYCStatus(userID)
	if err != nil {
		return err
	}
	now := time.Now()
	rec.Status = model.KYCRejected
	rec.RejectedReason = reason
	rec.UpdatedAt = now
	if err := s.db.SaveKYCRecord(rec); err != nil {
		return err
	}
	user, err := s.db.GetUserByID(userID)
	if err == nil {
		user.KYCStatus = model.KYCRejected
		user.KYCRejectedReason = reason
		user.UpdatedAt = now
		_ = s.db.SaveUser(user)
	}
	return nil
}

// hmacHex returns the HMAC-SHA256 hex of value using the service's hmac key.
func (s *KYCService) hmacHex(value string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// maskID masks all but the last 4 characters of an ID string.
// e.g. "123456789012" → "XXXX-XXXX-9012"
func maskID(id string) string {
	if len(id) <= 4 {
		return id
	}
	last4 := id[len(id)-4:]
	switch len(id) {
	case 12:
		return "XXXX-XXXX-" + last4
	default:
		masked := ""
		for range len(id) - 4 {
			masked += "X"
		}
		return masked + last4
	}
}
