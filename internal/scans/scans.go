// Package scans implements the InBody scan feature: HTTP handlers, DB
// queries, and its own migrations. CRUD endpoints land in Phase 3.
package scans

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Register mounts the scans feature's routes onto r. Routes here are assumed
// to already be behind the shared auth middleware.
func Register(r chi.Router, db *pgxpool.Pool) {
	r.Route("/api/scans", func(r chi.Router) {
		r.Get("/", notImplemented)
	})
	r.Route("/api/goals", func(r chi.Router) {
		r.Get("/", notImplemented)
	})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented yet", http.StatusNotImplemented)
}
