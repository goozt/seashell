package node_test

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/node"
)

const testSecret = "test-secret"

// TestPeerClientPing verifies the ping round-trip including header authentication.
func TestPeerClientPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Node-Secret") != testSecret {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.Method != "POST" || r.URL.Path != "/p2p/v1/ping" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"node_id":      "peer-node-1",
				"block_height": float64(42),
				"status":       "ok",
			},
		})
	}))
	defer srv.Close()

	client := node.NewPeerClient(srv.URL, testSecret)
	resp, err := client.Ping("my-node", 10)
	if err != nil {
		t.Fatalf("ping error: %v", err)
	}
	if resp.NodeID != "peer-node-1" {
		t.Errorf("expected node_id peer-node-1, got %s", resp.NodeID)
	}
	if resp.BlockHeight != 42 {
		t.Errorf("expected block_height 42, got %d", resp.BlockHeight)
	}
}

// TestPeerClientPing_WrongSecret verifies that requests with wrong secret fail.
func TestPeerClientPing_WrongSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Node-Secret") != testSecret {
			http.Error(w, "unauthorized", 401)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": map[string]interface{}{}})
	}))
	defer srv.Close()

	client := node.NewPeerClient(srv.URL, "wrong-secret")
	_, err := client.Ping("my-node", 10)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

// TestPeerClientGetPeers verifies the peer list is decoded correctly.
func TestPeerClientGetPeers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Node-Secret") != testSecret {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"peers": []map[string]interface{}{
					{"id": "node-1", "node_url": "http://node1.test", "status": "active"},
					{"id": "node-2", "node_url": "http://node2.test", "status": "active"},
				},
			},
		})
	}))
	defer srv.Close()

	client := node.NewPeerClient(srv.URL, testSecret)
	peers, err := client.GetPeers()
	if err != nil {
		t.Fatalf("get peers error: %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}
}

// TestNotifyNewValidator verifies the validator notification payload.
func TestNotifyNewValidator(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Node-Secret") != testSecret {
			http.Error(w, "unauthorized", 401)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		received = body["validator_pubkey"]
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": nil})
	}))
	defer srv.Close()

	testKey := hex.EncodeToString(make([]byte, 64))
	client := node.NewPeerClient(srv.URL, testSecret)
	if err := client.NotifyNewValidator(testKey); err != nil {
		t.Fatalf("notify validator: %v", err)
	}
	if received != testKey {
		t.Errorf("expected key %s, got %s", testKey, received)
	}
}

// TestSerializedBlockRoundTrip verifies JSON marshal/unmarshal of SerializedBlock.
func TestSerializedBlockRoundTrip(t *testing.T) {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}
	original := &node.SerializedBlock{
		Hash:      hex.EncodeToString(hash),
		PrevHash:  hex.EncodeToString(make([]byte, 32)),
		Height:    5,
		Timestamp: 1700000000,
		Validator: hex.EncodeToString(make([]byte, 64)),
		Signature: hex.EncodeToString(make([]byte, 64)),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded node.SerializedBlock
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Height != 5 {
		t.Errorf("height mismatch: %d", decoded.Height)
	}
	if decoded.Hash != original.Hash {
		t.Errorf("hash mismatch: got %s, want %s", decoded.Hash, original.Hash)
	}
	if decoded.Timestamp != 1700000000 {
		t.Errorf("timestamp mismatch: %d", decoded.Timestamp)
	}
}

// TestBroadcastBlockNilPeers verifies that broadcasting to nil peers doesn't panic.
func TestBroadcastBlockNilPeers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BroadcastBlock panicked: %v", r)
		}
	}()
	node.BroadcastBlock(nil, nil, testSecret)
}

// TestBroadcastNewValidatorNilPeers verifies that broadcasting to nil peers doesn't panic.
func TestBroadcastNewValidatorNilPeers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("BroadcastNewValidator panicked: %v", r)
		}
	}()
	node.BroadcastNewValidator(nil, "aabbcc", testSecret)
}

// TestPeerURLs verifies self-exclusion and offline-node filtering.
func TestPeerURLs(t *testing.T) {
	nodes := []model.Node{
		{NodeURL: "http://self", Status: model.NodeStatusActive},
		{NodeURL: "http://peer1", Status: model.NodeStatusActive},
		{NodeURL: "http://peer2", Status: model.NodeStatusOffline},
		{NodeURL: "http://peer3", Status: model.NodeStatusActive},
	}
	peers := node.PeerURLs(nodes, "http://self")
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d: %v", len(peers), peers)
	}
	for _, p := range peers {
		if p.NodeURL == "http://self" {
			t.Error("self should be excluded")
		}
		if p.Status != model.NodeStatusActive {
			t.Errorf("offline node %s should be excluded", p.NodeURL)
		}
	}
}
