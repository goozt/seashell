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

// SelectValidator returns the expected validator public key for a given block height
// using round-robin selection across the validator list.
func SelectValidator(height uint64, validators [][]byte) []byte {
	return validators[height%uint64(len(validators))]
}

// ValidateBlock verifies that the block was signed by the correct round-robin validator.
func ValidateBlock(block *Block, db *badger.DB) bool {
	validators := GetValidators(db)
	if len(validators) == 0 {
		return false
	}
	expected := SelectValidator(block.Height, validators)
	if !bytes.Equal(expected, block.Validator) {
		return false
	}
	if len(block.Signature) != 64 || len(block.Validator) != 64 {
		return false
	}
	r, s := decodeSig(block.Signature)
	x, y := decodePubKey(block.Validator)
	curve := elliptic.P256()
	pubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
	return ecdsa.Verify(&pubKey, block.Hash, &r, &s)
}

// ValidateBlockStandalone verifies a block's PoA signature against a given validator list
// (does not require a DB — used for P2P sync validation).
func ValidateBlockStandalone(block *Block, validators [][]byte) bool {
	if len(validators) == 0 {
		return false
	}
	expected := SelectValidator(block.Height, validators)
	if !bytes.Equal(expected, block.Validator) {
		return false
	}
	if len(block.Signature) != 64 || len(block.Validator) != 64 {
		return false
	}
	r, s := decodeSig(block.Signature)
	x, y := decodePubKey(block.Validator)
	curve := elliptic.P256()
	pubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
	return ecdsa.Verify(&pubKey, block.Hash, &r, &s)
}
