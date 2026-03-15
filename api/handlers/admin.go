package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// AdminHandler serves /api/v1/admin/* routes.
type AdminHandler struct {
	db          *store.DB
	chainDBPath string
	hub         *service.WSHub
	pushSvc     *service.PushService
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(db *store.DB, chainDBPath string, hub *service.WSHub, pushSvc *service.PushService) *AdminHandler {
	return &AdminHandler{db: db, chainDBPath: chainDBPath, hub: hub, pushSvc: pushSvc}
}

// GetAuthorityRequests handles GET /api/v1/admin/authority-requests.
func (h *AdminHandler) GetAuthorityRequests(w http.ResponseWriter, r *http.Request) {
	pending, err := h.db.ListPendingAuthorities()
	if err != nil {
		response.InternalError(w, "could not load requests")
		return
	}
	response.OK(w, pending)
}

// GetAuthorityRequest handles GET /api/v1/admin/authority-requests/{id}.
func (h *AdminHandler) GetAuthorityRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authority, err := h.db.GetAuthorityByID(id)
	if err != nil {
		response.NotFound(w, "authority not found")
		return
	}
	response.OK(w, authority)
}

// ApproveAuthorityRequest handles POST /api/v1/admin/authority-requests/{id}/approve.
func (h *AdminHandler) ApproveAuthorityRequest(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	id := chi.URLParam(r, "id")
	authority, err := h.db.GetAuthorityByID(id)
	if err != nil || authority.Status != model.AuthorityStatusPending {
		response.NotFound(w, "pending request not found")
		return
	}

	var req model.ApproveAuthorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if req.BasePrice <= 0 {
		req.BasePrice = 1.0
	}
	if req.SensitivityK <= 0 {
		req.SensitivityK = 0.1
	}

	// Generate a new ECDSA keypair for this authority's validator.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		response.InternalError(w, "could not generate validator key")
		return
	}
	pubKeyBytes := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)

	// Store private key raw D bytes in the API DB (never returned in responses).
	if err := h.db.StoreAuthorityPrivKey(id, privKey.D.Bytes()); err != nil {
		response.InternalError(w, "could not store validator key")
		return
	}

	// Register the public key as a blockchain validator.
	// Open the chain DB directly; we hold a mutex via ChainService but need a one-off write here.
	if err := h.registerValidator(pubKeyBytes); err != nil {
		response.InternalError(w, "could not register validator: "+err.Error())
		return
	}

	now := time.Now()
	authority.Status = model.AuthorityStatusActive
	authority.ValidatorPubKey = hex.EncodeToString(pubKeyBytes)
	authority.BasePrice = req.BasePrice
	authority.SensitivityK = req.SensitivityK
	authority.ApprovedAt = &now
	authority.ApprovedBy = claims.UserID
	if err := h.db.SaveAuthority(authority); err != nil {
		response.InternalError(w, "could not update authority")
		return
	}

	// Make the authority owner a full authority member.
	owner, err := h.db.GetUserByID(authority.OwnerID)
	if err == nil {
		owner.AuthorityID = authority.ID
		owner.AuthorityRole = model.AuthorityRoleOwner
		owner.UpdatedAt = time.Now()
		_ = h.db.SaveUser(owner)
	}

	response.OK(w, authority)
}

// registerValidator opens the blockchain and adds a validator pubkey.
func (h *AdminHandler) registerValidator(pubKeyBytes []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()
	blockchain.AddValidatorToDB(chain.Database, pubKeyBytes)
	return nil
}

// RejectAuthorityRequest handles POST /api/v1/admin/authority-requests/{id}/reject.
func (h *AdminHandler) RejectAuthorityRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	authority, err := h.db.GetAuthorityByID(id)
	if err != nil || authority.Status != model.AuthorityStatusPending {
		response.NotFound(w, "pending request not found")
		return
	}
	var req model.RejectAuthorityRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	authority.Status = model.AuthorityStatusRejected
	authority.RejectionReason = req.Reason
	if err := h.db.SaveAuthority(authority); err != nil {
		response.InternalError(w, "could not update authority")
		return
	}
	response.OK(w, authority)
}

// GetTickets handles GET /api/v1/admin/tickets.
// Returns node-level tickets scoped to the admin's own authority.
// Superadmins (no authority) see all node-level tickets.
func (h *AdminHandler) GetTickets(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}

	tickets, err := h.db.ListTicketsByLevel(model.TicketLevelNode)
	if err != nil {
		response.InternalError(w, "could not load tickets")
		return
	}

	// Scope to own authority unless superadmin (who has no authority).
	if claims.AuthorityID != "" && claims.Role != "superadmin" {
		filtered := tickets[:0]
		for _, t := range tickets {
			if t.AuthorityID == claims.AuthorityID {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}

	response.OK(w, tickets)
}

// GetTicketAdmin handles GET /api/v1/admin/tickets/{id}.
func (h *AdminHandler) GetTicketAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	response.OK(w, ticket)
}

// UpdateTicket handles PUT /api/v1/admin/tickets/{id}.
func (h *AdminHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	var req model.UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		response.BadRequest(w, "status is required")
		return
	}
	ticket.Status = req.Status
	ticket.UpdatedAt = time.Now()
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not update ticket")
		return
	}
	response.OK(w, ticket)
}

// ReplyToTicketAdmin handles POST /api/v1/admin/tickets/{id}/reply.
func (h *AdminHandler) ReplyToTicketAdmin(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	var req model.ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		response.BadRequest(w, "message is required")
		return
	}
	authorUsername := claims.UserID // fallback to ID if lookup fails
	if author, err := h.db.GetUserByID(claims.UserID); err == nil {
		authorUsername = author.Username
	}
	reply := model.TicketReply{
		ID:             generateID(),
		AuthorID:       claims.UserID,
		AuthorUsername: authorUsername,
		AuthorRole:     claims.Role,
		Message:        req.Message,
		CreatedAt:      time.Now(),
	}
	ticket.Replies = append(ticket.Replies, reply)
	ticket.Status = model.TicketStatusInProgress
	ticket.UpdatedAt = time.Now()
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not save reply")
		return
	}

	// Notify the ticket creator about the new reply.
	h.hub.SendTo(ticket.CreatedBy, model.WSTypeSupport, generateID(), model.SupportPayload{
		TicketID: ticket.ID,
		Reply:    reply,
	})
	if creator, err := h.db.GetUserByID(ticket.CreatedBy); err == nil {
		h.hub.SendTo(creator.ID, model.WSTypeNotification, generateID(), model.NotificationPayload{
			Title: "New reply on: " + ticket.Title,
			Body:  reply.Message,
			Link:  "/dashboard/tickets/" + ticket.ID,
		})
	}
	h.pushSvc.SendToUser(ticket.CreatedBy, model.PushPayload{
		Title: "New reply on: " + ticket.Title,
		Body:  reply.Message,
		Icon:  "/logo-icon.png",
		URL:   "/dashboard/tickets/" + ticket.ID,
		Tag:   "ticket-" + ticket.ID,
	})

	response.Created(w, ticket)
}

// EscalateTicketToRegional handles POST /api/v1/admin/tickets/{id}/escalate.
// Node admin escalates an unresolved ticket to the regional admin.
func (h *AdminHandler) EscalateTicketToRegional(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	if ticket.EscalatedTo == model.TicketLevelRegional || ticket.EscalatedTo == model.TicketLevelSuper {
		response.BadRequest(w, "ticket already escalated beyond node level")
		return
	}
	var req model.EscalateTicketRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	ticket.EscalatedTo = model.TicketLevelRegional
	ticket.EscalationNote = req.Note
	ticket.Status = model.TicketStatusOpen
	ticket.UpdatedAt = time.Now()
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not escalate ticket")
		return
	}

	// Alert: notify all active admins in this authority.
	if users, _, err := h.db.ListUsers(1, 1000); err == nil {
		for _, u := range users {
			if (u.Role == model.RoleAdmin || u.Role == model.RoleSuperAdmin) && u.AuthorityID == ticket.AuthorityID {
				h.hub.SendTo(u.ID, model.WSTypeAlert, generateID(), model.AlertPayload{
					Title:    "Ticket escalated to regional",
					Body:     ticket.Title,
					Severity: "warning",
				})
				h.pushSvc.SendToUser(u.ID, model.PushPayload{
					Title: "Ticket escalated",
					Body:  ticket.Title,
					Icon:  "/logo-icon.png",
					URL:   "/admin/tickets",
					Tag:   "escalation-" + ticket.ID,
				})
			}
		}
	}

	response.OK(w, ticket)
}

// GetTicketsRegional handles GET /api/v1/admin/tickets/regional.
// Returns tickets escalated to regional level (visible to regional/superadmin).
func (h *AdminHandler) GetTicketsRegional(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.db.ListTicketsByLevel(model.TicketLevelRegional)
	if err != nil {
		response.InternalError(w, "could not load tickets")
		return
	}
	response.OK(w, tickets)
}

// EscalateTicketToSuper handles POST /api/v1/admin/tickets/{id}/escalate-super.
// Regional admin escalates to superadmin.
func (h *AdminHandler) EscalateTicketToSuper(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	if ticket.EscalatedTo != model.TicketLevelRegional {
		response.BadRequest(w, "ticket must be at regional level to escalate to superadmin")
		return
	}
	var req model.EscalateTicketRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	ticket.EscalatedTo = model.TicketLevelSuper
	ticket.EscalationNote = req.Note
	ticket.Status = model.TicketStatusOpen
	ticket.UpdatedAt = time.Now()
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not escalate ticket")
		return
	}

	// Alert: notify all superadmins.
	if users, _, err := h.db.ListUsers(1, 1000); err == nil {
		for _, u := range users {
			if u.Role == model.RoleSuperAdmin {
				h.hub.SendTo(u.ID, model.WSTypeAlert, generateID(), model.AlertPayload{
					Title:    "Ticket escalated to superadmin",
					Body:     ticket.Title,
					Severity: "warning",
				})
				h.pushSvc.SendToUser(u.ID, model.PushPayload{
					Title: "Ticket escalated",
					Body:  ticket.Title,
					Icon:  "/logo-icon.png",
					URL:   "/admin/tickets",
					Tag:   "escalation-" + ticket.ID,
				})
			}
		}
	}

	response.OK(w, ticket)
}

// GetTicketsSuperAdmin handles GET /api/v1/superadmin/tickets.
// Returns tickets escalated to superadmin level.
func (h *AdminHandler) GetTicketsSuperAdmin(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.db.ListTicketsByLevel(model.TicketLevelSuper)
	if err != nil {
		response.InternalError(w, "could not load tickets")
		return
	}
	response.OK(w, tickets)
}

// GetAuthoritiesAdmin handles GET /api/v1/admin/authorities.
func (h *AdminHandler) GetAuthoritiesAdmin(w http.ResponseWriter, r *http.Request) {
	authorities, err := h.db.ListAuthorities()
	if err != nil {
		response.InternalError(w, "could not load authorities")
		return
	}
	response.OK(w, authorities)
}

// GetUsersAdmin handles GET /api/v1/admin/users.
func (h *AdminHandler) GetUsersAdmin(w http.ResponseWriter, r *http.Request) {
	users, _, err := h.db.ListUsers(1, 1000)
	if err != nil {
		response.InternalError(w, "could not load users")
		return
	}
	safe := make([]model.User, 0, len(users))
	for _, u := range users {
		safe = append(safe, u.SafeCopy())
	}
	response.OK(w, safe)
}

// GetStatsAdmin handles GET /api/v1/admin/stats.
func (h *AdminHandler) GetStatsAdmin(w http.ResponseWriter, r *http.Request) {
	users, total, _ := h.db.ListUsers(1, 1)
	authorities, _ := h.db.ListAuthorities()
	tickets, _ := h.db.ListAllTickets()
	_ = users
	response.OK(w, map[string]interface{}{
		"total_users":       total,
		"total_authorities": len(authorities),
		"total_tickets":     len(tickets),
	})
}

// generateID produces a short random hex ID for replies, etc.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
