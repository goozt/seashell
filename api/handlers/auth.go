package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// AuthHandler handles /api/v1/auth/* routes.
type AuthHandler struct {
	db      *store.DB
	authSvc *service.AuthService
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(db *store.DB, authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{db: db, authSvc: authSvc}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if msg := req.Validate(); msg != "" {
		response.BadRequest(w, msg)
		return
	}
	if h.db.EmailExists(req.Email) {
		response.Conflict(w, "email already registered")
		return
	}
	if h.db.UsernameExists(req.Username) {
		response.Conflict(w, "username already taken")
		return
	}

	hash, err := service.HashPassword(req.Password)
	if err != nil {
		response.InternalError(w, "could not hash password")
		return
	}

	now := time.Now()
	u := &model.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         model.RoleUser,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.db.SaveUser(u); err != nil {
		response.InternalError(w, "could not save user")
		return
	}

	tokens, err := h.authSvc.IssueTokenPair(u)
	if err != nil {
		response.InternalError(w, "could not issue tokens")
		return
	}

	response.Created(w, map[string]interface{}{
		"user":   u.SafeCopy(),
		"tokens": tokens,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := h.db.GetUserByEmail(req.Email)
	if err != nil || !u.Active {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := service.CheckPassword(u.PasswordHash, req.Password); err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	tokens, err := h.authSvc.IssueTokenPair(u)
	if err != nil {
		response.InternalError(w, "could not issue tokens")
		return
	}

	response.OK(w, map[string]interface{}{
		"user":   u.SafeCopy(),
		"tokens": tokens,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		response.BadRequest(w, "refresh_token required")
		return
	}

	u, err := h.authSvc.ValidateRefreshToken(body.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	tokens, err := h.authSvc.IssueTokenPair(u)
	if err != nil {
		response.InternalError(w, "could not issue tokens")
		return
	}
	// Revoke old refresh token after issuing new one.
	_ = h.authSvc.RevokeRefreshToken(body.RefreshToken)

	response.OK(w, tokens)
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.RefreshToken != "" {
		_ = h.authSvc.RevokeRefreshToken(body.RefreshToken)
	}
	// Also clear claims from context if needed (stateless access tokens expire naturally).
	response.OK(w, map[string]string{"message": "logged out"})
}

// Me handles GET /api/v1/auth/me - returns current user from token (no DB lookup needed for basic info).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	u, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}
	response.OK(w, u.SafeCopy())
}
