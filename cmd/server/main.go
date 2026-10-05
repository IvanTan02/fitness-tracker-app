// Command server wires together the router, middleware, and feature
// registration. It contains no business logic of its own.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
	"github.com/ivantan02/fitness-tracker-app/internal/platform"
	"github.com/ivantan02/fitness-tracker-app/internal/profile"
	"github.com/ivantan02/fitness-tracker-app/internal/scans"
	appweb "github.com/ivantan02/fitness-tracker-app/web"
)

func main() {
	ctx := context.Background()

	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	db, err := platform.NewDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	verifier, err := auth.NewVerifier(ctx, cfg.SupabaseURL)
	if err != nil {
		log.Fatalf("setting up auth verifier: %v", err)
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

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
