package api

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/goozt/seashell/api/handlers"
	apimw "github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/node"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// Start bootstraps the API server: opens databases, seeds superadmin, and begins listening.
func Start(cfg *config.Config) error {
	// Parse key encryption key (nil in dev mode — no encryption).
	kek, err := store.ParseKEK(cfg.KeyEncryptionKey)
	if err != nil {
		return fmt.Errorf("key encryption key: %w", err)
	}

	// Open API database.
	db, err := store.Open(cfg.DBPath+"/api", kek)
	if err != nil {
		return fmt.Errorf("open api db: %w", err)
	}
	defer db.Close()

	// Migrate private keys to encrypted format (idempotent).
	if err := db.MigratePrivKeysToEncrypted(); err != nil {
		return fmt.Errorf("migrate privkeys: %w", err)
	}

	// Derive HMAC key for KYC dedup indexes from KEK (or a fixed dev key).
	hmacKey := deriveHMACKey(kek)

	// Open archive store.
	archivePath := cfg.ArchivePath
	if archivePath == "" {
		archivePath = filepath.Join(cfg.DBPath, "archive")
	}
	archiveDB, err := store.OpenArchive(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer archiveDB.Close()

	// Build services.
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.AccessTokenMinutes, cfg.RefreshTokenDays)
	chainSvc := service.NewChainService(cfg.DBPath+"/blocks", db)
	valueSvc := service.NewValueService(db)
	nodeSvc := service.NewNodeService(db, chainSvc, cfg)
	quorumSvc := service.NewQuorumService(db, cfg, 0)
	idpClient := service.NewMockIdPClient(cfg.IdPBaseURL)
	kycSvc := service.NewKYCService(db, idpClient, nodeSvc.SelfID(), hmacKey)
	archivalSvc := service.NewArchivalService(chainSvc, archiveDB, cfg)

	// Initialize blockchain with genesis block if primary node and chain doesn't exist yet.
	if cfg.IsPrimary {
		if err := chainSvc.EnsureInitialized(); err != nil {
			return fmt.Errorf("init blockchain: %w", err)
		}
	}

	// Wire block-broadcast callback.
	chainSvc.SetOnBlock(nodeSvc.OnBlockMined)

	// Bootstrap superadmin if not present.
	if err := authSvc.BootstrapSuperAdmin(cfg.SuperAdminPassword, cfg.SuperAdminEmail); err != nil {
		return fmt.Errorf("bootstrap superadmin: %w", err)
	}

	// Seed initial admins/users from db_init.json.
	// Check next to the executable first, then fall back to the working directory
	// (so `go run .` also picks it up during development).
	{
		seedFile := "db_init.json" // working directory default
		if exe, err := os.Executable(); err == nil {
			candidate := filepath.Join(filepath.Dir(exe), "db_init.json")
			if _, err := os.Stat(candidate); err == nil {
				seedFile = candidate
			}
		}
		if err := authSvc.SeedFromFile(seedFile); err != nil {
			return fmt.Errorf("seed from db_init.json: %w", err)
		}
	}

	// Bootstrap node identity (register self / wait for primary approval).
	if cfg.NodeURL != "" {
		if err := nodeSvc.Bootstrap(); err != nil {
			return fmt.Errorf("bootstrap node: %w", err)
		}
		defer nodeSvc.Stop()
	}

	// Load or create CA if this is the primary node.
	var ca *node.CA
	if cfg.IsPrimary && cfg.TLSEnabled {
		certDir := filepath.Dir(cfg.CertPath)
		ca, err = node.LoadOrCreateCA(certDir, kek)
		if err != nil {
			return fmt.Errorf("load/create CA: %w", err)
		}
		fmt.Println("CA loaded/created at", certDir)
	}

	// Start nightly archival worker.
	archivalSvc.Start()

	// Build WebSocket hub.
	hub := service.NewWSHub()

	// Build push notification service.
	pushSvc := service.NewPushService(db, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDEmail)

	// Build handlers.
	authH := handlers.NewAuthHandler(db, authSvc)
	publicH := handlers.NewPublicHandler(chainSvc, valueSvc, db)
	userH := handlers.NewUserHandler(db, chainSvc, authSvc, hub, pushSvc)
	authorityOwnerH := handlers.NewAuthorityOwnerHandler(db, chainSvc, valueSvc)
	adminH := handlers.NewAdminHandler(db, cfg.DBPath+"/blocks", hub, pushSvc)
	superAdminH := handlers.NewSuperAdminHandler(db, authSvc)
	pushH := handlers.NewPushHandler(db, pushSvc, cfg)
	wsH := handlers.NewWSHandler(hub, authSvc)
	nodeH := handlers.NewNodeHandler(db, chainSvc, nodeSvc, quorumSvc, cfg.IsPrimary, cfg.NodeURL)
	kycH := handlers.NewKYCHandler(db, kycSvc)
	archiveH := handlers.NewArchiveHandler(archiveDB)
	_ = quorumSvc // used in future block creation flow

	// Build router.
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.StripSlashes)

	// Public routes (no auth required).
	r.Get("/api/v1/push/vapid-public-key", pushH.GetVAPIDPublicKey)
	r.Get("/api/v1/health", publicH.Health)
	r.Get("/api/v1/blocks", publicH.GetBlocks)
	r.Get("/api/v1/blocks/{hash}", publicH.GetBlock)
	r.Get("/api/v1/value", publicH.GetValue)

	// WebSocket (auth handled inside handler via query param token).
	r.Get("/api/v1/ws", wsH.ServeWS)

	// Node registration (called by new nodes wishing to join).
	r.Post("/api/v1/nodes/register", nodeH.RegisterNode)

	// Auth routes.
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/logout", authH.Logout)
		r.With(apimw.JWT(authSvc)).Get("/me", authH.Me)
	})

	// User routes (any authenticated user).
	r.Route("/api/v1/user", func(r chi.Router) {
		r.Use(apimw.JWT(authSvc))
		r.Get("/me", userH.GetMe)
		r.Put("/me", userH.UpdateMe)
		r.Get("/wallet", userH.GetWallet)
		r.Post("/wallet", userH.CreateWallet)
		r.Get("/transactions", userH.GetTransactions)
		r.Post("/transactions", userH.CreateTransaction)
		r.Get("/authority", userH.GetUserAuthority)
		r.Post("/authority", userH.CreateAuthorityRequest)
		r.Post("/authority/join", userH.JoinAuthority)
		r.Get("/tickets", userH.GetMyTickets)
		r.Post("/tickets", userH.CreateTicket)
		r.Get("/tickets/{id}", userH.GetTicket)
		r.Post("/tickets/{id}/reply", userH.ReplyToTicket)
		// Push notification subscription.
		r.Post("/push/subscribe", pushH.Subscribe)
		r.Delete("/push/subscribe", pushH.Unsubscribe)
		// KYC / identity verification.
		r.Post("/kyc", kycH.InitiateKYC)
		r.Get("/kyc", kycH.GetMyKYC)
	})

	// Authority owner routes.
	r.Route("/api/v1/authority", func(r chi.Router) {
		r.Use(apimw.JWT(authSvc))
		r.Use(apimw.RequireAuthorityOwner)
		r.Get("/members", authorityOwnerH.GetMembers)
		r.Delete("/members/{userID}", authorityOwnerH.RemoveMember)
		r.Get("/invitations", authorityOwnerH.GetInvitations)
		r.Post("/invitations", authorityOwnerH.CreateInvitation)
		r.Delete("/invitations/{code}", authorityOwnerH.RevokeInvitation)
		r.Get("/transactions", authorityOwnerH.GetAuthorityTransactions)
		r.Get("/value", authorityOwnerH.GetAuthorityValue)
		r.Get("/stats", authorityOwnerH.GetAuthorityStats)
	})

	// Admin routes (admin + superadmin).
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(apimw.JWT(authSvc))
		r.Use(apimw.RequireRole("admin", "superadmin"))
		r.Get("/authority-requests", adminH.GetAuthorityRequests)
		r.Get("/authority-requests/{id}", adminH.GetAuthorityRequest)
		r.Post("/authority-requests/{id}/approve", adminH.ApproveAuthorityRequest)
		r.Post("/authority-requests/{id}/reject", adminH.RejectAuthorityRequest)
		// Node-level support tickets.
		r.Get("/tickets", adminH.GetTickets)
		r.Get("/tickets/{id}", adminH.GetTicketAdmin)
		r.Put("/tickets/{id}", adminH.UpdateTicket)
		r.Post("/tickets/{id}/reply", adminH.ReplyToTicketAdmin)
		r.Post("/tickets/{id}/escalate", adminH.EscalateTicketToRegional)
		// Regional-level tickets (visible to admin+superadmin, escalated from nodes).
		r.Get("/tickets/regional", adminH.GetTicketsRegional)
		r.Post("/tickets/{id}/escalate-super", adminH.EscalateTicketToSuper)
		r.Get("/authorities", adminH.GetAuthoritiesAdmin)
		r.Get("/users", adminH.GetUsersAdmin)
		r.Get("/stats", adminH.GetStatsAdmin)
		// Network node routes.
		r.Get("/nodes", nodeH.GetNodes)
		r.Get("/nodes/{id}", nodeH.GetNode)
		r.Get("/nodes/tier/{tier}", nodeH.GetNodesByTier)
		// KYC admin routes.
		r.Get("/kyc", kycH.ListKYCAdmin)
		r.Post("/kyc/{userID}/reject", kycH.RejectKYCAdmin)
		r.Get("/kyc/summary", kycH.KYCSummaryAdmin)
		// Archive routes.
		r.Get("/archive/stats", archiveH.GetArchiveStats)
		r.Get("/archive/blocks", archiveH.GetArchiveRange)
		r.Get("/archive/blocks/{height}", archiveH.GetArchiveBlock)
	})

	// SuperAdmin routes.
	r.Route("/api/v1/superadmin", func(r chi.Router) {
		r.Use(apimw.JWT(authSvc))
		r.Use(apimw.RequireRole("superadmin"))
		r.Get("/admins", superAdminH.GetAdmins)
		r.Post("/admins", superAdminH.CreateAdmin)
		r.Delete("/admins/{id}", superAdminH.DemoteAdmin)
		r.Post("/authorities/{id}/suspend", superAdminH.SuspendAuthority)
		r.Post("/authorities/{id}/reinstate", superAdminH.ReinstateAuthority)
		r.Get("/stats", superAdminH.GetStatsSuperAdmin)
		// Superadmin-level tickets (escalated from regional).
		r.Get("/tickets", adminH.GetTicketsSuperAdmin)
		r.Get("/tickets/{id}", adminH.GetTicketAdmin)
		r.Post("/tickets/{id}/reply", adminH.ReplyToTicketAdmin)
		r.Put("/tickets/{id}", adminH.UpdateTicket)
		// Node management (primary node only).
		r.Get("/node-requests", nodeH.GetNodeJoinRequests)
		r.Post("/node-requests/{id}/approve", nodeH.ApproveNodeJoinRequest)
		r.Post("/node-requests/{id}/reject", nodeH.RejectNodeJoinRequest)
		r.Post("/nodes/{id}/suspend", nodeH.SuspendNode)
		r.Post("/nodes/{id}/reinstate", nodeH.ReinstateNode)
		// KYC network summary (primary only).
		if cfg.IsPrimary {
			r.Get("/kyc/network", kycH.KYCSummaryAdmin)
		}
	})

	// P2P routes (authenticated by shared node secret, not JWT).
	r.Route("/p2p/v1", func(r chi.Router) {
		r.Use(apimw.RequireNodeSecret(cfg.NodeSecret))
		r.Post("/ping", nodeH.P2PPing)
		r.Get("/peers", nodeH.P2PGetPeers)
		r.Get("/chain/sync", nodeH.P2PGetChainSync)
		r.Post("/blocks", nodeH.P2PReceiveBlock)
		r.Get("/validators", nodeH.P2PGetValidators)
		r.Post("/validators", nodeH.P2PReceiveValidator)
		r.Post("/cosign", nodeH.P2PCoSign)
		r.Get("/kyc/summary", kycH.P2PKYCSummary)
		// Cert issuance (primary + regional nodes only).
		if ca != nil {
			r.Post("/cert-issue", handlers.NewCertIssueHandler(db, ca).Issue)
		}
	})

	addr := ":" + cfg.Port
	fmt.Printf("SeaShell API server listening on %s (tier=%s)\n", addr, cfg.NodeTier)
	return http.ListenAndServe(addr, r)
}

// deriveHMACKey creates a 32-byte HMAC key from the KEK (or a fixed dev key).
func deriveHMACKey(kek []byte) []byte {
	if len(kek) == 0 {
		// Dev mode: fixed key (not secret).
		sum := sha256.Sum256([]byte("seashell-dev-hmac-key"))
		return sum[:]
	}
	// Derive by hashing "hmac" || kek.
	sum := sha256.Sum256(append([]byte("hmac:"), kek...))
	return sum[:]
}
