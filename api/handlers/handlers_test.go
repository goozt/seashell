package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/goozt/seashell/api/handlers"
	apimw "github.com/goozt/seashell/api/middleware"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "seashell-handler-test-*")
	if err != nil {
		t.Fatalf("mkdirtemp: %v", err)
	}
	db, err := store.Open(dir, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

func mustJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func decodeResp(t *testing.T, body []byte) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, body)
	}
	return e
}

func TestRegisterAndLogin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	authSvc := service.NewAuthService(db, "test-secret", 15, 7)
	h := handlers.NewAuthHandler(db, authSvc)

	r := chi.NewRouter()
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)

	// Register.
	payload := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", mustJSON(t, payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201\nbody: %s", rr.Code, rr.Body.String())
	}
	env := decodeResp(t, rr.Body.Bytes())
	if !env.Success {
		t.Fatalf("register not success: %s", env.Error)
	}

	// Duplicate registration should conflict.
	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", mustJSON(t, payload))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusConflict {
		t.Errorf("duplicate register = %d, want 409", rr2.Code)
	}

	// Login with correct credentials.
	loginPayload := map[string]string{"email": "test@example.com", "password": "password123"}
	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", mustJSON(t, loginPayload))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200\nbody: %s", rr3.Code, rr3.Body.String())
	}

	// Login with wrong password.
	badPayload := map[string]string{"email": "test@example.com", "password": "wrongpass"}
	req4 := httptest.NewRequest(http.MethodPost, "/auth/login", mustJSON(t, badPayload))
	req4.Header.Set("Content-Type", "application/json")
	rr4 := httptest.NewRecorder()
	r.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusUnauthorized {
		t.Errorf("bad login = %d, want 401", rr4.Code)
	}
}

func TestJWTMiddleware(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	authSvc := service.NewAuthService(db, "test-secret-mw", 15, 7)

	u := &model.User{
		ID:        uuid.New().String(),
		Username:  "mwuser",
		Email:     "mw@test.com",
		Role:      model.RoleUser,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = db.SaveUser(u)

	pair, _ := authSvc.IssueTokenPair(u)

	r := chi.NewRouter()
	r.With(apimw.JWT(authSvc)).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	})

	// Without token.
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", rr1.Code)
	}

	// With valid token.
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("valid token = %d, want 200", rr2.Code)
	}

	// With bad token.
	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Bearer invalidtoken")
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusUnauthorized {
		t.Errorf("bad token = %d, want 401", rr3.Code)
	}
}

func TestRequireRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	authSvc := service.NewAuthService(db, "test-secret-role", 15, 7)

	admin := &model.User{
		ID:        uuid.New().String(),
		Username:  "admin1",
		Email:     "admin@test.com",
		Role:      model.RoleAdmin,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	normalUser := &model.User{
		ID:        uuid.New().String(),
		Username:  "normal1",
		Email:     "normal@test.com",
		Role:      model.RoleUser,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = db.SaveUser(admin)
	_ = db.SaveUser(normalUser)

	adminPair, _ := authSvc.IssueTokenPair(admin)
	userPair, _ := authSvc.IssueTokenPair(normalUser)

	r := chi.NewRouter()
	r.With(apimw.JWT(authSvc), apimw.RequireRole("admin", "superadmin")).
		Get("/admin-only", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

	// Admin can access.
	req1 := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req1.Header.Set("Authorization", "Bearer "+adminPair.AccessToken)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("admin access = %d, want 200", rr1.Code)
	}

	// Normal user is forbidden.
	req2 := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req2.Header.Set("Authorization", "Bearer "+userPair.AccessToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Errorf("user access = %d, want 403", rr2.Code)
	}
}

func TestHealth(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	authSvc := service.NewAuthService(db, "s", 15, 7)
	chainSvc := service.NewChainService("./testdb", db)
	valueSvc := service.NewValueService(db)
	h := handlers.NewPublicHandler(chainSvc, valueSvc, db)

	r := chi.NewRouter()
	r.Get("/health", h.Health)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("health = %d, want 200", rr.Code)
	}
	_ = authSvc // suppress unused
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     model.RegisterRequest
		wantErr bool
	}{
		{"valid", model.RegisterRequest{Username: "alice", Email: "alice@test.com", Password: "password123"}, false},
		{"short_username", model.RegisterRequest{Username: "al", Email: "a@b.com", Password: "password123"}, true},
		{"no_at_email", model.RegisterRequest{Username: "alice", Email: "invalid", Password: "password123"}, true},
		{"short_password", model.RegisterRequest{Username: "alice", Email: "a@b.com", Password: "short"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.req.Validate()
			if tc.wantErr && msg == "" {
				t.Error("expected validation error, got none")
			}
			if !tc.wantErr && msg != "" {
				t.Errorf("unexpected validation error: %s", msg)
			}
		})
	}
}
