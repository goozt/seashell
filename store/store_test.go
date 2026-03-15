package store_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "seashell-store-test-*")
	if err != nil {
		t.Fatalf("mkdirtemp: %v", err)
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(dir)
	})
	return db
}

func newUser(role string) *model.User {
	return &model.User{
		ID:        uuid.New().String(),
		Username:  "user-" + uuid.New().String()[:8],
		Email:     uuid.New().String()[:8] + "@test.com",
		Role:      role,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestUserCRUD(t *testing.T) {
	db := openTestDB(t)

	u := newUser(model.RoleUser)
	u.PasswordHash = "hash"

	if err := db.SaveUser(u); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := db.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != u.Username {
		t.Errorf("username = %s, want %s", got.Username, u.Username)
	}

	byEmail, err := db.GetUserByEmail(u.Email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Errorf("email lookup id = %s, want %s", byEmail.ID, u.ID)
	}

	if !db.EmailExists(u.Email) {
		t.Error("EmailExists should return true")
	}
	if !db.UsernameExists(u.Username) {
		t.Error("UsernameExists should return true")
	}
	if db.EmailExists("nonexistent@test.com") {
		t.Error("EmailExists should return false for unknown email")
	}
}

func TestUserList(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 5; i++ {
		_ = db.SaveUser(newUser(model.RoleUser))
	}

	users, total, err := db.ListUsers(1, 3)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(users) != 3 {
		t.Errorf("page size = %d, want 3", len(users))
	}

	page2, total2, _ := db.ListUsers(2, 3)
	if total2 != 5 {
		t.Errorf("page2 total = %d, want 5", total2)
	}
	if len(page2) != 2 {
		t.Errorf("page2 size = %d, want 2", len(page2))
	}
}

func TestSuperAdminExists(t *testing.T) {
	db := openTestDB(t)
	if db.SuperAdminExists() {
		t.Error("should not exist initially")
	}
	sa := newUser(model.RoleSuperAdmin)
	_ = db.SaveUser(sa)
	if !db.SuperAdminExists() {
		t.Error("should exist after save")
	}
}

func TestAuthorityCRUD(t *testing.T) {
	db := openTestDB(t)

	a := &model.Authority{
		ID:          uuid.New().String(),
		Name:        "Test Authority",
		Description: "A test",
		OwnerID:     uuid.New().String(),
		Status:      model.AuthorityStatusPending,
		BasePrice:   100,
		SensitivityK: 0.1,
		CreatedAt:   time.Now(),
	}
	if err := db.SaveAuthority(a); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := db.GetAuthorityByID(a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != a.Name {
		t.Errorf("name = %s, want %s", got.Name, a.Name)
	}

	pending, err := db.ListPendingAuthorities()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("pending count = %d, want 1", len(pending))
	}

	a.Status = model.AuthorityStatusActive
	_ = db.SaveAuthority(a)
	active, _ := db.ListActiveAuthorities()
	if len(active) != 1 {
		t.Errorf("active count = %d, want 1", len(active))
	}
}

func TestInvitationCRUD(t *testing.T) {
	db := openTestDB(t)

	inv := &model.Invitation{
		Code:        "abc123",
		AuthorityID: "auth-1",
		CreatedBy:   "user-1",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	if err := db.SaveInvitation(inv); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := db.GetInvitationByCode("abc123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.IsValid() {
		t.Error("invitation should be valid")
	}

	// Mark used.
	got.Used = true
	_ = db.SaveInvitation(got)

	used, _ := db.GetInvitationByCode("abc123")
	if used.IsValid() {
		t.Error("used invitation should be invalid")
	}

	// Delete.
	if err := db.DeleteInvitation("abc123"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.GetInvitationByCode("abc123"); err == nil {
		t.Error("should be not found after delete")
	}
}

func TestTicketCRUD(t *testing.T) {
	db := openTestDB(t)

	ticket := &model.Ticket{
		ID:          uuid.New().String(),
		AuthorityID: "auth-1",
		CreatedBy:   "user-1",
		Title:       "Test ticket",
		Description: "Something is wrong",
		Status:      model.TicketStatusOpen,
		Replies:     []model.TicketReply{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.SaveTicket(ticket); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := db.GetTicketByID(ticket.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != ticket.Title {
		t.Errorf("title = %s, want %s", got.Title, ticket.Title)
	}

	// Add reply.
	got.Replies = append(got.Replies, model.TicketReply{
		ID:        "reply-1",
		AuthorID:  "admin-1",
		AuthorRole: "admin",
		Message:   "We are looking into it",
		CreatedAt: time.Now(),
	})
	got.Status = model.TicketStatusInProgress
	_ = db.SaveTicket(got)

	updated, _ := db.GetTicketByID(ticket.ID)
	if len(updated.Replies) != 1 {
		t.Errorf("replies count = %d, want 1", len(updated.Replies))
	}
	if updated.Status != model.TicketStatusInProgress {
		t.Errorf("status = %s, want in_progress", updated.Status)
	}

	// List by authority.
	byAuth, _ := db.ListTicketsByAuthority("auth-1")
	if len(byAuth) != 1 {
		t.Errorf("tickets by authority = %d, want 1", len(byAuth))
	}
}

func TestValueRecordCRUD(t *testing.T) {
	db := openTestDB(t)

	v := &model.ValueRecord{
		AuthorityID:       "auth-1",
		BlockHeight:       10,
		Price:             105.5,
		TransactionVolume: 500,
		CirculatingSupply: 1000,
		Velocity:          0.5,
		RecordedAt:        time.Now(),
	}
	if err := db.SaveValueRecord(v); err != nil {
		t.Fatalf("save: %v", err)
	}

	latest, err := db.GetLatestValue("auth-1")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if latest.Price != 105.5 {
		t.Errorf("price = %.1f, want 105.5", latest.Price)
	}
	if latest.BlockHeight != 10 {
		t.Errorf("height = %d, want 10", latest.BlockHeight)
	}

	// Add a newer record.
	v2 := &model.ValueRecord{
		AuthorityID: "auth-1",
		BlockHeight: 20,
		Price:       110.0,
		RecordedAt:  time.Now(),
	}
	_ = db.SaveValueRecord(v2)

	history, err := db.ListValueHistory("auth-1")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("history len = %d, want 2", len(history))
	}
	// Most recent first.
	if history[0].BlockHeight < history[1].BlockHeight {
		t.Error("history not sorted most-recent-first")
	}
}

func TestWalletPrivKeyStorage(t *testing.T) {
	db := openTestDB(t)

	dBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	userID := uuid.New().String()

	if err := db.StoreWalletPrivKey(userID, dBytes); err != nil {
		t.Fatalf("store: %v", err)
	}

	got, err := db.GetWalletPrivKey(userID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got) != len(dBytes) {
		t.Errorf("len = %d, want %d", len(got), len(dBytes))
	}
	for i, b := range got {
		if b != dBytes[i] {
			t.Errorf("byte[%d] = %d, want %d", i, b, dBytes[i])
		}
	}
}
