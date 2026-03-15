package store

import (
	"encoding/json"
	"fmt"
	"os"

	badger "github.com/dgraph-io/badger/v3"
)

// DB wraps a BadgerDB instance for the API data layer.
// It is stored at ./db/api (separate from the blockchain DB at ./db/blocks).
type DB struct {
	db  *badger.DB
	kek []byte // AES-256-GCM key encryption key; nil means no encryption (dev mode)
}

// Open opens or creates the API BadgerDB at the given path.
// kek is the key encryption key for private keys at rest; pass nil for dev mode (no encryption).
func Open(path string, kek []byte) (*DB, error) {
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	bdb, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}
	return &DB{db: bdb, kek: kek}, nil
}

// MigratePrivKeysToEncrypted encrypts any unencrypted private keys already stored in the DB.
// It is idempotent: guarded by a migration flag key so it only runs once per DB.
// Safe to call on every startup.
func (d *DB) MigratePrivKeysToEncrypted() error {
	if len(d.kek) == 0 {
		return nil // no-op in dev mode
	}
	const migrationFlag = "migration:privkey_encrypted"
	var flag string
	if err := d.get(migrationFlag, &flag); err == nil && flag == "1" {
		return nil // already migrated
	}

	// Collect all wallet_privkey: and authority_privkey: entries.
	type entry struct{ key string; val []byte }
	var entries []entry
	for _, prefix := range []string{"wallet_privkey:", "authority_privkey:"} {
		err := d.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.Prefix = []byte(prefix)
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := string(item.KeyCopy(nil))
				v, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				entries = append(entries, entry{k, v})
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan %s: %w", prefix, err)
		}
	}

	for _, e := range entries {
		// Try decrypting with kek — if it succeeds the value is already encrypted.
		if _, err := DecryptKey(d.kek, e.val); err == nil && len(e.val) > 12 {
			continue // already encrypted (has nonce prefix)
		}
		encrypted, err := EncryptKey(d.kek, e.val)
		if err != nil {
			return fmt.Errorf("encrypt %s: %w", e.key, err)
		}
		if err := d.setRaw(e.key, encrypted); err != nil {
			return fmt.Errorf("store %s: %w", e.key, err)
		}
	}

	return d.set(migrationFlag, "1")
}

// Close closes the underlying BadgerDB.
func (d *DB) Close() error {
	return d.db.Close()
}

// set marshals v as JSON and stores it under key.
func (d *DB) set(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// get retrieves the value at key and unmarshals it into v.
// Returns ErrKeyNotFound if the key does not exist.
func (d *DB) get(key string, v interface{}) error {
	return d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, v)
		})
	})
}

// del deletes the key. No error if the key does not exist.
func (d *DB) del(key string) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// setRaw stores raw bytes under key.
func (d *DB) setRaw(key string, data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// getRaw retrieves raw bytes stored at key.
func (d *DB) getRaw(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		result, err = item.ValueCopy(nil)
		return err
	})
	return result, err
}

// iterPrefix iterates all keys with the given prefix, calling fn for each value.
func (d *DB) iterPrefix(prefix string, fn func(val []byte) error) error {
	return d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			if err := item.Value(fn); err != nil {
				return err
			}
		}
		return nil
	})
}

// IsNotFound returns true if the error is a BadgerDB key-not-found error.
func IsNotFound(err error) bool {
	return err == badger.ErrKeyNotFound
}
