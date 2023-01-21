package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	dbPath      = "./db/blocks"
	dbFile      = "./db/blocks/MANIFEST"
	genesisData = "Initial transaction from Genesis"
)

var lastHashByte = []byte("lh")

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func InitBlockChain(enableLog bool, address string) *BlockChain {

	if DbExists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	var lastHash []byte
	_ = os.MkdirAll(dbPath, 0700)
	opts := badger.DefaultOptions(dbPath)
	if !enableLog {
		opts.Logger = nil
	}
	db, err := badger.Open(opts)
	HandleFatalErrors(err)

	err = db.Update(func(txn *badger.Txn) error {
		sstx := CoinbaseTx(address, genesisData)
		gen := Genesis(sstx)
		fmt.Println("Genesis proved!")
		err = txn.Set(gen.Hash, gen.Serialize())
		HandleFatalErrors(err)
		err = txn.Set(lastHashByte, gen.Hash)
		lastHash = gen.Hash
		return err
	})

	HandleFatalErrors(err)

	return &BlockChain{lastHash, db}
}

func ContinueBlockChain(enableLog bool, address string) *BlockChain {

	if !DbExists() {
		fmt.Println("Blockchain does not exists")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	if !enableLog {
		opts.Logger = nil
	}
	db, err := badger.Open(opts)
	HandleFatalErrors(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(lastHashByte)
		HandleFatalErrors(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})

	HandleFatalErrors(err)

	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) Close() error {
	return chain.Database.Close()
}

func (chain *BlockChain) AddBlock(txs []*Transaction) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(lastHashByte)
		HandleFatalErrors(err)
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	HandleFatalErrors(err)

	newBlock := NewBlock(txs, lastHash)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		HandleFatalErrors(err)
		err = txn.Set(lastHashByte, newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
}

func (chain *BlockChain) FindUnspentTransactions(publicKeyHash []byte) []Transaction {
	var unspentTxs []Transaction

	spentTxOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOs[txId] != nil {
					for _, spendOut := range spentTxOs[txId] {
						if spendOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(publicKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
				}
				if !tx.IsCoinbase() {
					for _, in := range tx.Inputs {
						if in.UsesKey(publicKeyHash) {
							inTxId := hex.EncodeToString(in.Id)
							spentTxOs[inTxId] = append(spentTxOs[inTxId], in.Out)
						}
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

func (chain *BlockChain) FindUTXO(publicKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(publicKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(publicKeyHash) {
				UTXOs = append(UTXOs, out)
			}
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

	return Transaction{}, fmt.Errorf("transaction does not exists")
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
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.Id)
		HandleFatalErrors(err)

		prevTxs[hex.EncodeToString(in.Id)] = prevTx
	}

	return tx.Verify(prevTxs)
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
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

func DbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}
