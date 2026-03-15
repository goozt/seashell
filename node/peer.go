// Package node provides peer-to-peer communication between SeaShell nodes.
// Communication is HTTP-based: each node exposes a /p2p/v1/* API protected
// by a shared secret passed in the X-Node-Secret header.
package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/model"
)

const p2pTimeout = 10 * time.Second

// PeerClient is an HTTP client for communicating with a single remote node.
type PeerClient struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// NewPeerClient creates a PeerClient targeting baseURL.
func NewPeerClient(baseURL, secret string) *PeerClient {
	return &PeerClient{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: p2pTimeout,
		},
	}
}

func (c *PeerClient) do(method, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Secret", c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("peer returned %d: %s", resp.StatusCode, string(b))
	}

	if out != nil {
		// All SeaShell API responses are wrapped in {success, data, error}.
		// Unwrap the envelope before decoding into the caller-provided struct.
		var envelope struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
			Error   string          `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !envelope.Success {
			return fmt.Errorf("peer error: %s", envelope.Error)
		}
		if len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}
	}
	return nil
}

// Ping sends a health-check to the peer and returns its response.
func (c *PeerClient) Ping(nodeID string, height uint64) (*model.PingResponse, error) {
	req := model.PingRequest{NodeID: nodeID, BlockHeight: height}
	var resp model.PingResponse
	if err := c.do("POST", "/p2p/v1/ping", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPeers returns the active peer list from the remote node.
func (c *PeerClient) GetPeers() ([]model.Node, error) {
	var out struct {
		Peers []model.Node `json:"peers"`
	}
	if err := c.do("GET", "/p2p/v1/peers", nil, &out); err != nil {
		return nil, err
	}
	return out.Peers, nil
}

// GetBlocks fetches up to limit serialised blocks starting from fromHeight.
func (c *PeerClient) GetBlocks(fromHeight uint64, limit int) ([]*blockchain.Block, error) {
	path := fmt.Sprintf("/p2p/v1/chain/sync?from=%d&limit=%d", fromHeight, limit)
	var out struct {
		Blocks []*SerializedBlock `json:"blocks"`
		Total  int                `json:"total"`
	}
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	blocks := make([]*blockchain.Block, 0, len(out.Blocks))
	for _, sb := range out.Blocks {
		b, err := sb.Decode()
		if err != nil {
			return nil, fmt.Errorf("decode block: %w", err)
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// BroadcastBlock sends a newly mined block to the peer for inclusion.
func (c *PeerClient) BroadcastBlock(block *blockchain.Block) error {
	sb := encodeBlock(block)
	payload := map[string]interface{}{"block": sb}
	var out map[string]interface{}
	return c.do("POST", "/p2p/v1/blocks", payload, &out)
}

// NotifyNewValidator tells the peer about a new validator public key.
func (c *PeerClient) NotifyNewValidator(pubKeyHex string) error {
	payload := map[string]string{"validator_pubkey": pubKeyHex}
	return c.do("POST", "/p2p/v1/validators", payload, nil)
}

// GetValidators fetches the validator list from the peer.
func (c *PeerClient) GetValidators() ([]string, error) {
	var out struct {
		Validators []string `json:"validators"`
	}
	if err := c.do("GET", "/p2p/v1/validators", nil, &out); err != nil {
		return nil, err
	}
	return out.Validators, nil
}
