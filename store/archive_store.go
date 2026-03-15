package store

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/goozt/seashell/blockchain"
)

// ArchiveDB manages gzip-compressed GOB block archives and a BadgerDB height index.
// Files are stored at: $basePath/YYYY/MM/<height>.blk.gz
// Index BadgerDB at:   $basePath/index/
type ArchiveDB struct {
	basePath string
	index    *badger.DB
}

// ArchiveMeta stores high-level statistics about the archive.
type ArchiveMeta struct {
	OldestHeight uint64    `json:"oldest_height"`
	NewestHeight uint64    `json:"newest_height"`
	TotalBlocks  int       `json:"total_blocks"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OpenArchive opens or creates the archive store at basePath.
func OpenArchive(basePath string) (*ArchiveDB, error) {
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("create archive dir: %w", err)
	}
	indexPath := filepath.Join(basePath, "index")
	opts := badger.DefaultOptions(indexPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open archive index: %w", err)
	}
	return &ArchiveDB{basePath: basePath, index: db}, nil
}

// Close closes the index database.
func (a *ArchiveDB) Close() error {
	return a.index.Close()
}

// WriteBlock archives a block as a gzip-compressed GOB file and updates the index.
func (a *ArchiveDB) WriteBlock(block *blockchain.Block) error {
	ts := time.Unix(int64(block.Timestamp), 0).UTC()
	dir := filepath.Join(a.basePath, fmt.Sprintf("%04d/%02d", ts.Year(), ts.Month()))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create archive subdir: %w", err)
	}

	relPath := fmt.Sprintf("%04d/%02d/%d.blk.gz", ts.Year(), ts.Month(), block.Height)
	absPath := filepath.Join(a.basePath, relPath)

	f, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	if err := gob.NewEncoder(gz).Encode(block); err != nil {
		return fmt.Errorf("encode block: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}

	// Update index.
	heightKey := fmt.Sprintf("arc:height:%020d", block.Height)
	if err := a.index.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(heightKey), []byte(relPath))
	}); err != nil {
		return fmt.Errorf("update index: %w", err)
	}

	return a.updateMeta(block.Height)
}

// ReadBlock retrieves an archived block by height.
func (a *ArchiveDB) ReadBlock(height uint64) (*blockchain.Block, error) {
	heightKey := fmt.Sprintf("arc:height:%020d", height)
	var relPath string
	err := a.index.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(heightKey))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		relPath = string(val)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("index lookup height %d: %w", height, err)
	}
	return a.readFile(filepath.Join(a.basePath, relPath))
}

// ReadBlockRange retrieves archived blocks from fromHeight to toHeight (inclusive).
func (a *ArchiveDB) ReadBlockRange(from, to uint64) ([]*blockchain.Block, error) {
	var blocks []*blockchain.Block
	for h := from; h <= to; h++ {
		b, err := a.ReadBlock(h)
		if err != nil {
			continue // skip missing heights
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// GetMeta returns archive statistics.
func (a *ArchiveDB) GetMeta() (*ArchiveMeta, error) {
	var meta ArchiveMeta
	err := a.index.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("arc:meta"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	if IsNotFound(err) {
		return &ArchiveMeta{}, nil
	}
	return &meta, err
}

// readFile decompresses a .blk.gz file and decodes the Block.
func (a *ArchiveDB) readFile(path string) (*blockchain.Block, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open archive file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	var block blockchain.Block
	if err := gob.NewDecoder(gz).Decode(&block); err != nil {
		return nil, fmt.Errorf("decode block: %w", err)
	}
	return &block, nil
}

// updateMeta updates the arc:meta index entry after archiving a block.
func (a *ArchiveDB) updateMeta(height uint64) error {
	meta, _ := a.GetMeta()
	if meta == nil {
		meta = &ArchiveMeta{}
	}
	meta.TotalBlocks++
	if meta.OldestHeight == 0 || height < meta.OldestHeight {
		meta.OldestHeight = height
	}
	if height > meta.NewestHeight {
		meta.NewestHeight = height
	}
	meta.UpdatedAt = time.Now()

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return a.index.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("arc:meta"), data)
	})
}

// ArchiveCutoff returns the Unix timestamp before which blocks should be archived.
func ArchiveCutoff(retentionYears int) int64 {
	return time.Now().AddDate(-retentionYears, 0, 0).Unix()
}

// keep gob aware of ValidatorSig
func init() {
	gob.Register(blockchain.ValidatorSig{})
	_ = bytes.NewBuffer // suppress unused import if needed
}
