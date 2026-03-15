package service

import (
	cryptorand "crypto/rand"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/store"
	"github.com/goozt/seashell/wallet"
)

// TxResult summarises a completed transaction.
type TxResult struct {
	TxID        string `json:"tx_id"`
	BlockHash   string `json:"block_hash"`
	BlockHeight uint64 `json:"block_height"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      int    `json:"amount"`
}

// BlockSummary is a lightweight block representation for list endpoints.
type BlockSummary struct {
	Hash           string `json:"hash"`
	Height         uint64 `json:"height"`
	Timestamp      uint   `json:"timestamp"`
	LeadValidator  string `json:"lead_validator"`
	SignatureCount int    `json:"signature_count"`
	TxCount        int    `json:"tx_count"`
	ValidPoA       bool   `json:"valid_poa"`
}

// ChainService wraps blockchain operations with proper error handling (panics → errors).
// The mu mutex serialises all blockchain access within this process; it must also be
// held by callers that add externally-received blocks (e.g. P2P sync handler).
type ChainService struct {
	mu          sync.Mutex
	chainDBPath string
	db          *store.DB
	// onBlock is an optional callback invoked (in a goroutine) after a block is
	// successfully added locally.  Used by the node service to broadcast to peers.
	onBlock func(blockHash []byte)
}

// NewChainService creates a ChainService.
func NewChainService(chainDBPath string, db *store.DB) *ChainService {
	return &ChainService{chainDBPath: chainDBPath, db: db}
}

// ChainDBPath returns the filesystem path of the blockchain database.
func (cs *ChainService) ChainDBPath() string {
	return cs.chainDBPath
}

// SetOnBlock registers a callback invoked (in a new goroutine) after a block is
// locally added.  The callback receives the new block's hash.
func (cs *ChainService) SetOnBlock(fn func(blockHash []byte)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.onBlock = fn
}

// EnsureInitialized creates a genesis block if the blockchain DB does not exist yet.
// This should only be called on the primary node on first boot.
func (cs *ChainService) EnsureInitialized() error {
	if blockchain.DbExistsAt(cs.chainDBPath) {
		return nil
	}

	// Generate a one-time genesis keypair. The address receives the coinbase reward.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("generate genesis key: %w", err)
	}
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	pubKeyHash := wallet.PublicKeyHash(pubKey)
	versionedPayload := append([]byte{0x00}, pubKeyHash...)
	genesisAddr := string(wallet.Base58Encode(append(versionedPayload, wallet.Checksum(versionedPayload)...)))

	chain := blockchain.InitBlockChainAt(cs.chainDBPath, false, genesisAddr, pubKey, *privKey)
	chain.Close()
	log.Println("chain: genesis block created at", cs.chainDBPath)
	return nil
}

// openChain opens the existing blockchain at the configured path. Caller must close it.
func (cs *ChainService) openChain() (chain *blockchain.BlockChain, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("chain open error: %v", r)
		}
	}()
	chain = blockchain.ContinueBlockChainAt(cs.chainDBPath, false)
	return chain, nil
}

// AddExternalBlock adds a peer-received block to the chain under the service mutex.
// It validates PoA and chain linkage before storing.
func (cs *ChainService) AddExternalBlock(block *blockchain.Block) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	chain, err := cs.openChain()
	if err != nil {
		return err
	}
	defer chain.Close()

	return chain.AddExternalBlock(block)
}

// GetBalance returns the total UTXO balance for a blockchain address.
func (cs *ChainService) GetBalance(address string) (int, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !wallet.ValidateAddress(address) {
		return 0, fmt.Errorf("invalid address")
	}
	chain, err := cs.openChain()
	if err != nil {
		return 0, err
	}
	defer chain.Close()

	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	utxos := chain.FindUTXO(pubKeyHash)
	total := 0
	for _, out := range utxos {
		total += out.Value
	}
	return total, nil
}

// CreateAndSubmit builds, signs, and seals a transaction in a new block.
// senderPrivKey is the transaction sender's ECDSA key.
// The block validator is determined by round-robin from the registered validator list.
func (cs *ChainService) CreateAndSubmit(
	fromAddress, toAddress string,
	amount int,
	senderPrivKey ecdsa.PrivateKey,
	senderPubKey []byte,
) (*TxResult, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !wallet.ValidateAddress(fromAddress) {
		return nil, fmt.Errorf("invalid from address")
	}
	if !wallet.ValidateAddress(toAddress) {
		return nil, fmt.Errorf("invalid to address")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	chain, err := cs.openChain()
	if err != nil {
		return nil, err
	}
	defer chain.Close()

	// Build transaction without touching the wallet file.
	pubKeyHash := wallet.PublicKeyHash(senderPubKey)
	acc, validOutput := chain.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		return nil, fmt.Errorf("insufficient funds: have %d, need %d", acc, amount)
	}

	var inputs []blockchain.TxInput
	for txid, outs := range validOutput {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			return nil, fmt.Errorf("decode txid: %w", err)
		}
		for _, out := range outs {
			inputs = append(inputs, blockchain.TxInput{Id: txID, Out: out, Signature: nil, PubKey: senderPubKey})
		}
	}

	outputs := []blockchain.TxOutput{*blockchain.NewTxOutput(amount, toAddress)}
	if acc > amount {
		outputs = append(outputs, *blockchain.NewTxOutput(acc-amount, fromAddress))
	}

	tx := &blockchain.Transaction{Inputs: inputs, Outputs: outputs}
	tx.Id = tx.Hash()
	chain.SignTransaction(tx, senderPrivKey)

	// Determine which authority is the designated validator for the next block.
	validatorPubKey, validatorPrivKey, err := cs.getNextValidator(chain)
	if err != nil {
		return nil, fmt.Errorf("get validator: %w", err)
	}

	var addErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				addErr = fmt.Errorf("add block: %v", r)
			}
		}()
		chain.AddBlock([]*blockchain.Transaction{tx}, validatorPubKey, validatorPrivKey)
	}()
	if addErr != nil {
		return nil, addErr
	}

	newHash := make([]byte, len(chain.LastHash))
	copy(newHash, chain.LastHash)

	// Notify the node service to broadcast this block to peers (non-blocking).
	if cb := cs.onBlock; cb != nil {
		go cb(newHash)
	}

	return &TxResult{
		TxID:        hex.EncodeToString(tx.Id),
		BlockHash:   hex.EncodeToString(chain.LastHash),
		BlockHeight: chain.GetCurrentHeight(),
		From:        fromAddress,
		To:          toAddress,
		Amount:      amount,
	}, nil
}

// getNextValidator finds the authority whose turn it is to seal the next block.
func (cs *ChainService) getNextValidator(chain *blockchain.BlockChain) ([]byte, ecdsa.PrivateKey, error) {
	validators := blockchain.GetValidators(chain.Database)
	if len(validators) == 0 {
		return nil, ecdsa.PrivateKey{}, fmt.Errorf("no validators registered")
	}
	nextHeight := chain.GetCurrentHeight() + 1
	expectedPubKey := blockchain.SelectValidator(nextHeight, validators)

	// Find the authority that owns this public key.
	pubKeyHex := hex.EncodeToString(expectedPubKey)
	authorities, err := cs.db.ListActiveAuthorities()
	if err != nil {
		return nil, ecdsa.PrivateKey{}, fmt.Errorf("list authorities: %w", err)
	}
	for _, a := range authorities {
		if a.ValidatorPubKey == pubKeyHex {
			dBytes, err := cs.db.GetAuthorityPrivKey(a.ID)
			if err != nil {
				return nil, ecdsa.PrivateKey{}, fmt.Errorf("get authority privkey: %w", err)
			}
			privKey := privKeyFromD(dBytes)
			return expectedPubKey, privKey, nil
		}
	}
	return nil, ecdsa.PrivateKey{}, fmt.Errorf("no authority holds validator key %s", pubKeyHex)
}

// GetBlocks returns a paginated list of block summaries (most recent first).
func (cs *ChainService) GetBlocks(limit int) ([]*BlockSummary, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	chain, err := cs.openChain()
	if err != nil {
		return nil, err
	}
	defer chain.Close()

	var summaries []*BlockSummary
	iter := chain.Iterator()
	count := 0
	for {
		block := iter.Next()
		summaries = append(summaries, &BlockSummary{
			Hash:           hex.EncodeToString(block.Hash),
			Height:         block.Height,
			Timestamp:      block.Timestamp,
			LeadValidator:  hex.EncodeToString(block.LeadValidator()),
			SignatureCount: len(block.Signatures),
			TxCount:        len(block.Transactions),
			ValidPoA:       blockchain.ValidateBlock(block, chain.Database),
		})
		count++
		if len(block.PrevHash) == 0 || count >= limit {
			break
		}
	}
	return summaries, nil
}

// GetBlock returns the full block for a given hex hash string.
func (cs *ChainService) GetBlock(hashHex string) (*blockchain.Block, bool, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	chain, err := cs.openChain()
	if err != nil {
		return nil, false, err
	}
	defer chain.Close()

	iter := chain.Iterator()
	for {
		block := iter.Next()
		if hex.EncodeToString(block.Hash) == hashHex {
			valid := blockchain.ValidateBlock(block, chain.Database)
			return block, valid, nil
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return nil, false, nil
}

// CreateWalletForUser generates a new ECDSA keypair, stores the private key in the
// API DB, and returns the blockchain address.
func (cs *ChainService) CreateWalletForUser(userID string) (address string, pubKey []byte, err error) {
	w := wallet.NewWallet()
	addr := string(w.Address())

	dBytes := w.PrivateKey.D.Bytes()
	if err := cs.db.StoreWalletPrivKey(userID, dBytes); err != nil {
		return "", nil, fmt.Errorf("store wallet key: %w", err)
	}
	return addr, w.PublicKey, nil
}

// LoadUserPrivKey reconstructs the ecdsa.PrivateKey for a user from stored D bytes.
func (cs *ChainService) LoadUserPrivKey(userID string) (ecdsa.PrivateKey, []byte, error) {
	dBytes, err := cs.db.GetWalletPrivKey(userID)
	if err != nil {
		return ecdsa.PrivateKey{}, nil, fmt.Errorf("wallet key not found: %w", err)
	}
	priv := privKeyFromD(dBytes)
	pubKey := append(priv.PublicKey.X.Bytes(), priv.PublicKey.Y.Bytes()...)
	return priv, pubKey, nil
}

// privKeyFromD reconstructs an ecdsa.PrivateKey on P256 from the scalar D bytes.
func privKeyFromD(dBytes []byte) ecdsa.PrivateKey {
	curve := elliptic.P256()
	priv := new(ecdsa.PrivateKey)
	priv.D = new(big.Int).SetBytes(dBytes)
	priv.PublicKey.Curve = curve
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(dBytes)
	return *priv
}

// GetTransactionsForAddress returns all transactions involving a given address.
func (cs *ChainService) GetTransactionsForAddress(address string) ([]map[string]interface{}, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !wallet.ValidateAddress(address) {
		return nil, fmt.Errorf("invalid address")
	}

	chain, err := cs.openChain()
	if err != nil {
		return nil, err
	}
	defer chain.Close()

	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]

	var result []map[string]interface{}
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			involved := false
			for _, out := range tx.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					involved = true
					break
				}
			}
			if !involved {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
						involved = true
						break
					}
				}
			}
			if involved {
				result = append(result, map[string]interface{}{
					"tx_id":        hex.EncodeToString(tx.Id),
					"block_height": block.Height,
					"block_hash":   hex.EncodeToString(block.Hash),
					"is_coinbase":  tx.IsCoinbase(),
				})
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return result, nil
}
