package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/node"
	"github.com/goozt/seashell/store"
)

// CertIssueHandler handles POST /p2p/v1/cert-issue.
// Only mounted on primary and regional nodes (where a CA is available).
type CertIssueHandler struct {
	db *store.DB
	ca *node.CA
}

// NewCertIssueHandler creates a CertIssueHandler.
func NewCertIssueHandler(db *store.DB, ca *node.CA) *CertIssueHandler {
	return &CertIssueHandler{db: db, ca: ca}
}

type certIssueRequest struct {
	CSRPEM   []byte `json:"csr_pem"`
	NodeID   string `json:"node_id"`
	NodeTier string `json:"node_tier"`
}

type certIssueResponse struct {
	CertPEM    []byte `json:"cert_pem"`
	CACertPEM  []byte `json:"ca_cert_pem"`
}

// Issue handles POST /p2p/v1/cert-issue.
// The requesting node must already be in an approved state in the registry.
func (h *CertIssueHandler) Issue(w http.ResponseWriter, r *http.Request) {
	var req certIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if len(req.CSRPEM) == 0 || req.NodeID == "" || req.NodeTier == "" {
		response.BadRequest(w, "csr_pem, node_id, and node_tier are required")
		return
	}

	// Verify the requesting node is approved.
	n, err := h.db.GetNodeByID(req.NodeID)
	if err != nil || n.Status != "active" {
		response.Forbidden(w)
		return
	}

	certPEM, err := h.ca.IssueNodeCert(req.CSRPEM, req.NodeTier, req.NodeID)
	if err != nil {
		response.InternalError(w, "cert issuance failed: "+err.Error())
		return
	}

	response.OK(w, certIssueResponse{
		CertPEM:   certPEM,
		CACertPEM: h.ca.CACertPEM(),
	})
}
