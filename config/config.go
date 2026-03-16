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

	// KeyEncryptionKey is a 32-byte hex-encoded AES-256-GCM key used to encrypt
	// private keys at rest. Required in production; optional in development.
	KeyEncryptionKey string // ENV: KEY_ENCRYPTION_KEY

	// TLSEnabled enables mutual TLS for P2P communication.
	TLSEnabled bool // ENV: TLS_ENABLED, default false

	// CertPath is the path to this node's TLS certificate file.
	CertPath string // ENV: CERT_PATH, default ./certs/node.crt

	// CACertPath is the path to the CA certificate chain file.
	CACertPath string // ENV: CA_CERT_PATH, default ./certs/ca.crt

	// NodeTier identifies this node's role in the network.
	// Values: "primary" | "branch". Default: "branch".
	NodeTier string // ENV: NODE_TIER

	// ArchivePath is the directory where archived blockchain blocks are stored.
	// Default: $DB_PATH/archive
	ArchivePath string // ENV: ARCHIVE_PATH

	// ArchiveRetentionYears is the number of years to keep blocks in the live chain.
	// Blocks older than this are moved to the archive. Default: 6.
	ArchiveRetentionYears int // ENV: ARCHIVE_RETENTION_YEARS

	// IdPBaseURL is the base URL of the Identity Provider service for National ID verification.
	// Default: http://localhost:9999
	IdPBaseURL string // ENV: IDP_BASE_URL

	// VAPID keys for Web Push notifications.
	VAPIDPublicKey  string // ENV: VAPID_PUBLIC_KEY
	VAPIDPrivateKey string // ENV: VAPID_PRIVATE_KEY
	VAPIDEmail      string // ENV: VAPID_EMAIL, default "mailto:admin@seashell.local"
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

	isPrimaryNode := getEnvBool("IS_PRIMARY", false)
	if isPrimaryNode {
		fmt.Println("Running on PRIMARY Node")
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

		KeyEncryptionKey:      getEnv("DB_ENCRYPTION_KEY", ""),
		TLSEnabled:            getEnvBool("TLS_ENABLED", false),
		CertPath:              getEnv("CERT_PATH", "./certs/node.crt"),
		CACertPath:            getEnv("CA_CERT_PATH", "./certs/ca.crt"),
		NodeTier:    getEnv("NODE_TIER", map[bool]string{true: "primary", false: "branch"}[isPrimaryNode]),
		ArchivePath:           getEnv("ARCHIVE_PATH", ""),
		ArchiveRetentionYears: getEnvInt("ARCHIVE_RETENTION_YEARS", 6),
		IdPBaseURL:            getEnv("IDP_BASE_URL", "http://localhost:9999"),
		VAPIDPublicKey:        getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:       getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDEmail:            getEnv("VAPID_EMAIL", "mailto:admin@seashell.local"),
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
