package node

import (
	"log"
	"time"

	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

const (
	defaultPingInterval = 30 * time.Second
	offlineThreshold    = 3 // missed pings before marking offline
)

// HealthChecker periodically pings all known active peers and updates their
// status in the node registry.
type HealthChecker struct {
	db       *store.DB
	selfID   string
	selfURL  string
	secret   string
	interval time.Duration
	stop     chan struct{}
}

// NewHealthChecker creates a HealthChecker.
func NewHealthChecker(db *store.DB, selfID, selfURL, secret string, interval time.Duration) *HealthChecker {
	if interval <= 0 {
		interval = defaultPingInterval
	}
	return &HealthChecker{
		db:       db,
		selfID:   selfID,
		selfURL:  selfURL,
		secret:   secret,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start launches the background ping loop.  Call Stop() to shut down cleanly.
func (h *HealthChecker) Start() {
	go h.loop()
}

// Stop signals the health checker to stop and waits for it to exit.
func (h *HealthChecker) Stop() {
	close(h.stop)
}

func (h *HealthChecker) loop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.pingAll()
		case <-h.stop:
			return
		}
	}
}

func (h *HealthChecker) pingAll() {
	nodes, err := h.db.ListNodes()
	if err != nil {
		log.Printf("healthchecker: list nodes: %v", err)
		return
	}

	// Track miss counts in memory per invocation.
	for _, n := range nodes {
		if n.NodeURL == h.selfURL || n.IsPrimary && h.selfID == n.ID {
			// Don't ping ourselves.
			continue
		}
		if n.Status == model.NodeStatusPending || n.Status == model.NodeStatusRejected {
			continue
		}

		client := NewPeerClient(n.NodeURL, h.secret)
		resp, err := client.Ping(h.selfID, 0)
		now := time.Now()

		if err != nil {
			if n.Status != model.NodeStatusOffline {
				log.Printf("healthchecker: node %s (%s) unreachable: %v", n.ID, n.NodeURL, err)
				n.Status = model.NodeStatusOffline
				if saveErr := h.db.SaveNode(n); saveErr != nil {
					log.Printf("healthchecker: save node: %v", saveErr)
				}
			}
			continue
		}

		// Update node as online.
		n.LastSeenAt = &now
		n.BlockHeight = resp.BlockHeight
		if n.Status == model.NodeStatusOffline {
			n.Status = model.NodeStatusActive
			log.Printf("healthchecker: node %s back online", n.ID)
		}
		if saveErr := h.db.SaveNode(n); saveErr != nil {
			log.Printf("healthchecker: save node: %v", saveErr)
		}
	}
}
