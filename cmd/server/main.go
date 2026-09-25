// Command server wires together the router, middleware, and feature
// registration. It contains no business logic of its own.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
	"github.com/ivantan02/fitness-tracker-app/internal/platform"
	"github.com/ivantan02/fitness-tracker-app/internal/scans"
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

	r.Group(func(r chi.Router) {
		r.Use(verifier.Middleware())
		scans.Register(r, db)
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
