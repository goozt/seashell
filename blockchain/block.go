package blockchain

import (
	"bytes"
	"encoding/gob"
	"time"

	"crypto/sha256"
)

type Block struct {
	Timestamp    uint
	PrevHash     []byte
	Transactions []*Transaction
	Hash         []byte
	Nonce        int
}

func Genesis(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func NewBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{
		Timestamp:    uint(time.Now().Unix()),
		PrevHash:     prevHash,
		Transactions: txs,
		Nonce:        0,
	}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash
	block.Nonce = nonce

	return block
}

func (b *Block) HashTransaction() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Id)
	}

	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer

	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	HandleFatalErrors(err)

	return result.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	HandleFatalErrors(err)

	return &block
}
