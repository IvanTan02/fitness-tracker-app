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
	AppEnvironment         string
	DatabaseURL            string
	SupabaseURL            string
	SupabasePublishableKey string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (Config, error) {
	cfg := Config{
		Port:                   getEnvDefault("PORT", "8080"),
		AppEnvironment:         os.Getenv("APP_ENV"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabasePublishableKey: os.Getenv("SUPABASE_PUBLISHABLE_KEY"),
	}

	required := map[string]string{
		"APP_ENV":                  cfg.AppEnvironment,
		"DATABASE_URL":             cfg.DatabaseURL,
		"SUPABASE_URL":             cfg.SupabaseURL,
		"SUPABASE_PUBLISHABLE_KEY": cfg.SupabasePublishableKey,
	}
	for name, val := range required {
		if val == "" {
			return Config{}, fmt.Errorf("missing required env var %s", name)
		}
	}
	if cfg.AppEnvironment != "development" && cfg.AppEnvironment != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production")
	}

	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
