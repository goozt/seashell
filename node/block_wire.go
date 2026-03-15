package node

import (
	"encoding/hex"
	"fmt"

	"github.com/goozt/seashell/blockchain"
)

// SerializedBlock is the JSON-safe wire format for a Block.
// blockchain.Block contains []byte fields that must be hex-encoded for JSON.
type SerializedBlock struct {
	Hash         string              `json:"hash"`
	PrevHash     string              `json:"prev_hash"`
	Height       uint64              `json:"height"`
	Timestamp    uint                `json:"timestamp"`
	Validator    string              `json:"validator"`
	Signature    string              `json:"signature"`
	Transactions []SerializedTx      `json:"transactions"`
}

// SerializedTx is the JSON-safe wire format for a Transaction.
type SerializedTx struct {
	ID      string              `json:"id"`
	Inputs  []SerializedTxInput `json:"inputs"`
	Outputs []SerializedTxOutput `json:"outputs"`
}

// SerializedTxInput is the wire format for a TxInput.
type SerializedTxInput struct {
	ID        string `json:"id"`
	Out       int    `json:"out"`
	Signature string `json:"signature"`
	PubKey    string `json:"pub_key"`
}

// SerializedTxOutput is the wire format for a TxOutput.
type SerializedTxOutput struct {
	Value      int    `json:"value"`
	PubKeyHash string `json:"pub_key_hash"`
}

// encodeBlock converts a Block to its wire format.
func encodeBlock(b *blockchain.Block) *SerializedBlock {
	sb := &SerializedBlock{
		Hash:      hex.EncodeToString(b.Hash),
		PrevHash:  hex.EncodeToString(b.PrevHash),
		Height:    b.Height,
		Timestamp: b.Timestamp,
		Validator: hex.EncodeToString(b.Validator),
		Signature: hex.EncodeToString(b.Signature),
	}
	for _, tx := range b.Transactions {
		stx := SerializedTx{ID: hex.EncodeToString(tx.Id)}
		for _, in := range tx.Inputs {
			stx.Inputs = append(stx.Inputs, SerializedTxInput{
				ID:        hex.EncodeToString(in.Id),
				Out:       in.Out,
				Signature: hex.EncodeToString(in.Signature),
				PubKey:    hex.EncodeToString(in.PubKey),
			})
		}
		for _, out := range tx.Outputs {
			stx.Outputs = append(stx.Outputs, SerializedTxOutput{
				Value:      out.Value,
				PubKeyHash: hex.EncodeToString(out.PubKeyHash),
			})
		}
		sb.Transactions = append(sb.Transactions, stx)
	}
	return sb
}

// Decode converts the wire format back to a blockchain.Block.
func (sb *SerializedBlock) Decode() (*blockchain.Block, error) {
	hashB, err := hex.DecodeString(sb.Hash)
	if err != nil {
		return nil, fmt.Errorf("decode hash: %w", err)
	}
	prevHashB, err := hex.DecodeString(sb.PrevHash)
	if err != nil {
		return nil, fmt.Errorf("decode prevHash: %w", err)
	}
	validatorB, err := hex.DecodeString(sb.Validator)
	if err != nil {
		return nil, fmt.Errorf("decode validator: %w", err)
	}
	sigB, err := hex.DecodeString(sb.Signature)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	block := &blockchain.Block{
		Hash:      hashB,
		PrevHash:  prevHashB,
		Height:    sb.Height,
		Timestamp: sb.Timestamp,
		Validator: validatorB,
		Signature: sigB,
	}

	for _, stx := range sb.Transactions {
		txIDBytes, err := hex.DecodeString(stx.ID)
		if err != nil {
			return nil, fmt.Errorf("decode tx id: %w", err)
		}
		tx := &blockchain.Transaction{Id: txIDBytes}
		for _, sin := range stx.Inputs {
			inIDBytes, _ := hex.DecodeString(sin.ID)
			sigBytes, _ := hex.DecodeString(sin.Signature)
			pubBytes, _ := hex.DecodeString(sin.PubKey)
			tx.Inputs = append(tx.Inputs, blockchain.TxInput{
				Id:        inIDBytes,
				Out:       sin.Out,
				Signature: sigBytes,
				PubKey:    pubBytes,
			})
		}
		for _, sout := range stx.Outputs {
			pkh, _ := hex.DecodeString(sout.PubKeyHash)
			tx.Outputs = append(tx.Outputs, blockchain.TxOutput{
				Value:      sout.Value,
				PubKeyHash: pkh,
			})
		}
		block.Transactions = append(block.Transactions, tx)
	}

	return block, nil
}

// EncodeBlock is the exported wrapper for use by handlers.
func EncodeBlock(b *blockchain.Block) *SerializedBlock {
	return encodeBlock(b)
}
