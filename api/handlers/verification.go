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

// VerificationHandler serves verification-related routes.
type VerificationHandler struct {
	db        *store.DB
	verifySvc *service.VerificationService
}

// NewVerificationHandler creates a VerificationHandler.
func NewVerificationHandler(db *store.DB, verifySvc *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{db: db, verifySvc: verifySvc}
}

// --------------------------------------------------------------------------
// Authority owner endpoints
// --------------------------------------------------------------------------

// SaveConfig handles PUT /api/v1/authority/verification/config.
func (h *VerificationHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AuthorityID == "" {
		response.Unauthorized(w)
		return
	}

	var req model.SaveVerificationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	cfg, err := h.verifySvc.SaveConfig(claims.AuthorityID, &req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, cfg)
}

// GetConfig handles GET /api/v1/authority/verification/config.
func (h *VerificationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AuthorityID == "" {
		response.Unauthorized(w)
		return
	}

	cfg, err := h.verifySvc.GetConfig(claims.AuthorityID)
	if err != nil {
		if store.IsNotFound(err) {
			response.OK(w, nil)
			return
		}
		response.InternalError(w, "could not load verification config")
		return
	}
	response.OK(w, cfg)
}

// ListSubmissions handles GET /api/v1/authority/verification/submissions?status=.
func (h *VerificationHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AuthorityID == "" {
		response.Unauthorized(w)
		return
	}

	status := r.URL.Query().Get("status")
	subs, err := h.db.ListVerificationSubmissions(claims.AuthorityID, status)
	if err != nil {
		response.InternalError(w, "could not load submissions")
		return
	}
	if subs == nil {
		subs = []*model.VerificationSubmission{}
	}

	// Enrich submissions with user info.
	type enrichedSubmission struct {
		*model.VerificationSubmission
		Username string `json:"username"`
		FullName string `json:"full_name"`
	}
	enriched := make([]enrichedSubmission, 0, len(subs))
	for _, sub := range subs {
		es := enrichedSubmission{VerificationSubmission: sub}
		if user, err := h.db.GetUserByID(sub.UserID); err == nil {
			es.Username = user.Username
			es.FullName = user.FullName()
		}
		enriched = append(enriched, es)
	}

	response.OK(w, enriched)
}

// GetSubmission handles GET /api/v1/authority/verification/submissions/{id}.
func (h *VerificationHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AuthorityID == "" {
		response.Unauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	sub, err := h.db.GetVerificationSubmission(id)
	if err != nil {
		if store.IsNotFound(err) {
			response.NotFound(w, "submission not found")
			return
		}
		response.InternalError(w, "could not load submission")
		return
	}

	// Ensure submission belongs to the owner's authority.
	if sub.AuthorityID != claims.AuthorityID {
		response.NotFound(w, "submission not found")
		return
	}

	response.OK(w, sub)
}

// ReviewSubmission handles POST /api/v1/authority/verification/submissions/{id}/review.
func (h *VerificationHandler) ReviewSubmission(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.AuthorityID == "" {
		response.Unauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")

	// Verify submission belongs to this authority before reviewing.
	existing, err := h.db.GetVerificationSubmission(id)
	if err != nil {
		if store.IsNotFound(err) {
			response.NotFound(w, "submission not found")
			return
		}
		response.InternalError(w, "could not load submission")
		return
	}
	if existing.AuthorityID != claims.AuthorityID {
		response.NotFound(w, "submission not found")
		return
	}

	var req model.ReviewVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	sub, err := h.verifySvc.Review(id, claims.UserID, &req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, sub)
}

// --------------------------------------------------------------------------
// User endpoints
// --------------------------------------------------------------------------

// VerificationStatusResponse is the composite response for GET /user/verification.
type VerificationStatusResponse struct {
	ConfigStatus string                         `json:"config_status"` // "not_configured" | "configured"
	Config       *model.VerificationConfig      `json:"config,omitempty"`
	Submission   *model.VerificationSubmission   `json:"submission,omitempty"`
}

// GetMyVerification handles GET /api/v1/user/verification.
func (h *VerificationHandler) GetMyVerification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		response.InternalError(w, "could not load user")
		return
	}

	// User must be affiliated with an authority.
	if !user.IsAffiliated() {
		response.OK(w, VerificationStatusResponse{ConfigStatus: "not_configured"})
		return
	}

	// Check if authority has a verification config.
	cfg, err := h.verifySvc.GetConfig(user.AuthorityID)
	if err != nil {
		if store.IsNotFound(err) {
			response.OK(w, VerificationStatusResponse{ConfigStatus: "not_configured"})
			return
		}
		response.InternalError(w, "could not load verification config")
		return
	}

	resp := VerificationStatusResponse{
		ConfigStatus: "configured",
		Config:       cfg,
	}

	// Get user's latest submission if any.
	sub, err := h.verifySvc.GetUserStatus(user.ID, user.AuthorityID)
	if err != nil {
		response.InternalError(w, "could not load verification status")
		return
	}
	resp.Submission = sub

	response.OK(w, resp)
}

// SubmitVerification handles POST /api/v1/user/verification.
func (h *VerificationHandler) SubmitVerification(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		response.InternalError(w, "could not load user")
		return
	}

	if !user.IsAffiliated() {
		response.BadRequest(w, "you must belong to an authority to submit verification")
		return
	}

	var req model.SubmitVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	sub, err := h.verifySvc.Submit(user.ID, user.AuthorityID, &req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, sub)
}
