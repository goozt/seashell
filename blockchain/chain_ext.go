package blockchain

// GetCurrentHeight returns the current chain height stored in the DB.
func (chain *BlockChain) GetCurrentHeight() uint64 {
	return chain.getHeight()
}
