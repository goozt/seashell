package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"

	badger "github.com/dgraph-io/badger/v3"
)

var validatorsKey = []byte("va")

// GetValidators loads the list of authorized validator public keys from the DB.
func GetValidators(db *badger.DB) [][]byte {
	var validators [][]byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(validatorsKey)
		if err != nil {
			return err
		}
		data, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return gob.NewDecoder(bytes.NewReader(data)).Decode(&validators)
	})
	if err != nil {
		return nil
	}
	return validators
}

// AddValidatorToDB appends a public key to the validators list stored in the DB.
// Duplicate public keys are silently ignored.
func AddValidatorToDB(db *badger.DB, pubKey []byte) {
	validators := GetValidators(db)
	for _, v := range validators {
		if bytes.Equal(v, pubKey) {
			return
		}
	}
	validators = append(validators, pubKey)

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(validators)
	HandleFatalErrors(err)

	encoded := buf.Bytes()
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(validatorsKey, encoded)
	})
	HandleFatalErrors(err)
}

// countValidSigs counts how many of the block's signatures come from registered validators
// and are cryptographically valid over block.Hash. Duplicate signer pubkeys are counted once.
func countValidSigs(block *Block, validators [][]byte) int {
	curve := elliptic.P256()
	seen := make(map[string]bool)
	count := 0
	for _, vs := range block.Signatures {
		if len(vs.PubKey) != 64 || len(vs.Sig) != 64 {
			continue
		}
		// Must be a registered validator.
		registered := false
		for _, v := range validators {
			if bytes.Equal(v, vs.PubKey) {
				registered = true
				break
			}
		}
		if !registered {
			continue
		}
		// Deduplicate.
		key := string(vs.PubKey)
		if seen[key] {
			continue
		}
		seen[key] = true
		// Verify signature.
		r, s := decodeSig(vs.Sig)
		x, y := decodePubKey(vs.PubKey)
		pubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&pubKey, block.Hash, &r, &s) {
			count++
		}
	}
	return count
}

// ValidateBlock verifies the block has a quorum of valid signatures from registered validators.
func ValidateBlock(block *Block, db *badger.DB) bool {
	// Genesis block predates the validator set and is always considered valid.
	if block.Height == 0 {
		return true
	}
	validators := GetValidators(db)
	if len(validators) == 0 {
		return false
	}
	return countValidSigs(block, validators) >= QuorumThreshold(len(validators))
}

// ValidateBlockStandalone verifies a block's quorum against a given validator list
// (does not require a DB — used for P2P sync validation).
func ValidateBlockStandalone(block *Block, validators [][]byte) bool {
	if len(validators) == 0 {
		return false
	}
	return countValidSigs(block, validators) >= QuorumThreshold(len(validators))
}
