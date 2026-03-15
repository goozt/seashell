package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// KYCHandler serves KYC-related routes.
type KYCHandler struct {
	db     *store.DB
	kycSvc *service.KYCService
}

// NewKYCHandler creates a KYCHandler.
func NewKYCHandler(db *store.DB, kycSvc *service.KYCService) *KYCHandler {
	return &KYCHandler{db: db, kycSvc: kycSvc}
}

// InitiateKYC handles POST /api/v1/user/kyc.
func (h *KYCHandler) InitiateKYC(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	var req model.KYCInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	rec, err := h.kycSvc.InitiateKYC(r.Context(), claims.UserID, req.NationalID, req.TaxID)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, rec)
}

// GetMyKYC handles GET /api/v1/user/kyc.
func (h *KYCHandler) GetMyKYC(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	rec, err := h.kycSvc.GetKYCStatus(claims.UserID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, rec)
}

// ListKYCAdmin handles GET /api/v1/admin/kyc (admin+).
func (h *KYCHandler) ListKYCAdmin(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	records, err := h.db.ListKYCRecords(status)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, map[string]interface{}{"records": records, "total": len(records)})
}

// RejectKYCAdmin handles POST /api/v1/admin/kyc/{userID}/reject (admin+).
func (h *KYCHandler) RejectKYCAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req model.KYCRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Reason == "" {
		response.BadRequest(w, "reason is required")
		return
	}
	if err := h.kycSvc.AdminRejectKYC(userID, req.Reason); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, map[string]string{"status": "rejected"})
}

// KYCSummaryAdmin handles GET /api/v1/admin/kyc/summary (admin+).
func (h *KYCHandler) KYCSummaryAdmin(w http.ResponseWriter, r *http.Request) {
	// nodeID from claims node context
	claims := middleware.ClaimsFromContext(r.Context())
	nodeID := ""
	if claims != nil {
		nodeID = claims.UserID // approximate; real nodeID would come from config
	}
	summary, err := h.db.KYCSummary(nodeID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, summary)
}

// P2PKYCSummary handles GET /p2p/v1/kyc/summary (node secret auth).
func (h *KYCHandler) P2PKYCSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.db.KYCSummary("")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.OK(w, summary)
}
