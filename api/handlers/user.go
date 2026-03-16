package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// UserHandler serves /api/v1/user/* routes (any authenticated user).
type UserHandler struct {
	db        *store.DB
	chainSvc  *service.ChainService
	authSvc   *service.AuthService
	hub       *service.WSHub
	pushSvc   *service.PushService
	verifySvc *service.VerificationService
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(db *store.DB, chainSvc *service.ChainService, authSvc *service.AuthService, hub *service.WSHub, pushSvc *service.PushService, verifySvc *service.VerificationService) *UserHandler {
	return &UserHandler{db: db, chainSvc: chainSvc, authSvc: authSvc, hub: hub, pushSvc: pushSvc, verifySvc: verifySvc}
}

func (h *UserHandler) currentUser(r *http.Request) (*model.User, bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return nil, false
	}
	u, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		return nil, false
	}
	return u, true
}

// GetMe handles GET /api/v1/user/me.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	response.OK(w, u.SafeCopy())
}

// UpdateMe handles PUT /api/v1/user/me.
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if req.Username != "" && req.Username != u.Username {
		if h.db.UsernameExists(req.Username) {
			response.Conflict(w, "username already taken")
			return
		}
		u.Username = req.Username
	}
	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}
	if req.LastName != "" {
		u.LastName = req.LastName
	}
	if req.Email != "" && req.Email != u.Email {
		if h.db.EmailExists(req.Email) {
			response.Conflict(w, "email already registered")
			return
		}
		u.Email = req.Email
	}
	if req.Password != "" {
		if len(req.Password) < 8 {
			response.BadRequest(w, "password must be at least 8 characters")
			return
		}
		hash, err := service.HashPassword(req.Password)
		if err != nil {
			response.InternalError(w, "could not hash password")
			return
		}
		u.PasswordHash = hash
	}
	u.UpdatedAt = time.Now()
	if err := h.db.SaveUser(u); err != nil {
		response.InternalError(w, "could not save user")
		return
	}
	response.OK(w, u.SafeCopy())
}

// GetWallet handles GET /api/v1/user/wallet.
func (h *UserHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if u.WalletAddress == "" {
		response.NotFound(w, "no wallet; POST /api/v1/user/wallet to create one")
		return
	}
	balance, err := h.chainSvc.GetBalance(u.WalletAddress)
	if err != nil {
		response.InternalError(w, "could not query balance")
		return
	}
	response.OK(w, map[string]interface{}{
		"address": u.WalletAddress,
		"balance": balance,
	})
}

// CreateWallet handles POST /api/v1/user/wallet (idempotent).
func (h *UserHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	// Enforce verification gate for authority-affiliated users.
	if u.IsAffiliated() {
		verified, err := h.verifySvc.IsUserVerified(u.ID, u.AuthorityID)
		if err != nil {
			response.InternalError(w, "verification check failed")
			return
		}
		if !verified {
			response.Error(w, http.StatusForbidden, "verification required before creating a wallet")
			return
		}
	}
	if u.WalletAddress != "" {
		response.OK(w, map[string]string{"address": u.WalletAddress, "message": "wallet already exists"})
		return
	}
	addr, _, err := h.chainSvc.CreateWalletForUser(u.ID)
	if err != nil {
		response.InternalError(w, "could not create wallet")
		return
	}
	u.WalletAddress = addr
	u.UpdatedAt = time.Now()
	if err := h.db.SaveUser(u); err != nil {
		response.InternalError(w, "could not save wallet address")
		return
	}

	// Issue an authority-signed identity proof and record a UserVerified on-chain event.
	if u.IsAffiliated() {
		proof, proofErr := h.verifySvc.IssueIdentityProof(u.ID, u.AuthorityID, addr)
		if proofErr == nil && proof != nil {
			if authority, err := h.db.GetAuthorityByID(u.AuthorityID); err == nil {
				authorityName := authority.Name
				userIDHash := proof.UserIDHash
				go func() {
					_ = h.chainSvc.RecordUserVerified(u.AuthorityID, authorityName, userIDHash, addr)
				}()
			}
		}
	}

	response.Created(w, map[string]string{"address": addr})
}

// GetTransactions handles GET /api/v1/user/transactions.
func (h *UserHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if u.WalletAddress == "" {
		response.OK(w, []interface{}{})
		return
	}
	txs, err := h.chainSvc.GetTransactionsForAddress(u.WalletAddress)
	if err != nil {
		response.InternalError(w, "could not load transactions")
		return
	}
	response.OK(w, txs)
}

// CreateTransaction handles POST /api/v1/user/transactions.
func (h *UserHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if !u.IsAffiliated() {
		response.Forbidden(w)
		return
	}
	authority, err := h.db.GetAuthorityByID(u.AuthorityID)
	if err != nil || !authority.IsActive() {
		response.Error(w, http.StatusForbidden, "your authority is not active")
		return
	}
	if u.WalletAddress == "" {
		response.BadRequest(w, "create a wallet first via POST /api/v1/user/wallet")
		return
	}

	var req struct {
		ToAddress string `json:"to_address"`
		Amount    int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if req.Amount <= 0 {
		response.BadRequest(w, "amount must be positive")
		return
	}

	privKey, pubKey, err := h.chainSvc.LoadUserPrivKey(u.ID)
	if err != nil {
		response.InternalError(w, "could not load wallet key")
		return
	}

	proof, _ := h.db.GetIdentityProof(u.ID) // nil if not yet issued; still valid, proof is optional

	result, err := h.chainSvc.CreateAndSubmit(u.WalletAddress, req.ToAddress, req.Amount, privKey, pubKey, proof)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(w, result)
}

// GetUserAuthority handles GET /api/v1/user/authority.
func (h *UserHandler) GetUserAuthority(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if !u.IsAffiliated() {
		response.NotFound(w, "not affiliated with any authority")
		return
	}
	authority, err := h.db.GetAuthorityByID(u.AuthorityID)
	if err != nil {
		response.NotFound(w, "authority not found")
		return
	}
	response.OK(w, authority)
}

// CreateAuthorityRequest handles POST /api/v1/user/authority.
func (h *UserHandler) CreateAuthorityRequest(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if u.IsAffiliated() {
		response.Conflict(w, "already affiliated with an authority")
		return
	}
	var req model.CreateAuthorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if len(req.Name) < 3 {
		response.BadRequest(w, "authority name must be at least 3 characters")
		return
	}

	authority := &model.Authority{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Description:  req.Description,
		OwnerID:      u.ID,
		Status:       model.AuthorityStatusPending,
		BasePrice:    1.0,
		SensitivityK: 0.1,
		CreatedAt:    time.Now(),
	}
	if err := h.db.SaveAuthority(authority); err != nil {
		response.InternalError(w, "could not save authority request")
		return
	}
	response.Created(w, authority)
}

// JoinAuthority handles POST /api/v1/user/authority/join.
func (h *UserHandler) JoinAuthority(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	if u.IsAffiliated() {
		response.Conflict(w, "already affiliated with an authority")
		return
	}
	var req model.JoinAuthorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		response.BadRequest(w, "invitation code required")
		return
	}
	inv, err := h.db.GetInvitationByCode(req.Code)
	if err != nil || !inv.IsValid() {
		response.Error(w, http.StatusGone, "invitation code invalid or expired")
		return
	}
	authority, err := h.db.GetAuthorityByID(inv.AuthorityID)
	if err != nil || !authority.IsActive() {
		response.Error(w, http.StatusGone, "authority no longer active")
		return
	}

	inv.Used = true
	inv.UsedBy = u.ID
	_ = h.db.SaveInvitation(inv)

	u.AuthorityID = authority.ID
	u.AuthorityRole = model.AuthorityRoleMember
	u.UpdatedAt = time.Now()
	if err := h.db.SaveUser(u); err != nil {
		response.InternalError(w, "could not join authority")
		return
	}
	response.OK(w, map[string]interface{}{
		"authority":      authority,
		"authority_role": u.AuthorityRole,
	})
}

// GetMyTickets handles GET /api/v1/user/tickets.
func (h *UserHandler) GetMyTickets(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	tickets, err := h.db.ListTicketsByUser(u.ID)
	if err != nil {
		response.InternalError(w, "could not load tickets")
		return
	}
	response.OK(w, tickets)
}

// CreateTicket handles POST /api/v1/user/tickets.
func (h *UserHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	var req model.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON")
		return
	}
	if req.Title == "" {
		response.BadRequest(w, "title is required")
		return
	}
	now := time.Now()
	ticket := &model.Ticket{
		ID:          uuid.New().String(),
		AuthorityID: u.AuthorityID,
		CreatedBy:   u.ID,
		Title:       req.Title,
		Description: req.Description,
		Status:      model.TicketStatusOpen,
		EscalatedTo: model.TicketLevelNode,
		Replies:     []model.TicketReply{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not save ticket")
		return
	}
	response.Created(w, ticket)
}

// GetTicket handles GET /api/v1/user/tickets/{id}.
func (h *UserHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	if ticket.CreatedBy != u.ID && ticket.AuthorityID != u.AuthorityID {
		response.Forbidden(w)
		return
	}
	response.OK(w, ticket)
}

// ReplyToTicket handles POST /api/v1/user/tickets/{id}/reply.
func (h *UserHandler) ReplyToTicket(w http.ResponseWriter, r *http.Request) {
	u, ok := h.currentUser(r)
	if !ok {
		response.Unauthorized(w)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, err := h.db.GetTicketByID(id)
	if err != nil {
		response.NotFound(w, "ticket not found")
		return
	}
	if ticket.CreatedBy != u.ID && ticket.AuthorityID != u.AuthorityID {
		response.Forbidden(w)
		return
	}
	var req model.ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		response.BadRequest(w, "message is required")
		return
	}
	reply := model.TicketReply{
		ID:             uuid.New().String(),
		AuthorID:       u.ID,
		AuthorUsername: u.Username,
		AuthorRole:     u.Role,
		Message:        req.Message,
		CreatedAt:      time.Now(),
	}
	ticket.Replies = append(ticket.Replies, reply)
	ticket.UpdatedAt = time.Now()
	if err := h.db.SaveTicket(ticket); err != nil {
		response.InternalError(w, "could not save reply")
		return
	}

	// Notify all admins of this authority about the new user reply.
	if users, _, err := h.db.ListUsers(1, 1000); err == nil {
		for _, admin := range users {
			if (admin.Role == model.RoleAdmin || admin.Role == model.RoleSuperAdmin) && admin.AuthorityID == ticket.AuthorityID {
				h.hub.SendTo(admin.ID, model.WSTypeSupport, generateID(), model.SupportPayload{
					TicketID: ticket.ID,
					Reply:    reply,
				})
				h.hub.SendTo(admin.ID, model.WSTypeNotification, generateID(), model.NotificationPayload{
					Title: "User replied: " + ticket.Title,
					Body:  reply.Message,
					Link:  "/admin/tickets",
				})
				h.pushSvc.SendToUser(admin.ID, model.PushPayload{
					Title: "User replied: " + ticket.Title,
					Body:  reply.Message,
					Icon:  "/logo-icon.png",
					URL:   "/admin/tickets",
					Tag:   "ticket-" + ticket.ID,
				})
			}
		}
	}

	response.Created(w, ticket)
}
