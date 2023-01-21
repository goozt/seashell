package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"github.com/goozt/seashell/wallet"
)

type Transaction struct {
	Id      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	walletDB, err := wallet.CreateWalletDB()
	HandleFatalErrors(err)

	w := walletDB.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutput := chain.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		HandleFatalErrors(fmt.Errorf("error: not enough funds"))
	}

	for txid, outs := range validOutput {
		txId, err := hex.DecodeString(txid)
		HandleFatalErrors(err)

		for _, out := range outs {
			input := TxInput{txId, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTxOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, *NewTxOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.Id = tx.Hash()
	chain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Shells to %s", to)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTxOutput(100, to)

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.SetID()

	return &tx
}

func (tx Transaction) Serialize() []byte {
	var encoder bytes.Buffer

	enc := gob.NewEncoder(&encoder)
	err := enc.Encode(tx)
	HandleFatalErrors(err)

	return encoder.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.Id = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encoder := gob.NewEncoder(&encoded)
	err := encoder.Encode(tx)
	HandleFatalErrors(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.Id = hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].Id) == 0 && tx.Inputs[0].Out == -1
}

func (tx *Transaction) Sign(privateKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}
	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.Id)].Id == nil {
			log.Fatalln("Error: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.Id)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.Id = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, txCopy.Id)
		HandleFatalErrors(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inId].Signature = signature
	}

}

func (tx *Transaction) TrimmedCopy() *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.Id, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.Id, inputs, outputs}

	return &txCopy
}

func SplitBinary(data []byte) (a big.Int, b big.Int) {
	length := len(data)
	a.SetBytes(data[:(length / 2)])
	b.SetBytes(data[(length / 2):])
	return a, b
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.Id)].Id == nil {
			log.Fatalln("previous transaction does not exists")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.Id)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.Id = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r, s := SplitBinary(in.Signature)
		x, y := SplitBinary(in.PubKey)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, txCopy.Id, &r, &s) {
			return false
		}
	}

	return true
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("-- Transaction %x: ", tx.Id))
	for inId, in := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", inId))
		lines = append(lines, fmt.Sprintf("       TxID: %x", in.Id))
		lines = append(lines, fmt.Sprintf("       Out: %d", in.Out))
		lines = append(lines, fmt.Sprintf("       Sig: %x", in.Signature))
		lines = append(lines, fmt.Sprintf("       Pub: %x", in.PubKey))
	}

	for outId, out := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", outId))
		lines = append(lines, fmt.Sprintf("       Value: %d", out.Value))
		lines = append(lines, fmt.Sprintf("       PubHash: %x", out.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
