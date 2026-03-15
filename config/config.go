package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port               string
	JWTSecret          string
	SuperAdminPassword string
	SuperAdminEmail    string
	DBPath             string // base path; blockchain uses DBPath/blocks, API uses DBPath/api
	AccessTokenMinutes int
	RefreshTokenDays   int

	// Node / P2P configuration.
	// NodeURL is this node's publicly reachable base URL (e.g. https://mynode.example.com).
	NodeURL string
	// PrimaryNodeURL is the URL of the primary (genesis) node.
	// Empty when this node IS the primary node.
	PrimaryNodeURL string
	// IsPrimary indicates this is the genesis/primary node that owns the node registry.
	IsPrimary bool
	// NodeSecret is a shared secret used to authenticate P2P requests between nodes.
	// All nodes in the network must share the same value.
	NodeSecret string
}

// Load reads configuration from environment variables with sensible defaults.
// It panics if required variables are missing.
func Load() *Config {
	secret := getEnv("JWT_SECRET", "")
	if secret == "" {
		secret = "seashell-change-me-in-production-jwt-secret-key"
		fmt.Fprintln(os.Stderr, "WARNING: JWT_SECRET not set, using insecure default")
	}

	superAdminPw := getEnv("SUPERADMIN_PASSWORD", "")
	if superAdminPw == "" {
		panic("SUPERADMIN_PASSWORD environment variable must be set")
	}

	nodeSecret := getEnv("NODE_SECRET", "")
	if nodeSecret == "" {
		nodeSecret = "seashell-change-me-node-secret"
		fmt.Fprintln(os.Stderr, "WARNING: NODE_SECRET not set, using insecure default")
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          secret,
		SuperAdminPassword: superAdminPw,
		SuperAdminEmail:    getEnv("SUPERADMIN_EMAIL", "superadmin@seashell.local"),
		DBPath:             getEnv("DB_PATH", "./db"),
		AccessTokenMinutes: getEnvInt("ACCESS_TOKEN_MINUTES", 15),
		RefreshTokenDays:   getEnvInt("REFRESH_TOKEN_DAYS", 7),

		NodeURL:        getEnv("NODE_URL", ""),
		PrimaryNodeURL: getEnv("PRIMARY_NODE_URL", ""),
		IsPrimary:      getEnvBool("IS_PRIMARY", false),
		NodeSecret:     nodeSecret,
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

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
