// Package scans implements the compact manual body-composition tracking
// feature: HTTP handlers, DB queries, and its own migrations.
package scans

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Register mounts the scans feature's routes onto r. Routes here are assumed
// to already be behind the shared auth middleware.
func Register(r chi.Router, db *pgxpool.Pool) {
	r.Route("/api/scans", func(r chi.Router) {
		r.Get("/", listScansHandler(db))
		r.Post("/", createScanHandler(db))
		r.Delete("/{id}", deleteScanHandler(db))
	})
	r.Route("/api/goals", func(r chi.Router) {
		r.Get("/", getGoalsHandler(db))
		r.Put("/", putGoalsHandler(db))
	})
}
