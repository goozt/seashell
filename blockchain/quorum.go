package blockchain

// ValidatorSig holds a single co-signature from one validator.
// PubKey is the 64-byte uncompressed ECDSA public key (X||Y).
// Sig is the 64-byte encoded ECDSA signature (r||s).
type ValidatorSig struct {
	PubKey []byte
	Sig    []byte
}

// SelectValidator returns the public key of the validator whose turn it is to
// lead block creation at the given height, using round-robin selection.
func SelectValidator(height uint64, validators [][]byte) []byte {
	if len(validators) == 0 {
		return nil
	}
	return validators[height%uint64(len(validators))]
}

// QuorumThreshold returns the minimum number of valid signatures required
// to accept a block given n registered validators.
// Uses ceiling of 2/3: ceil(2n/3) = (2n + 2) / 3 (integer arithmetic).
// For n=1 → 1, n=2 → 2, n=3 → 2, n=4 → 3, n=6 → 4.
func QuorumThreshold(n int) int {
	if n <= 0 {
		return 1
	}
	return (2*n + 2) / 3
}
