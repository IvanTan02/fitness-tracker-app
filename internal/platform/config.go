// Package platform holds shared infrastructure used by every feature:
// configuration and the Postgres pool.
package platform

import (
	"fmt"
	"os"
)

// Config holds all required environment configuration. Fail fast on startup
// if anything required is missing, rather than discovering it at request time.
type Config struct {
	Port                   string
	DatabaseURL            string
	SupabaseURL            string
	SupabasePublishableKey string
	SupabaseSecretKey      string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (Config, error) {
	cfg := Config{
		Port:                   getEnvDefault("PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabasePublishableKey: os.Getenv("SUPABASE_PUBLISHABLE_KEY"),
		SupabaseSecretKey:      os.Getenv("SUPABASE_SECRET_KEY"),
	}

	required := map[string]string{
		"DATABASE_URL":             cfg.DatabaseURL,
		"SUPABASE_URL":             cfg.SupabaseURL,
		"SUPABASE_PUBLISHABLE_KEY": cfg.SupabasePublishableKey,
		"SUPABASE_SECRET_KEY":      cfg.SupabaseSecretKey,
	}
	for name, val := range required {
		if val == "" {
			return Config{}, fmt.Errorf("missing required env var %s", name)
		}
	}

	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
