package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/store"
)

// ArchiveHandler serves archive query routes.
type ArchiveHandler struct {
	archive *store.ArchiveDB
}

// NewArchiveHandler creates an ArchiveHandler.
func NewArchiveHandler(archive *store.ArchiveDB) *ArchiveHandler {
	return &ArchiveHandler{archive: archive}
}

// GetArchiveStats handles GET /api/v1/archive/stats.
func (h *ArchiveHandler) GetArchiveStats(w http.ResponseWriter, r *http.Request) {
	meta, err := h.archive.GetMeta()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, meta)
}

// GetArchiveBlock handles GET /api/v1/archive/blocks/{height}.
func (h *ArchiveHandler) GetArchiveBlock(w http.ResponseWriter, r *http.Request) {
	heightStr := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid height")
		return
	}
	block, err := h.archive.ReadBlock(height)
	if err != nil {
		response.NotFound(w, "block not found in archive")
		return
	}
	response.OK(w, block)
}

// GetArchiveRange handles GET /api/v1/archive/blocks?from=<h>&to=<h>.
func (h *ArchiveHandler) GetArchiveRange(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err := strconv.ParseUint(fromStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid 'from' parameter")
		return
	}
	to, err := strconv.ParseUint(toStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid 'to' parameter")
		return
	}
	if to < from || to-from > 1000 {
		response.BadRequest(w, "'to' must be >= 'from' and range must not exceed 1000 blocks")
		return
	}

	blocks, err := h.archive.ReadBlockRange(from, to)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, map[string]interface{}{"blocks": blocks, "total": len(blocks)})
}
