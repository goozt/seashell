package node

import (
	"fmt"

	"github.com/goozt/seashell/blockchain"
)

const syncBatchSize = 100

// FullSync downloads the entire blockchain from peer and writes it into chain,
// starting from block height 1 (genesis is assumed to already exist on chain).
// The chain must be writable (caller holds the chain service mutex).
func FullSync(peer *PeerClient, chain *blockchain.BlockChain) error {
	return incrementalSync(peer, chain, 1)
}

// IncrementalSync fetches blocks from peer starting just after the local tip.
func IncrementalSync(peer *PeerClient, chain *blockchain.BlockChain) error {
	localHeight := chain.GetCurrentHeight()
	return incrementalSync(peer, chain, localHeight+1)
}

func incrementalSync(peer *PeerClient, chain *blockchain.BlockChain, fromHeight uint64) error {
	for {
		blocks, err := peer.GetBlocks(fromHeight, syncBatchSize)
		if err != nil {
			return fmt.Errorf("fetch blocks from %d: %w", fromHeight, err)
		}
		if len(blocks) == 0 {
			return nil // fully synced
		}
		for _, block := range blocks {
			if err := chain.AddExternalBlock(block); err != nil {
				return fmt.Errorf("add external block height %d: %w", block.Height, err)
			}
			fromHeight = block.Height + 1
		}
		if len(blocks) < syncBatchSize {
			return nil // no more blocks
		}
	}
}
