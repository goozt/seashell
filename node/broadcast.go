package node

import (
	"fmt"
	"log"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/model"
)

// BroadcastBlock sends block to every active peer concurrently.
// Errors are logged; the function does not return until all goroutines complete.
func BroadcastBlock(peers []model.Node, block *blockchain.Block, secret string) {
	if len(peers) == 0 {
		return
	}
	done := make(chan error, len(peers))
	for _, p := range peers {
		p := p
		go func() {
			client := NewPeerClient(p.NodeURL, secret)
			done <- client.BroadcastBlock(block)
		}()
	}
	for range peers {
		if err := <-done; err != nil {
			log.Printf("broadcast block: %v", err)
		}
	}
}

// BroadcastNewValidator notifies all active peers of a new validator public key.
func BroadcastNewValidator(peers []model.Node, pubKeyHex string, secret string) {
	if len(peers) == 0 {
		return
	}
	done := make(chan error, len(peers))
	for _, p := range peers {
		p := p
		go func() {
			client := NewPeerClient(p.NodeURL, secret)
			done <- client.NotifyNewValidator(pubKeyHex)
		}()
	}
	for range peers {
		if err := <-done; err != nil {
			log.Printf("broadcast validator: %v", err)
		}
	}
}

// BroadcastError collects non-nil errors from a slice returned by concurrent ops.
func BroadcastError(errs []error) error {
	var first error
	for _, e := range errs {
		if e != nil && first == nil {
			first = e
		}
	}
	return first
}

// PeerURLs extracts node URLs, excluding the local node URL.
func PeerURLs(nodes []model.Node, selfURL string) []model.Node {
	var peers []model.Node
	for _, n := range nodes {
		if n.NodeURL != selfURL && n.Status == model.NodeStatusActive {
			peers = append(peers, n)
		}
	}
	return peers
}

// ValidatorPubKeysFromPeers asks each peer for its validator list and returns
// the union.  Used when a new node joins and needs to bootstrap the validator set.
func ValidatorPubKeysFromPeers(peers []model.Node, secret string) ([]string, error) {
	for _, p := range peers {
		client := NewPeerClient(p.NodeURL, secret)
		keys, err := client.GetValidators()
		if err == nil && len(keys) > 0 {
			return keys, nil
		}
	}
	return nil, fmt.Errorf("could not obtain validator list from any peer")
}
