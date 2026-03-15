package service

import (
	"encoding/hex"
	"time"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
	"github.com/goozt/seashell/wallet"
)

const velocityWindow = 50 // number of recent blocks to inspect

// ValueService calculates and records token prices per authority.
type ValueService struct {
	db *store.DB
}

// NewValueService creates a ValueService.
func NewValueService(db *store.DB) *ValueService {
	return &ValueService{db: db}
}

// RecalculateForAuthority recomputes the price for an authority and saves the record.
// It must be called after a new block is added (inside ChainService.CreateAndSubmit).
func (vs *ValueService) RecalculateForAuthority(authorityID string, chain *blockchain.BlockChain) error {
	authority, err := vs.db.GetAuthorityByID(authorityID)
	if err != nil || !authority.IsActive() {
		return nil // skip inactive or missing
	}

	members, err := vs.db.ListUsersByAuthorityID(authorityID)
	if err != nil {
		return err
	}

	// Build set of member wallet addresses.
	memberAddrs := make(map[string]bool, len(members))
	for _, m := range members {
		if m.WalletAddress != "" {
			memberAddrs[m.WalletAddress] = true
		}
	}
	if len(memberAddrs) == 0 {
		return nil
	}

	height := chain.GetCurrentHeight()
	txVolume := vs.transactionVolume(chain, memberAddrs, velocityWindow)
	supply := vs.circulatingSupply(chain, memberAddrs)

	var velocity float64
	if supply > 0 {
		velocity = float64(txVolume) / float64(supply)
	}
	price := authority.BasePrice * (1 + authority.SensitivityK*velocity)

	rec := &model.ValueRecord{
		AuthorityID:       authorityID,
		BlockHeight:       height,
		Price:             price,
		TransactionVolume: txVolume,
		CirculatingSupply: supply,
		Velocity:          velocity,
		RecordedAt:        time.Now(),
	}
	return vs.db.SaveValueRecord(rec)
}

// transactionVolume sums shells moved by authority members in the last n blocks.
func (vs *ValueService) transactionVolume(chain *blockchain.BlockChain, memberAddrs map[string]bool, n int) int {
	total := 0
	count := 0
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() {
				continue
			}
			for _, out := range tx.Outputs {
				// Compute address from PubKeyHash to check membership
				addr := pubKeyHashToAddressStr(out.PubKeyHash)
				if memberAddrs[addr] {
					total += out.Value
				}
			}
		}
		count++
		if len(block.PrevHash) == 0 || count >= n {
			break
		}
	}
	return total
}

// circulatingSupply sums all UTXO values owned by authority members.
func (vs *ValueService) circulatingSupply(chain *blockchain.BlockChain, memberAddrs map[string]bool) int {
	total := 0
	for addr := range memberAddrs {
		if !wallet.ValidateAddress(addr) {
			continue
		}
		pubKeyHash := wallet.Base58Decode([]byte(addr))
		pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
		utxos := chain.FindUTXO(pubKeyHash)
		for _, out := range utxos {
			total += out.Value
		}
	}
	return total
}

// pubKeyHashToAddressStr is a best-effort reverse lookup used only for volume counting.
// It reconstructs the address bytes and encodes them. If the hash is malformed it returns "".
func pubKeyHashToAddressStr(pubKeyHash []byte) string {
	if len(pubKeyHash) == 0 {
		return ""
	}
	return hex.EncodeToString(pubKeyHash)
}
