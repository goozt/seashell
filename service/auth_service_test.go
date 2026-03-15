package service_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/service"
	"github.com/goozt/seashell/store"
)

func tempDB(t *testing.T) *store.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "seashell-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	db, err := store.Open(dir, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(dir)
	})
	return db
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := service.HashPassword("mysecret123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := service.CheckPassword(hash, "mysecret123"); err != nil {
		t.Errorf("correct password rejected: %v", err)
	}
	if err := service.CheckPassword(hash, "wrongpassword"); err == nil {
		t.Error("wrong password accepted")
	}
}

func TestIssueAndValidateAccessToken(t *testing.T) {
	db := tempDB(t)
	svc := service.NewAuthService(db, "test-secret-key", 15, 7)

	u := &model.User{
		ID:            uuid.New().String(),
		Username:      "alice",
		Email:         "alice@example.com",
		Role:          model.RoleUser,
		AuthorityID:   "auth-1",
		AuthorityRole: model.AuthorityRoleOwner,
		Active:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := db.SaveUser(u); err != nil {
		t.Fatalf("save user: %v", err)
	}

	pair, err := svc.IssueTokenPair(u)
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.UserID != u.ID {
		t.Errorf("claims.UserID = %s, want %s", claims.UserID, u.ID)
	}
	if claims.Role != model.RoleUser {
		t.Errorf("claims.Role = %s, want %s", claims.Role, model.RoleUser)
	}
	if claims.AuthorityID != "auth-1" {
		t.Errorf("claims.AuthorityID = %s, want auth-1", claims.AuthorityID)
	}
}

func TestRefreshTokenFlow(t *testing.T) {
	db := tempDB(t)
	svc := service.NewAuthService(db, "test-secret", 15, 7)

	u := &model.User{
		ID:        uuid.New().String(),
		Username:  "bob",
		Email:     "bob@test.com",
		Role:      model.RoleUser,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = db.SaveUser(u)

	pair, err := svc.IssueTokenPair(u)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// Validate refresh token returns the user.
	got, err := svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("validate refresh: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("got.ID = %s, want %s", got.ID, u.ID)
	}

	// Revoke refresh token.
	if err := svc.RevokeRefreshToken(pair.RefreshToken); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Validate after revoke should fail.
	if _, err := svc.ValidateRefreshToken(pair.RefreshToken); err == nil {
		t.Error("expected error after revoke, got nil")
	}
}

func TestSeedFromFile(t *testing.T) {
	db := tempDB(t)
	svc := service.NewAuthService(db, "secret", 15, 7)

	// Write a temporary db_init.json.
	dir := t.TempDir()
	seedPath := dir + "/db_init.json"
	content := `{
		"admins": [{"username":"Kerala Admin","email":"kerala@nikz.in","password":"secretpassword"}],
		"users":  [{"username":"Alice","email":"alice@nikz.in","password":"secretpassword"},
		           {"username":"Nikz","email":"me@nikz.in","password":"secretpassword"}]
	}`
	if err := os.WriteFile(seedPath, []byte(content), 0600); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	if err := svc.SeedFromFile(seedPath); err != nil {
		t.Fatalf("SeedFromFile: %v", err)
	}

	tests := []struct {
		email    string
		password string
		role     string
	}{
		{"kerala@nikz.in", "secretpassword", "admin"},
		{"alice@nikz.in", "secretpassword", "user"},
		{"me@nikz.in", "secretpassword", "user"},
	}
	for _, tt := range tests {
		u, err := db.GetUserByEmail(tt.email)
		if err != nil {
			t.Errorf("user %s not found: %v", tt.email, err)
			continue
		}
		if u.Role != tt.role {
			t.Errorf("%s: role = %s, want %s", tt.email, u.Role, tt.role)
		}
		if !u.Active {
			t.Errorf("%s: expected active=true", tt.email)
		}
		if err := service.CheckPassword(u.PasswordHash, tt.password); err != nil {
			t.Errorf("%s: password check failed: %v", tt.email, err)
		}
	}

	// Idempotent — calling again should not create duplicates.
	if err := svc.SeedFromFile(seedPath); err != nil {
		t.Fatalf("second SeedFromFile: %v", err)
	}
	_, total, _ := db.ListUsers(1, 100)
	if total != 3 {
		t.Errorf("expected 3 users after second seed, got %d", total)
	}

	// Missing file should be a no-op.
	if err := svc.SeedFromFile(dir + "/nonexistent.json"); err != nil {
		t.Errorf("missing file should not error: %v", err)
	}
}

func TestBootstrapSuperAdmin(t *testing.T) {
	db := tempDB(t)
	svc := service.NewAuthService(db, "secret", 15, 7)

	if err := svc.BootstrapSuperAdmin("superpassword", "super@test.com"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Should be idempotent.
	if err := svc.BootstrapSuperAdmin("superpassword", "super@test.com"); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}

	if !db.SuperAdminExists() {
		t.Error("superadmin should exist after bootstrap")
	}

	// Verify login works.
	u, err := db.GetUserByEmail("super@test.com")
	if err != nil {
		t.Fatalf("get superadmin: %v", err)
	}
	if u.Role != model.RoleSuperAdmin {
		t.Errorf("role = %s, want superadmin", u.Role)
	}
	if err := service.CheckPassword(u.PasswordHash, "superpassword"); err != nil {
		t.Errorf("password check failed: %v", err)
	}
}
