package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"time"
)

type Block struct {
	Timestamp    uint
	PrevHash     []byte
	Transactions []*Transaction
	Hash         []byte
	Height       uint64
	Validator    []byte
	Signature    []byte
}

func Genesis(coinbase *Transaction, pubKey []byte, privKey ecdsa.PrivateKey) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0, pubKey, privKey)
}

func NewBlock(txs []*Transaction, prevHash []byte, height uint64, pubKey []byte, privKey ecdsa.PrivateKey) *Block {
	block := &Block{
		Timestamp:    uint(time.Now().Unix()),
		PrevHash:     prevHash,
		Transactions: txs,
		Height:       height,
		Validator:    pubKey,
	}
	block.Hash = block.computeHash()

	r, s, err := ecdsa.Sign(rand.Reader, &privKey, block.Hash)
	HandleFatalErrors(err)
	block.Signature = append(r.Bytes(), s.Bytes()...)

	return block
}

func (b *Block) computeHash() []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, b.Height)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(b.Timestamp))
	data := bytes.Join([][]byte{b.PrevHash, b.HashTransaction(), tsBytes, heightBytes}, []byte{})
	hash := sha256.Sum256(data)
	return hash[:]
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
