package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(ChainEvent{})
}

const (
	// EventTypeAuthorityRegistered is recorded when an authority is approved and
	// its validator key enters the network.
	EventTypeAuthorityRegistered = "authority_registered"

	// EventTypeUserVerified is recorded when a verified user creates a wallet,
	// binding their identity hash to a wallet address on-chain.
	EventTypeUserVerified = "user_verified"
)

// ChainEvent is an authority-signed governance record embedded in a block.
// Events are distinct from UTXO transactions — they carry no value but create
// an immutable, publicly verifiable audit trail of governance actions.
//
// Each event is individually signed by the authority's validator private key
// (the same key used to sign blocks). Signature covers all semantic fields,
// so events cannot be altered or replayed across authorities/heights.
type ChainEvent struct {
	Type          string `json:"type"`
	AuthorityID   string `json:"authority_id"`
	AuthorityName string `json:"authority_name"`
	Height        uint64 `json:"height"`

	// AuthorityRegistered fields (non-empty when Type == EventTypeAuthorityRegistered)
	ValidatorPubKey string `json:"validator_pub_key,omitempty"`

	// UserVerified fields (non-empty when Type == EventTypeUserVerified)
	UserIDHash    string `json:"user_id_hash,omitempty"`    // hex(SHA256(userID+":"+authorityID))
	WalletAddress string `json:"wallet_address,omitempty"` // sender's blockchain address

	// Cryptographic proof
	SignerPubKey []byte `json:"signer_pub_key"` // 64-byte ECDSA P256 pubkey
	Sig          []byte `json:"sig"`             // 64-byte r||s ECDSA signature
}

// signingPayload returns the deterministic bytes that are signed/verified.
// Height and AuthorityID are included to prevent cross-chain or replay attacks.
func (e *ChainEvent) signingPayload() []byte {
	unsigned := struct {
		Type            string `json:"type"`
		AuthorityID     string `json:"authority_id"`
		Height          uint64 `json:"height"`
		ValidatorPubKey string `json:"validator_pub_key,omitempty"`
		UserIDHash      string `json:"user_id_hash,omitempty"`
		WalletAddress   string `json:"wallet_address,omitempty"`
	}{
		Type:            e.Type,
		AuthorityID:     e.AuthorityID,
		Height:          e.Height,
		ValidatorPubKey: e.ValidatorPubKey,
		UserIDHash:      e.UserIDHash,
		WalletAddress:   e.WalletAddress,
	}
	b, _ := json.Marshal(unsigned)
	h := sha256.Sum256(b)
	return h[:]
}

// Sign signs the event with the given private key, populating Sig and SignerPubKey.
func (e *ChainEvent) Sign(privKey ecdsa.PrivateKey) error {
	payload := e.signingPayload()
	r, s, err := ecdsa.Sign(rand.Reader, &privKey, payload)
	if err != nil {
		return err
	}
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	s.FillBytes(sBytes)
	e.Sig = append(rBytes, sBytes...)
	e.SignerPubKey = append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	return nil
}

// Verify checks that the event's ECDSA signature is valid.
// It does NOT verify that SignerPubKey is in the registered validator set —
// callers should cross-check that separately.
func (e *ChainEvent) Verify() error {
	if len(e.Sig) != 64 || len(e.SignerPubKey) != 64 {
		return fmt.Errorf("invalid signature or pubkey length")
	}
	payload := e.signingPayload()
	r, s := decodeSig(e.Sig)
	x, y := decodePubKey(e.SignerPubKey)
	pubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: &x, Y: &y}
	if !ecdsa.Verify(&pubKey, payload, &r, &s) {
		return fmt.Errorf("event signature invalid")
	}
	return nil
}
