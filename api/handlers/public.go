package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// PublicHandler serves unauthenticated endpoints.
type PublicHandler struct {
	chainSvc *service.ChainService
	valueSvc *service.ValueService
	db       *store.DB
}

// NewPublicHandler creates a PublicHandler.
func NewPublicHandler(chainSvc *service.ChainService, valueSvc *service.ValueService, db *store.DB) *PublicHandler {
	return &PublicHandler{chainSvc: chainSvc, valueSvc: valueSvc, db: db}
}

// Health handles GET /api/v1/health.
func (h *PublicHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"status": "ok", "service": "seashell"})
}

// GetBlocks handles GET /api/v1/blocks.
func (h *PublicHandler) GetBlocks(w http.ResponseWriter, r *http.Request) {
	limit := 20
	blocks, err := h.chainSvc.GetBlocks(limit)
	if err != nil {
		response.InternalError(w, "could not load blocks")
		return
	}
	response.OK(w, blocks)
}

// GetBlock handles GET /api/v1/blocks/{hash}.
func (h *PublicHandler) GetBlock(w http.ResponseWriter, r *http.Request) {
	hashHex := chi.URLParam(r, "hash")
	block, valid, err := h.chainSvc.GetBlock(hashHex)
	if err != nil {
		response.InternalError(w, "could not load block")
		return
	}
	if block == nil {
		response.NotFound(w, "block not found")
		return
	}
	response.OK(w, map[string]interface{}{
		"hash":      hashHex,
		"height":    block.Height,
		"timestamp": block.Timestamp,
		"valid_poa": valid,
		"tx_count":  len(block.Transactions),
	})
}

// GetValue handles GET /api/v1/value — returns current price for all active authorities.
func (h *PublicHandler) GetValue(w http.ResponseWriter, r *http.Request) {
	authorities, err := h.db.ListActiveAuthorities()
	if err != nil {
		response.InternalError(w, "could not load authorities")
		return
	}
	type priceEntry struct {
		AuthorityID   string  `json:"authority_id"`
		AuthorityName string  `json:"authority_name"`
		Price         float64 `json:"price"`
		BlockHeight   uint64  `json:"block_height"`
	}
	var entries []priceEntry
	for _, a := range authorities {
		rec, err := h.db.GetLatestValue(a.ID)
		if err != nil {
			entries = append(entries, priceEntry{AuthorityID: a.ID, AuthorityName: a.Name, Price: a.BasePrice})
			continue
		}
		entries = append(entries, priceEntry{
			AuthorityID:   a.ID,
			AuthorityName: a.Name,
			Price:         rec.Price,
			BlockHeight:   rec.BlockHeight,
		})
	}
	response.OK(w, entries)
}
