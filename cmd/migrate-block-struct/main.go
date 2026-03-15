// cmd/migrate-block-struct migrates a blockchain database from the old Block
// format (Validator []byte + Signature []byte) to the new format
// (Signatures []ValidatorSig). Run this once before upgrading the server binary.
//
// Usage:
//
//	go run ./cmd/migrate-block-struct -db ./db/blocks
package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/goozt/seashell/blockchain"
)

// blockV1 is the old Block struct layout before quorum signing.
// Used only for reading legacy data during migration.
type blockV1 struct {
	Timestamp    uint
	PrevHash     []byte
	Transactions []*blockchain.Transaction
	Hash         []byte
	Height       uint64
	Validator    []byte
	Signature    []byte
}

func init() {
	gob.Register(elliptic.P256())
}

func main() {
	dbPath := flag.String("db", "./db/blocks", "Path to the blockchain BadgerDB directory")
	flag.Parse()

	if _, err := os.Stat(filepath.Join(*dbPath, "MANIFEST")); os.IsNotExist(err) {
		log.Fatalf("No blockchain database found at %s", *dbPath)
	}

	opts := badger.DefaultOptions(*dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var lastHashKey = []byte("lh")
	var heightKey = []byte("bh")

	// Collect all block hashes (keys that are 32 bytes).
	var blockKeys [][]byte
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().KeyCopy(nil)
			// Block hashes are 32-byte keys; skip control keys.
			if len(k) == 32 {
				blockKeys = append(blockKeys, k)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("scan keys: %v", err)
	}

	log.Printf("Found %d block(s) to migrate", len(blockKeys))
	migrated := 0
	skipped := 0

	for _, key := range blockKeys {
		var rawData []byte
		err := db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			rawData, err = item.ValueCopy(nil)
			return err
		})
		if err != nil {
			log.Printf("skip key %x: %v", key, err)
			skipped++
			continue
		}

		// Try decoding as new format first.
		var newBlock blockchain.Block
		if err := gob.NewDecoder(bytes.NewReader(rawData)).Decode(&newBlock); err == nil && len(newBlock.Signatures) > 0 {
			skipped++
			continue // already migrated
		}

		// Decode as old format.
		var old blockV1
		if err := gob.NewDecoder(bytes.NewReader(rawData)).Decode(&old); err != nil {
			log.Printf("skip %x: cannot decode as V1: %v", key, err)
			skipped++
			continue
		}

		// Build new Block wrapping the single old sig as Signatures[0].
		newB := &blockchain.Block{
			Timestamp:    old.Timestamp,
			PrevHash:     old.PrevHash,
			Transactions: old.Transactions,
			Hash:         old.Hash,
			Height:       old.Height,
		}
		if len(old.Validator) > 0 && len(old.Signature) > 0 {
			newB.Signatures = []blockchain.ValidatorSig{
				{PubKey: old.Validator, Sig: old.Signature},
			}
		}

		// Write back.
		newData := newB.Serialize()
		err = db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, newData)
		})
		if err != nil {
			log.Fatalf("write block %x: %v", key, err)
		}
		migrated++
	}

	// Rebuild height index (idx:height:<N> → hash) for all blocks.
	log.Printf("Rebuilding height index…")
	err = db.View(func(txn *badger.Txn) error {
		for _, key := range blockKeys {
			item, err := txn.Get(key)
			if err != nil {
				continue
			}
			data, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var b blockchain.Block
			if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&b); err != nil {
				continue
			}
			hBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(hBytes, b.Height)
			idxKey := append([]byte("idx:height:"), hBytes...)
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set(idxKey, b.Hash)
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("height index rebuild warning: %v", err)
	}

	// Suppress unused variable warnings for control keys.
	_ = lastHashKey
	_ = heightKey

	fmt.Printf("Migration complete: %d migrated, %d skipped\n", migrated, skipped)
}
