package model

// IdentityProof is an authority-signed credential that binds a verified user's
// wallet address to their identity on the blockchain. It is attached to every
// transaction sent by a verified user, enabling any node to cryptographically
// confirm the sender was KYC'd by their authority — directly from chain data.
//
// Signing input: SHA256(wallet_address + ":" + user_id_hash + ":" + authority_id)
// Signed with:   Authority's ECDSA P256 private key (same key used to sign blocks)
type IdentityProof struct {
	// WalletAddress is the sender's blockchain address. Must match the transaction sender.
	WalletAddress string `json:"wallet_address"`

	// UserIDHash is hex(SHA256(userID + ":" + authorityID)).
	// Allows auditors to confirm a specific user maps to this proof without
	// exposing the raw user ID on-chain.
	UserIDHash string `json:"user_id_hash"`

	// AuthorityID is the UUID of the authority that issued this proof.
	AuthorityID string `json:"authority_id"`

	// AuthorityName is a human-readable label for display purposes.
	AuthorityName string `json:"authority_name"`

	// AuthorityPubKey is the hex-encoded 64-byte ECDSA P256 public key of the
	// issuing authority. Verifiers confirm this key is in the registered validator set.
	AuthorityPubKey string `json:"authority_pub_key"`

	// Sig is the hex-encoded 64-byte ECDSA signature (r || s, 32 bytes each)
	// over SHA256(wallet_address + ":" + user_id_hash + ":" + authority_id).
	Sig string `json:"sig"`
}
