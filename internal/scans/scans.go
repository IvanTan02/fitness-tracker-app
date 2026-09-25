// Package scans implements the body composition scan feature (Evolt 360
// scans): HTTP handlers, DB queries, and its own migrations. CRUD endpoints
// land in Phase 3.
package scans

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivantan02/fitness-tracker-app/internal/platform"
)

// maxExtractionsPerUserPerDay keeps one user from burning through Gemini's
// shared 250 req/day free-tier quota and locking out everyone else.
const maxExtractionsPerUserPerDay = 20

// Register mounts the scans feature's routes onto r. Routes here are assumed
// to already be behind the shared auth middleware.
func Register(r chi.Router, db *pgxpool.Pool, gemini *platform.GeminiClient) {
	limiter := newExtractRateLimiter(maxExtractionsPerUserPerDay)

	r.Route("/api/scans", func(r chi.Router) {
		r.Get("/", listScansHandler(db))
		r.Post("/", createScanHandler(db))
		r.Delete("/{id}", deleteScanHandler(db))
	})
	r.Route("/api/goals", func(r chi.Router) {
		r.Get("/", getGoalsHandler(db))
		r.Put("/", putGoalsHandler(db))
	})
	r.Route("/api/extract", func(r chi.Router) {
		r.Use(limiter.middleware)
		r.Post("/", extractHandler(gemini, limiter))
	})
}
