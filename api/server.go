package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/goozt/seashell/api/handlers"
	apimw "github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/config"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

// Start bootstraps the API server: opens databases, seeds superadmin, and begins listening.
func Start(cfg *config.Config) error {
	// Open API database.
	db, err := store.Open(cfg.DBPath + "/api")
	if err != nil {
		return fmt.Errorf("open api db: %w", err)
	}
	defer db.Close()

	// Build services.
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.AccessTokenMinutes, cfg.RefreshTokenDays)
	chainSvc := service.NewChainService(cfg.DBPath+"/blocks", db)
	valueSvc := service.NewValueService(db)

	// Bootstrap superadmin if not present.
	if err := authSvc.BootstrapSuperAdmin(cfg.SuperAdminPassword, cfg.SuperAdminEmail); err != nil {
		return fmt.Errorf("bootstrap superadmin: %w", err)
	}

	// Build handlers.
	authH := handlers.NewAuthHandler(db, authSvc)
	publicH := handlers.NewPublicHandler(chainSvc, valueSvc, db)
	userH := handlers.NewUserHandler(db, chainSvc, authSvc)
	authorityOwnerH := handlers.NewAuthorityOwnerHandler(db, chainSvc, valueSvc)
	adminH := handlers.NewAdminHandler(db, cfg.DBPath+"/blocks")
	superAdminH := handlers.NewSuperAdminHandler(db, authSvc)

	// Build router.
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.StripSlashes)

	// Public routes (no auth required).
	r.Get("/api/v1/health", publicH.Health)
	r.Get("/api/v1/blocks", publicH.GetBlocks)
	r.Get("/api/v1/blocks/{hash}", publicH.GetBlock)
	r.Get("/api/v1/value", publicH.GetValue)

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
		r.Get("/tickets", adminH.GetTickets)
		r.Get("/tickets/{id}", adminH.GetTicketAdmin)
		r.Put("/tickets/{id}", adminH.UpdateTicket)
		r.Post("/tickets/{id}/reply", adminH.ReplyToTicketAdmin)
		r.Get("/authorities", adminH.GetAuthoritiesAdmin)
		r.Get("/users", adminH.GetUsersAdmin)
		r.Get("/stats", adminH.GetStatsAdmin)
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
	})

	addr := ":" + cfg.Port
	fmt.Printf("SeaShell API server listening on %s\n", addr)
	return http.ListenAndServe(addr, r)
}
