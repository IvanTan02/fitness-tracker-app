package scans

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
)

const dateLayout = "2006-01-02"

func listScansHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		scans, err := ListScans(r.Context(), db, userID)
		if err != nil {
			log.Printf("listing scans: %v", err)
			http.Error(w, "failed to list scans", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, scans)
	}
}

// createScanRequest mirrors NewScan but as JSON-decodable pointer fields, so
// omitted fields stay nil rather than becoming zero values.
type createScanRequest struct {
	Date              string   `json:"date"`
	Weight            *float64 `json:"weight"`
	BodyFat           *float64 `json:"body_fat"`
	LeanBodyMass      *float64 `json:"lean_body_mass"`
	BodyFatMass       *float64 `json:"body_fat_mass"`
	SMM               *float64 `json:"smm"`
	VisceralFat       *float64 `json:"visceral_fat"`
	BMR               *float64 `json:"bmr"`
	TEE               *float64 `json:"tee"`
	LeanLeftArm       *float64 `json:"lean_left_arm"`
	LeanRightArm      *float64 `json:"lean_right_arm"`
	LeanTrunk         *float64 `json:"lean_trunk"`
	LeanLeftLeg       *float64 `json:"lean_left_leg"`
	LeanRightLeg      *float64 `json:"lean_right_leg"`
	FatLeftArm        *float64 `json:"fat_left_arm"`
	FatRightArm       *float64 `json:"fat_right_arm"`
	FatTrunk          *float64 `json:"fat_trunk"`
	FatLeftLeg        *float64 `json:"fat_left_leg"`
	FatRightLeg       *float64 `json:"fat_right_leg"`
	Notes             *string  `json:"notes"`
	ReportPhotoPath   *string  `json:"report_photo_path"`
	ProgressPhotoPath *string  `json:"progress_photo_path"`
}

func createScanHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		var req createScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if err := validateScanRequest(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		scan, err := CreateScan(r.Context(), db, userID, NewScan{
			Date:              req.Date,
			Weight:            req.Weight,
			BodyFat:           req.BodyFat,
			LeanBodyMass:      req.LeanBodyMass,
			BodyFatMass:       req.BodyFatMass,
			SMM:               req.SMM,
			VisceralFat:       req.VisceralFat,
			BMR:               req.BMR,
			TEE:               req.TEE,
			LeanLeftArm:       req.LeanLeftArm,
			LeanRightArm:      req.LeanRightArm,
			LeanTrunk:         req.LeanTrunk,
			LeanLeftLeg:       req.LeanLeftLeg,
			LeanRightLeg:      req.LeanRightLeg,
			FatLeftArm:        req.FatLeftArm,
			FatRightArm:       req.FatRightArm,
			FatTrunk:          req.FatTrunk,
			FatLeftLeg:        req.FatLeftLeg,
			FatRightLeg:       req.FatRightLeg,
			Notes:             req.Notes,
			ReportPhotoPath:   req.ReportPhotoPath,
			ProgressPhotoPath: req.ProgressPhotoPath,
		})
		if err != nil {
			log.Printf("creating scan: %v", err)
			http.Error(w, "failed to create scan", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, scan)
	}
}

func deleteScanHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())
		scanID := chi.URLParam(r, "id")

		deleted, err := DeleteScan(r.Context(), db, userID, scanID)
		if err != nil {
			log.Printf("deleting scan: %v", err)
			http.Error(w, "failed to delete scan", http.StatusInternalServerError)
			return
		}
		if !deleted {
			http.Error(w, "scan not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getGoalsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		goals, err := GetGoals(r.Context(), db, userID)
		if errors.Is(err, ErrGoalsNotFound) {
			writeJSON(w, http.StatusOK, Goals{})
			return
		}
		if err != nil {
			log.Printf("getting goals: %v", err)
			http.Error(w, "failed to get goals", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, goals)
	}
}

func putGoalsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		var req Goals
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if err := validateGoalRanges(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		goals, err := UpsertGoals(r.Context(), db, userID, req)
		if err != nil {
			log.Printf("upserting goals: %v", err)
			http.Error(w, "failed to save goals", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, goals)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func validateScanRequest(req createScanRequest) error {
	if req.Date == "" {
		return errors.New("date is required")
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		return errors.New("date must be in YYYY-MM-DD format")
	}

	ranges := map[string]struct {
		val      *float64
		min, max float64
	}{
		"weight":         {req.Weight, 20, 400},
		"body_fat":       {req.BodyFat, 1, 70},
		"lean_body_mass": {req.LeanBodyMass, 5, 200},
		"body_fat_mass":  {req.BodyFatMass, 0, 200},
		"smm":            {req.SMM, 5, 100},
		"visceral_fat":   {req.VisceralFat, 0, 60},
		"bmr":            {req.BMR, 500, 5000},
		"tee":            {req.TEE, 500, 8000},
		"lean_left_arm":  {req.LeanLeftArm, 0, 30},
		"lean_right_arm": {req.LeanRightArm, 0, 30},
		"lean_trunk":     {req.LeanTrunk, 0, 60},
		"lean_left_leg":  {req.LeanLeftLeg, 0, 40},
		"lean_right_leg": {req.LeanRightLeg, 0, 40},
		"fat_left_arm":   {req.FatLeftArm, 0, 20},
		"fat_right_arm":  {req.FatRightArm, 0, 20},
		"fat_trunk":      {req.FatTrunk, 0, 60},
		"fat_left_leg":   {req.FatLeftLeg, 0, 30},
		"fat_right_leg":  {req.FatRightLeg, 0, 30},
	}

	for name, r := range ranges {
		if r.val == nil {
			continue
		}
		if *r.val < r.min || *r.val > r.max {
			return errFieldRange(name, r.min, r.max)
		}
	}

	return nil
}

func validateGoalRanges(g Goals) error {
	ranges := map[string]struct {
		val      *float64
		min, max float64
	}{
		"weight":   {g.Weight, 20, 400},
		"body_fat": {g.BodyFat, 1, 70},
		"smm":      {g.SMM, 5, 100},
	}

	for name, r := range ranges {
		if r.val == nil {
			continue
		}
		if *r.val < r.min || *r.val > r.max {
			return errFieldRange(name, r.min, r.max)
		}
	}

	return nil
}

func errFieldRange(field string, min, max float64) error {
	return fmt.Errorf("%s must be between %g and %g", field, min, max)
}
