package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/goozt/seashell/model"
	"github.com/goozt/seashell/store"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// TokenPair holds an access token and a refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AccessClaims are the JWT claims embedded in access tokens.
type AccessClaims struct {
	UserID        string `json:"uid"`
	Role          string `json:"role"`
	AuthorityID   string `json:"aid"`
	AuthorityRole string `json:"ar"`
	jwt.RegisteredClaims
}

// refreshRecord is stored in the DB to allow revocation.
type refreshRecord struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthService handles password hashing, JWT issuance, and superadmin bootstrap.
type AuthService struct {
	db                 *store.DB
	jwtSecret          []byte
	accessTokenMinutes int
	refreshTokenDays   int
}

// NewAuthService creates an AuthService.
func NewAuthService(db *store.DB, jwtSecret string, accessMins, refreshDays int) *AuthService {
	return &AuthService{
		db:                 db,
		jwtSecret:          []byte(jwtSecret),
		accessTokenMinutes: accessMins,
		refreshTokenDays:   refreshDays,
	}
}

// HashPassword bcrypt-hashes the plain-text password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

// CheckPassword returns nil if the plain-text password matches the hash.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// IssueTokenPair generates a new access + refresh token pair for a user.
func (s *AuthService) IssueTokenPair(u *model.User) (*TokenPair, error) {
	now := time.Now()

	// Access token
	accessClaims := AccessClaims{
		UserID:        u.ID,
		Role:          u.Role,
		AuthorityID:   u.AuthorityID,
		AuthorityRole: u.AuthorityRole,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.accessTokenMinutes) * time.Minute)),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token: random 32-byte hex string stored in DB
	tokenID, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}
	expiry := now.Add(time.Duration(s.refreshTokenDays) * 24 * time.Hour)
	rec := refreshRecord{UserID: u.ID, ExpiresAt: expiry}
	if err := s.db.SetRefreshToken(tokenID, rec.UserID, rec.ExpiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessStr, RefreshToken: tokenID}, nil
}

// ValidateAccessToken parses and validates a JWT access token, returning its claims.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// ValidateRefreshToken checks the DB for the token and returns the owning user.
func (s *AuthService) ValidateRefreshToken(tokenID string) (*model.User, error) {
	userID, expiry, err := s.db.GetRefreshToken(tokenID)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found")
	}
	if time.Now().After(expiry) {
		_ = s.db.DeleteRefreshToken(tokenID)
		return nil, fmt.Errorf("refresh token expired")
	}
	return s.db.GetUserByID(userID)
}

// RevokeRefreshToken removes a refresh token from the DB (logout).
func (s *AuthService) RevokeRefreshToken(tokenID string) error {
	return s.db.DeleteRefreshToken(tokenID)
}

// BootstrapSuperAdmin creates the superadmin account if none exists.
func (s *AuthService) BootstrapSuperAdmin(password, email string) error {
	if s.db.SuperAdminExists() {
		return nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash superadmin password: %w", err)
	}
	now := time.Now()
	u := &model.User{
		ID:           uuid.New().String(),
		Username:     "superadmin",
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleSuperAdmin,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return s.db.SaveUser(u)
}

// dbInitFile is the seed file loaded on first boot.
type dbInitFile struct {
	Admins []dbInitUser `json:"admins"`
	Users  []dbInitUser `json:"users"`
}

type dbInitUser struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// SeedFromFile reads db_init.json next to the executable and creates any
// admins/users that don't already exist. Safe to call on every startup —
// existing accounts are skipped.
func (s *AuthService) SeedFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file absent — nothing to do
		}
		return fmt.Errorf("read db_init.json: %w", err)
	}
	var init dbInitFile
	if err := json.Unmarshal(data, &init); err != nil {
		return fmt.Errorf("parse db_init.json: %w", err)
	}
	seed := func(u dbInitUser, role string) {
		if s.db.EmailExists(u.Email) || s.db.UsernameExists(u.Username) {
			return
		}
		hash, err := HashPassword(u.Password)
		if err != nil {
			log.Printf("db_init: skip %s: %v", u.Email, err)
			return
		}
		now := time.Now()
		user := &model.User{
			ID:           uuid.New().String(),
			Username:     u.Username,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Email:        u.Email,
			PasswordHash: hash,
			Role:         role,
			Active:       true,
			KYCStatus:    model.KYCUnverified,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.db.SaveUser(user); err != nil {
			log.Printf("db_init: save %s: %v", u.Email, err)
			return
		}
		log.Printf("db_init: created %s (%s)", u.Email, role)
	}
	for _, u := range init.Admins {
		seed(u, model.RoleAdmin)
	}
	for _, u := range init.Users {
		seed(u, model.RoleUser)
	}
	return nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
