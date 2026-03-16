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

func init() {
	// Register ValidatorSig for GOB encoding (used inside Block.Signatures slice).
	gob.Register(ValidatorSig{})
}

type Block struct {
	Timestamp    uint
	PrevHash     []byte
	Transactions []*Transaction
	Events       []ChainEvent   // governance audit log (authority registrations, user verifications)
	Hash         []byte
	Height       uint64
	Signatures   []ValidatorSig // replaces Validator + Signature (M-of-N quorum)
}

func Genesis(coinbase *Transaction, pubKey []byte, privKey ecdsa.PrivateKey) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0, pubKey, privKey)
}

// NewBlock creates a new block signed by the lead validator (pubKey/privKey).
// The block starts with a single signature; co-signatures can be added via AddCoSig.
func NewBlock(txs []*Transaction, prevHash []byte, height uint64, pubKey []byte, privKey ecdsa.PrivateKey) *Block {
	block := &Block{
		Timestamp:    uint(time.Now().Unix()),
		PrevHash:     prevHash,
		Transactions: txs,
		Height:       height,
	}
	block.Hash = block.computeHash()

	r, s, err := ecdsa.Sign(rand.Reader, &privKey, block.Hash)
	HandleFatalErrors(err)
	sig := encodeSig(r, s)

	block.Signatures = []ValidatorSig{{PubKey: pubKey, Sig: sig}}
	return block
}

// AddCoSig appends a co-signature from another validator.
// Duplicate public keys are silently ignored.
func (b *Block) AddCoSig(pubKey, sig []byte) {
	for _, s := range b.Signatures {
		if bytes.Equal(s.PubKey, pubKey) {
			return
		}
	}
	b.Signatures = append(b.Signatures, ValidatorSig{PubKey: pubKey, Sig: sig})
}

// LeadValidator returns the public key of the first (lead) validator signature.
// Returns nil if the block has no signatures.
func (b *Block) LeadValidator() []byte {
	if len(b.Signatures) == 0 {
		return nil
	}
	return b.Signatures[0].PubKey
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
