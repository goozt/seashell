package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port                string
	JWTSecret           string
	SuperAdminPassword  string
	SuperAdminEmail     string
	DBPath              string // base path; blockchain uses DBPath/blocks, API uses DBPath/api
	AccessTokenMinutes  int
	RefreshTokenDays    int
}

// Load reads configuration from environment variables with sensible defaults.
// It panics if required variables are missing.
func Load() *Config {
	secret := getEnv("JWT_SECRET", "")
	if secret == "" {
		// Generate a warning but allow startup with a default (not for production)
		secret = "seashell-change-me-in-production-jwt-secret-key"
		fmt.Fprintln(os.Stderr, "WARNING: JWT_SECRET not set, using insecure default")
	}

	superAdminPw := getEnv("SUPERADMIN_PASSWORD", "")
	if superAdminPw == "" {
		panic("SUPERADMIN_PASSWORD environment variable must be set")
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          secret,
		SuperAdminPassword: superAdminPw,
		SuperAdminEmail:    getEnv("SUPERADMIN_EMAIL", "superadmin@seashell.local"),
		DBPath:             getEnv("DB_PATH", "./db"),
		AccessTokenMinutes: getEnvInt("ACCESS_TOKEN_MINUTES", 15),
		RefreshTokenDays:   getEnvInt("REFRESH_TOKEN_DAYS", 7),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
