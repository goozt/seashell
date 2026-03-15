package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// AuthorityOwnerHandler serves /api/v1/authority/* (authority owners only).
type AuthorityOwnerHandler struct {
	db       *store.DB
	chainSvc *service.ChainService
	valueSvc *service.ValueService
}

// NewAuthorityOwnerHandler creates an AuthorityOwnerHandler.
func NewAuthorityOwnerHandler(db *store.DB, chainSvc *service.ChainService, valueSvc *service.ValueService) *AuthorityOwnerHandler {
	return &AuthorityOwnerHandler{db: db, chainSvc: chainSvc, valueSvc: valueSvc}
}

func (h *AuthorityOwnerHandler) ownerAuthorityID(r *http.Request) string {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return ""
	}
	return claims.AuthorityID
}

// GetMembers handles GET /api/v1/authority/members.
func (h *AuthorityOwnerHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	authorityID := h.ownerAuthorityID(r)
	members, err := h.db.ListUsersByAuthorityID(authorityID)
	if err != nil {
		response.InternalError(w, "could not load members")
		return
	}
	safe := make([]model.User, 0, len(members))
	for _, m := range members {
		safe = append(safe, m.SafeCopy())
	}
	response.OK(w, safe)
}

// RemoveMember handles DELETE /api/v1/authority/members/{userID}.
func (h *AuthorityOwnerHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	targetID := chi.URLParam(r, "userID")
	if targetID == claims.UserID {
		response.BadRequest(w, "cannot remove yourself")
		return
	}
	target, err := h.db.GetUserByID(targetID)
	if err != nil || target.AuthorityID != claims.AuthorityID {
		response.NotFound(w, "member not found")
		return
	}
	target.AuthorityID = ""
	target.AuthorityRole = ""
	target.UpdatedAt = time.Now()
	if err := h.db.SaveUser(target); err != nil {
		response.InternalError(w, "could not remove member")
		return
	}
	response.OK(w, map[string]string{"message": "member removed"})
}

// GetInvitations handles GET /api/v1/authority/invitations.
func (h *AuthorityOwnerHandler) GetInvitations(w http.ResponseWriter, r *http.Request) {
	authorityID := h.ownerAuthorityID(r)
	invitations, err := h.db.ListInvitationsByAuthority(authorityID)
	if err != nil {
		response.InternalError(w, "could not load invitations")
		return
	}
	response.OK(w, invitations)
}

// CreateInvitation handles POST /api/v1/authority/invitations.
func (h *AuthorityOwnerHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	code, err := generateInviteCode()
	if err != nil {
		response.InternalError(w, "could not generate code")
		return
	}
	inv := &model.Invitation{
		Code:        code,
		AuthorityID: claims.AuthorityID,
		CreatedBy:   claims.UserID,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	if err := h.db.SaveInvitation(inv); err != nil {
		response.InternalError(w, "could not save invitation")
		return
	}
	response.Created(w, inv)
}

// RevokeInvitation handles DELETE /api/v1/authority/invitations/{code}.
func (h *AuthorityOwnerHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	authorityID := h.ownerAuthorityID(r)
	inv, err := h.db.GetInvitationByCode(code)
	if err != nil || inv.AuthorityID != authorityID {
		response.NotFound(w, "invitation not found")
		return
	}
	if err := h.db.DeleteInvitation(code); err != nil {
		response.InternalError(w, "could not revoke invitation")
		return
	}
	response.NoContent(w)
}

// GetAuthorityTransactions handles GET /api/v1/authority/transactions.
func (h *AuthorityOwnerHandler) GetAuthorityTransactions(w http.ResponseWriter, r *http.Request) {
	authorityID := h.ownerAuthorityID(r)
	members, err := h.db.ListUsersByAuthorityID(authorityID)
	if err != nil {
		response.InternalError(w, "could not load members")
		return
	}
	var allTxs []map[string]interface{}
	for _, m := range members {
		if m.WalletAddress == "" {
			continue
		}
		txs, err := h.chainSvc.GetTransactionsForAddress(m.WalletAddress)
		if err != nil {
			continue
		}
		for _, tx := range txs {
			tx["member_id"] = m.ID
			tx["member_username"] = m.Username
			allTxs = append(allTxs, tx)
		}
	}
	if allTxs == nil {
		allTxs = []map[string]interface{}{}
	}
	response.OK(w, allTxs)
}

// GetAuthorityValue handles GET /api/v1/authority/value.
func (h *AuthorityOwnerHandler) GetAuthorityValue(w http.ResponseWriter, r *http.Request) {
	authorityID := h.ownerAuthorityID(r)
	history, err := h.db.ListValueHistory(authorityID)
	if err != nil {
		response.InternalError(w, "could not load value history")
		return
	}
	response.OK(w, history)
}

// GetAuthorityStats handles GET /api/v1/authority/stats.
func (h *AuthorityOwnerHandler) GetAuthorityStats(w http.ResponseWriter, r *http.Request) {
	authorityID := h.ownerAuthorityID(r)
	members, err := h.db.ListUsersByAuthorityID(authorityID)
	if err != nil {
		response.InternalError(w, "could not load members")
		return
	}
	tickets, err := h.db.ListTicketsByAuthority(authorityID)
	if err != nil {
		response.InternalError(w, "could not load tickets")
		return
	}
	latest, _ := h.db.GetLatestValue(authorityID)
	currentPrice := 0.0
	if latest != nil {
		currentPrice = latest.Price
	}
	response.OK(w, map[string]interface{}{
		"member_count":  len(members),
		"ticket_count":  len(tickets),
		"current_price": currentPrice,
	})
}

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
