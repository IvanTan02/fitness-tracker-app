package profile

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
)

func Register(r chi.Router, db *pgxpool.Pool, environment string) {
	r.Route("/api/profile", func(r chi.Router) {
		r.Get("/", getHandler(db, environment))
		r.Put("/", putHandler(db, environment))
	})
}

func getHandler(db *pgxpool.Pool, environment string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := Get(r.Context(), db, auth.UserID(r.Context()), environment)
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusOK, nil)
			return
		}
		if err != nil {
			log.Printf("getting profile for user %s: %v", auth.UserID(r.Context()), err)
			http.Error(w, "failed to get profile", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func putHandler(db *pgxpool.Pool, environment string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p Profile
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := validate(p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		out, err := Upsert(r.Context(), db, auth.UserID(r.Context()), environment, p)
		if err != nil {
			log.Printf("saving profile for user %s: %v", auth.UserID(r.Context()), err)
			http.Error(w, "failed to save profile", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func validate(p Profile) error {
	if p.HeightCM < 100 || p.HeightCM > 250 {
		return errors.New("height_cm must be between 100 and 250")
	}
	if p.Age < 13 || p.Age > 120 {
		return errors.New("age must be between 13 and 120")
	}
	if p.Gender != "male" && p.Gender != "female" && p.Gender != "other" && p.Gender != "prefer_not_to_say" {
		return errors.New("gender must be male, female, other, or prefer_not_to_say")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
