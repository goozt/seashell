package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// SuperAdminHandler serves /api/v1/superadmin/* routes.
type SuperAdminHandler struct {
	db      *store.DB
	authSvc *service.AuthService
}

// NewSuperAdminHandler creates a SuperAdminHandler.
func NewSuperAdminHandler(db *store.DB, authSvc *service.AuthService) *SuperAdminHandler {
	return &SuperAdminHandler{db: db, authSvc: authSvc}
}

// GetAdmins handles GET /api/v1/superadmin/admins.
func (h *SuperAdminHandler) GetAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.db.ListAdmins()
	if err != nil {
		response.InternalError(w, "could not load admins")
		return
	}
	safe := make([]model.User, 0, len(admins))
	for _, a := range admins {
		safe = append(safe, a.SafeCopy())
	}
	response.OK(w, safe)
}

// CreateAdmin handles POST /api/v1/superadmin/admins.
func (h *SuperAdminHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
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
	admin := &model.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.db.SaveUser(admin); err != nil {
		response.InternalError(w, "could not create admin")
		return
	}
	response.Created(w, admin.SafeCopy())
}

// DemoteAdmin handles DELETE /api/v1/superadmin/admins/{id} — demotes to user role.
func (h *SuperAdminHandler) DemoteAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.db.GetUserByID(id)
	if err != nil || u.Role != model.RoleAdmin {
		response.NotFound(w, "admin not found")
		return
	}
	u.Role = model.RoleUser
	u.UpdatedAt = time.Now()
	if err := h.db.SaveUser(u); err != nil {
		response.InternalError(w, "could not update user")
		return
	}
	response.OK(w, map[string]string{"message": "admin demoted to user"})
}

// SuspendAuthority handles POST /api/v1/superadmin/authorities/{id}/suspend.
func (h *SuperAdminHandler) SuspendAuthority(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authority, err := h.db.GetAuthorityByID(id)
	if err != nil {
		response.NotFound(w, "authority not found")
		return
	}
	authority.Status = model.AuthorityStatusSuspended
	if err := h.db.SaveAuthority(authority); err != nil {
		response.InternalError(w, "could not suspend authority")
		return
	}
	response.OK(w, authority)
}

// ReinstateAuthority handles POST /api/v1/superadmin/authorities/{id}/reinstate.
func (h *SuperAdminHandler) ReinstateAuthority(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authority, err := h.db.GetAuthorityByID(id)
	if err != nil {
		response.NotFound(w, "authority not found")
		return
	}
	authority.Status = model.AuthorityStatusActive
	if err := h.db.SaveAuthority(authority); err != nil {
		response.InternalError(w, "could not reinstate authority")
		return
	}
	response.OK(w, authority)
}

// GetStatsSuperAdmin handles GET /api/v1/superadmin/stats.
func (h *SuperAdminHandler) GetStatsSuperAdmin(w http.ResponseWriter, r *http.Request) {
	_, totalUsers, _ := h.db.ListUsers(1, 1)
	admins, _ := h.db.ListAdmins()
	authorities, _ := h.db.ListAuthorities()
	pending, _ := h.db.ListPendingAuthorities()
	tickets, _ := h.db.ListAllTickets()
	activeCount := 0
	for _, a := range authorities {
		if a.Status == model.AuthorityStatusActive {
			activeCount++
		}
	}
	response.OK(w, map[string]interface{}{
		"total_users":         totalUsers,
		"total_admins":        len(admins),
		"total_authorities":   len(authorities),
		"active_authorities":  activeCount,
		"pending_authorities": len(pending),
		"total_tickets":       len(tickets),
	})
}
