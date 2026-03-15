package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

const defaultQuorumTimeout = 10 * time.Second

// QuorumService collects co-signatures from peer validators for a newly created block.
// It fans out cosign requests concurrently and returns once quorum is reached or timeout fires.
type QuorumService struct {
	db      *store.DB
	cfg     *config.Config
	timeout time.Duration
}

// NewQuorumService creates a QuorumService with the given timeout (0 = default 10s).
func NewQuorumService(db *store.DB, cfg *config.Config, timeout time.Duration) *QuorumService {
	if timeout == 0 {
		timeout = defaultQuorumTimeout
	}
	return &QuorumService{db: db, cfg: cfg, timeout: timeout}
}

// PeerCoSigner is the interface QuorumService uses to contact peer nodes.
// The node package implements this with PeerClient.
type PeerCoSigner interface {
	RequestCoSign(ctx context.Context, nodeURL string, req model.CoSignRequest) (blockchain.ValidatorSig, error)
}

// CollectQuorum fans out cosign requests to all active peer nodes concurrently.
// It adds valid co-signatures to block.Signatures in place.
// Returns when quorum is reached or timeout fires (best-effort).
func (q *QuorumService) CollectQuorum(ctx context.Context, block *blockchain.Block, peers PeerCoSigner) error {
	nodes, err := q.db.ListActiveNodes()
	if err != nil {
		return fmt.Errorf("list active nodes: %w", err)
	}

	// Count total validators including self.
	totalValidators := len(nodes)
	if totalValidators <= 1 {
		return nil // single-node deployment; lead sig is sufficient
	}

	threshold := blockchain.QuorumThreshold(totalValidators)
	if len(block.Signatures) >= threshold {
		return nil
	}

	type result struct {
		sig blockchain.ValidatorSig
		err error
	}

	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()

	ch := make(chan result, len(nodes))
	var wg sync.WaitGroup

	req := model.CoSignRequest{
		BlockHash:     hex.EncodeToString(block.Hash),
		Height:        block.Height,
		LeadValidator: hex.EncodeToString(block.LeadValidator()),
	}

	for _, n := range nodes {
		if n.NodeURL == q.cfg.NodeURL {
			continue // skip self
		}
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()
			vs, err := peers.RequestCoSign(ctx, nodeURL, req)
			ch <- result{sig: vs, err: err}
		}(n.NodeURL)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	collected := len(block.Signatures)
	for r := range ch {
		if r.err != nil {
			log.Printf("quorum: cosign error: %v", r.err)
			continue
		}
		block.AddCoSig(r.sig.PubKey, r.sig.Sig)
		collected++
		if collected >= threshold {
			cancel()
		}
	}
	return nil
}

// SignBlock produces a ValidatorSig for blockHash using the authority private key
// associated with this node. authorityID identifies which key to load.
func (q *QuorumService) SignBlock(blockHash []byte, authorityID string) (*blockchain.ValidatorSig, error) {
	dBytes, err := q.db.GetAuthorityPrivKey(authorityID)
	if err != nil {
		return nil, fmt.Errorf("get authority privkey: %w", err)
	}
	priv := privKeyFromD(dBytes)

	pubKey := make([]byte, 64)
	xb := priv.PublicKey.X.Bytes()
	yb := priv.PublicKey.Y.Bytes()
	copy(pubKey[32-len(xb):32], xb)
	copy(pubKey[64-len(yb):64], yb)

	r, s, err := ecdsa.Sign(rand.Reader, &priv, blockHash)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	sig := encodeSig64(r, s)
	return &blockchain.ValidatorSig{PubKey: pubKey, Sig: sig}, nil
}

// encodeSig64 encodes (r, s) as fixed 64-byte big-endian zero-padded signature.
func encodeSig64(r, s *big.Int) []byte {
	sig := make([]byte, 64)
	rb := r.Bytes()
	sb := s.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	return sig
}
