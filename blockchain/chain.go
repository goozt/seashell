package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	defaultDBPath = "./db/blocks"
	genesisData   = "Initial transaction from Genesis"
)

var lastHashByte = []byte("lh")
var heightKey = []byte("bh")

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
	dbPath   string
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// DbExistsAt returns true when the BadgerDB MANIFEST exists at the given path.
func DbExistsAt(path string) bool {
	_, err := os.Stat(filepath.Join(path, "MANIFEST"))
	return !os.IsNotExist(err)
}

// DbExists checks the default path (backward compat for CLI).
func DbExists() bool {
	return DbExistsAt(defaultDBPath)
}

func openDB(path string, enableLog bool) (*badger.DB, error) {
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	opts := badger.DefaultOptions(path)
	if !enableLog {
		opts.Logger = nil
	}
	return badger.Open(opts)
}

// InitBlockChainAt initialises a new blockchain at the given path.
func InitBlockChainAt(path string, enableLog bool, address string, pubKey []byte, privKey ecdsa.PrivateKey) *BlockChain {
	if DbExistsAt(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	db, err := openDB(path, enableLog)
	HandleFatalErrors(err)

	AddValidatorToDB(db, pubKey)

	var lastHash []byte
	err = db.Update(func(txn *badger.Txn) error {
		sstx := CoinbaseTx(address, genesisData)
		gen := Genesis(sstx, pubKey, privKey)
		fmt.Println("Genesis created!")
		if err := txn.Set(gen.Hash, gen.Serialize()); err != nil {
			return err
		}
		if err := txn.Set(lastHashByte, gen.Hash); err != nil {
			return err
		}
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, 0)
		if err := txn.Set(heightKey, heightBytes); err != nil {
			return err
		}
		lastHash = gen.Hash
		return nil
	})
	HandleFatalErrors(err)

	return &BlockChain{lastHash, db, path}
}

// ContinueBlockChainAt opens an existing blockchain at the given path.
func ContinueBlockChainAt(path string, enableLog bool) *BlockChain {
	if !DbExistsAt(path) {
		fmt.Println("Blockchain does not exist")
		runtime.Goexit()
	}

	db, err := openDB(path, enableLog)
	HandleFatalErrors(err)

	var lastHash []byte
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(lastHashByte)
		HandleFatalErrors(err)
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	HandleFatalErrors(err)

	return &BlockChain{lastHash, db, path}
}

// InitBlockChain uses the default path (backward compat for CLI).
func InitBlockChain(enableLog bool, address string, pubKey []byte, privKey ecdsa.PrivateKey) *BlockChain {
	return InitBlockChainAt(defaultDBPath, enableLog, address, pubKey, privKey)
}

// ContinueBlockChain uses the default path (backward compat for CLI).
func ContinueBlockChain(enableLog bool, address string) *BlockChain {
	return ContinueBlockChainAt(defaultDBPath, enableLog)
}

func (chain *BlockChain) Close() error {
	return chain.Database.Close()
}

func (chain *BlockChain) getHeight() uint64 {
	var height uint64
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(heightKey)
		if err != nil {
			return err
		}
		data, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		height = binary.BigEndian.Uint64(data)
		return nil
	})
	HandleFatalErrors(err)
	return height
}

// AddBlock mines a new block containing txs and appends it to the chain.
// Caller must hold any concurrency lock before calling this.
func (chain *BlockChain) AddBlock(txs []*Transaction, pubKey []byte, privKey ecdsa.PrivateKey) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(lastHashByte)
		HandleFatalErrors(err)
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	HandleFatalErrors(err)

	newHeight := chain.getHeight() + 1
	newBlock := NewBlock(txs, lastHash, newHeight, pubKey, privKey)

	if !ValidateBlock(newBlock, chain.Database) {
		HandleFatalErrors(fmt.Errorf("block rejected: %x is not an authorized validator for height %d", pubKey, newHeight))
	}

	err = chain.Database.Update(func(txn *badger.Txn) error {
		if err := txn.Set(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		if err := txn.Set(lastHashByte, newBlock.Hash); err != nil {
			return err
		}
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, newHeight)
		chain.LastHash = newBlock.Hash
		return txn.Set(heightKey, heightBytes)
	})
	HandleFatalErrors(err)
}

// AddExternalBlock accepts a fully-formed, already-signed block received from a peer.
// It validates PoA signature, prevHash linkage, and block height before storing.
// Returns an error if validation fails; does not panic.
func (chain *BlockChain) AddExternalBlock(block *Block) error {
	if !bytes.Equal(block.PrevHash, chain.LastHash) {
		return fmt.Errorf("block prevHash mismatch: expected %x, got %x", chain.LastHash, block.PrevHash)
	}
	currentHeight := chain.getHeight()
	if block.Height != currentHeight+1 {
		return fmt.Errorf("block height mismatch: expected %d, got %d", currentHeight+1, block.Height)
	}
	if !ValidateBlock(block, chain.Database) {
		return fmt.Errorf("block PoA validation failed for height %d", block.Height)
	}
	expectedHash := block.computeHash()
	if !bytes.Equal(block.Hash, expectedHash) {
		return fmt.Errorf("block hash mismatch at height %d", block.Height)
	}

	return chain.Database.Update(func(txn *badger.Txn) error {
		if err := txn.Set(block.Hash, block.Serialize()); err != nil {
			return err
		}
		if err := txn.Set(lastHashByte, block.Hash); err != nil {
			return err
		}
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, block.Height)
		chain.LastHash = block.Hash
		return txn.Set(heightKey, heightBytes)
	})
}

func (chain *BlockChain) AddValidator(pubKey []byte) {
	AddValidatorToDB(chain.Database, pubKey)
}

// FindUnspentTransactions returns all transactions that have at least one output
// locked to publicKeyHash that has not been spent elsewhere in the chain.
//
// Fixed: previously a tx with multiple outputs for the same address was appended
// once per matching output, causing double-counting; now each tx appears at most once.
func (chain *BlockChain) FindUnspentTransactions(publicKeyHash []byte) []Transaction {
	var unspentTxs []Transaction
	spentTxOs := make(map[string][]int)
	seen := make(map[string]bool)

	iter := chain.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

			// Record all outputs spent by this tx's inputs.
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(publicKeyHash) {
						inTxId := hex.EncodeToString(in.Id)
						spentTxOs[inTxId] = append(spentTxOs[inTxId], in.Out)
					}
				}
			}

			if seen[txId] {
				continue
			}
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOs[txId] != nil {
					for _, spentOut := range spentTxOs[txId] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(publicKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
					seen[txId] = true
					break // one match is enough to include the tx
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

// FindUTXO returns all unspent outputs locked to publicKeyHash.
// Uses a two-pass approach identical to the reference implementation but
// avoids the duplicate-tx issue by iterating outputs directly.
func (chain *BlockChain) FindUTXO(publicKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	spentTxOs := make(map[string][]int)

	iter := chain.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(publicKeyHash) {
						inTxId := hex.EncodeToString(in.Id)
						spentTxOs[inTxId] = append(spentTxOs[inTxId], in.Out)
					}
				}
			}

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOs[txId] != nil {
					for _, spentOut := range spentTxOs[txId] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(publicKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(publicKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txId := hex.EncodeToString(tx.Id)

		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(publicKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txId] = append(unspentOuts[txId], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}

func (chain *BlockChain) FindTransaction(id []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.Id, id) {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, fmt.Errorf("transaction does not exist")
}

func (bc *BlockChain) SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.Id)
		HandleFatalErrors(err)
		prevTxs[hex.EncodeToString(in.Id)] = prevTx
	}

	tx.Sign(privateKey, prevTxs)
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.Id)
		if err != nil {
			return false
		}
		prevTxs[hex.EncodeToString(in.Id)] = prevTx
	}

	return tx.Verify(prevTxs)
}

// GetBlocksFromHeight returns up to limit blocks starting from fromHeight (inclusive),
// oldest-first.  Used for P2P incremental sync.
func (chain *BlockChain) GetBlocksFromHeight(fromHeight uint64, limit int) []*Block {
	var all []*Block
	iter := chain.Iterator()
	for {
		block := iter.Next()
		all = append(all, block)
		if len(block.PrevHash) == 0 {
			break
		}
	}
	// Reverse to oldest-first.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	var result []*Block
	for _, b := range all {
		if b.Height >= fromHeight {
			result = append(result, b)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		HandleFatalErrors(err)
		encodedBlock, err := item.ValueCopy(nil)
		block = Deserialize(encodedBlock)
		return err
	})
	HandleFatalErrors(err)

	iter.CurrentHash = block.PrevHash
	return block
}
