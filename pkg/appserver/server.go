// Package appserver assembles the HTTP application for both the local binary
// and serverless deployment targets.
package appserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
	"github.com/ivantan02/fitness-tracker-app/internal/platform"
	"github.com/ivantan02/fitness-tracker-app/internal/profile"
	"github.com/ivantan02/fitness-tracker-app/internal/scans"
	appweb "github.com/ivantan02/fitness-tracker-app/web"
)

// App owns the shared router and database pool.
type App struct {
	Handler http.Handler
	Port    string
	db      *pgxpool.Pool
}

// New loads configuration and constructs the complete HTTP application.
func New(ctx context.Context) (*App, error) {
	cfg, err := platform.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	db, err := platform.NewDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	verifier, err := auth.NewVerifier(ctx, cfg.SupabaseURL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("setting up auth verifier: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/health", healthHandler)
	r.Get("/api/config", configHandler(cfg))
	r.Group(func(r chi.Router) {
		r.Use(verifier.Middleware())
		scans.Register(r, db)
		profile.Register(r, db)
	})
	r.Handle("/*", appweb.Handler())

	return &App{Handler: r, Port: cfg.Port, db: db}, nil
}

// Close releases resources held by the application.
func (a *App) Close() {
	a.db.Close()
}

func configHandler(cfg platform.Config) http.HandlerFunc {
	type publicConfig struct {
		SupabaseURL            string `json:"supabase_url"`
		SupabasePublishableKey string `json:"supabase_publishable_key"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(publicConfig{
			SupabaseURL:            cfg.SupabaseURL,
			SupabasePublishableKey: cfg.SupabasePublishableKey,
		}); err != nil {
			log.Printf("writing public config: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Printf("writing health response: %v", err)
	}
}
