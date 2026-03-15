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
	db *badger.DB
}

// Open opens or creates the API BadgerDB at the given path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	bdb, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}
	return &DB{db: bdb}, nil
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
