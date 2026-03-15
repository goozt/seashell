package service

import (
	"context"
	"log"
	"time"

	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/store"
)

// ArchivalService moves blocks older than the retention window from the live chain
// into the gzip-compressed GOB archive. It runs nightly at 02:00 local time.
type ArchivalService struct {
	chainSvc *ChainService
	archive  *store.ArchiveDB
	cfg      *config.Config
}

// NewArchivalService creates an ArchivalService.
func NewArchivalService(chainSvc *ChainService, archive *store.ArchiveDB, cfg *config.Config) *ArchivalService {
	return &ArchivalService{chainSvc: chainSvc, archive: archive, cfg: cfg}
}

// Start launches the nightly archival goroutine.
func (s *ArchivalService) Start() {
	go s.loop()
	log.Println("archival: background worker started")
}

// loop wakes at 02:00 local time every day and runs RunArchival.
func (s *ArchivalService) loop() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		if err := s.RunArchival(ctx); err != nil {
			log.Printf("archival: error: %v", err)
		}
		cancel()
	}
}

// RunArchival walks the live chain from height 0 upward and moves blocks whose
// timestamp precedes the retention cutoff into the archive, then deletes them
// from the live BadgerDB.
func (s *ArchivalService) RunArchival(ctx context.Context) error {
	cutoff := store.ArchiveCutoff(s.cfg.ArchiveRetentionYears)
	log.Printf("archival: starting — cutoff unix=%d (%s)", cutoff, time.Unix(cutoff, 0).Format("2006-01-02"))

	s.chainSvc.mu.Lock()
	defer s.chainSvc.mu.Unlock()

	chain, err := s.chainSvc.openChain()
	if err != nil {
		return err
	}
	defer chain.Close()

	archived := 0
	errCount := 0

	// Walk from height 0 upward. Stop when we reach a block newer than the cutoff
	// or exhaust the height index. Use current height as upper bound.
	currentHeight := chain.GetCurrentHeight()
	for h := uint64(0); h <= currentHeight; h++ {
		select {
		case <-ctx.Done():
			log.Printf("archival: context cancelled after %d archived", archived)
			return ctx.Err()
		default:
		}

		block, err := chain.GetBlockByHeight(h)
		if err != nil {
			// Height index may be missing for genesis / pre-index blocks.
			continue
		}

		if int64(block.Timestamp) >= cutoff {
			break // remaining blocks are within retention window
		}

		if err := s.archive.WriteBlock(block); err != nil {
			log.Printf("archival: write block %d: %v", h, err)
			errCount++
			continue
		}
		if err := chain.DeleteBlock(block.Hash); err != nil {
			log.Printf("archival: delete block %d from live chain: %v", h, err)
			errCount++
		} else {
			archived++
		}
	}

	log.Printf("archival: done — %d archived, %d errors", archived, errCount)
	return nil
}
