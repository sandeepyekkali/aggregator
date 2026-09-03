package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all heavily-typed, validated environment variables.
type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	FrontendOrigin string

	// Auth & Security
	SupabaseJWKSURL string

	// Plaid Integration (Basic Tier)
	PlaidClientID string
	PlaidSecret   string
	PlaidEnv      string

	// SnapTrade Integration (Premium Tier)
	SnapTradeClientID    string
	SnapTradeConsumerKey string
}

// Load reads environment variables and fails fast if required ones are missing.
func Load() (*Config, error) {
	// Attempt to load .env file.
	// We ignore the error because in production, there is no .env file;
	// variables are injected directly into the container by the orchestrator.
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:               getEnvOrDefault("APP_ENV", "development"),
		Port:                 getEnvOrDefault("PORT", "8080"),
		PlaidEnv:             getEnvOrDefault("PLAID_ENV", "sandbox"),
		FrontendOrigin:       getEnvOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		SupabaseJWKSURL:      os.Getenv("SUPABASE_JWKS_URL"),
		PlaidClientID:        os.Getenv("PLAID_CLIENT_ID"),
		PlaidSecret:          os.Getenv("PLAID_SECRET"),
		SnapTradeClientID:    os.Getenv("SNAPTRADE_CLIENT_ID"),
		SnapTradeConsumerKey: os.Getenv("SNAPTRADE_CONSUMER_KEY"),
	}

	// Fail-Fast Validations
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("missing required environment variable: DATABASE_URL")
	}
	if cfg.SupabaseJWKSURL == "" {
		return nil, fmt.Errorf("missing required environment variable: SUPABASE_JWKS_URL")
	}
	if cfg.PlaidClientID == "" {
		return nil, fmt.Errorf("missing required environment variable: PLAID_CLIENT_ID")
	}
	if cfg.PlaidSecret == "" {
		return nil, fmt.Errorf("missing required environment variable: PLAID_SECRET")
	}
	if cfg.SnapTradeClientID == "" {
		return nil, fmt.Errorf("missing required environment variable: SNAPTRADE_CLIENT_ID")
	}
	if cfg.SnapTradeConsumerKey == "" {
		return nil, fmt.Errorf("missing required environment variable: SNAPTRADE_CONSUMER_KEY")
	}

	return cfg, nil
}

// Helper to fallback to a default value if the env var is empty
func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
