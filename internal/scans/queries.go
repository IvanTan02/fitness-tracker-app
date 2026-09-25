package scans

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Scan is one logged body composition scan.
type Scan struct {
	ID                string    `json:"id"`
	Date              string    `json:"date"`
	Weight            *float64  `json:"weight"`
	BodyFat           *float64  `json:"body_fat"`
	LeanBodyMass      *float64  `json:"lean_body_mass"`
	BodyFatMass       *float64  `json:"body_fat_mass"`
	SMM               *float64  `json:"smm"`
	VisceralFat       *float64  `json:"visceral_fat"`
	BMR               *float64  `json:"bmr"`
	TEE               *float64  `json:"tee"`
	LeanLeftArm       *float64  `json:"lean_left_arm"`
	LeanRightArm      *float64  `json:"lean_right_arm"`
	LeanTrunk         *float64  `json:"lean_trunk"`
	LeanLeftLeg       *float64  `json:"lean_left_leg"`
	LeanRightLeg      *float64  `json:"lean_right_leg"`
	FatLeftArm        *float64  `json:"fat_left_arm"`
	FatRightArm       *float64  `json:"fat_right_arm"`
	FatTrunk          *float64  `json:"fat_trunk"`
	FatLeftLeg        *float64  `json:"fat_left_leg"`
	FatRightLeg       *float64  `json:"fat_right_leg"`
	Notes             *string   `json:"notes"`
	ReportPhotoPath   *string   `json:"report_photo_path"`
	ProgressPhotoPath *string   `json:"progress_photo_path"`
	CreatedAt         time.Time `json:"created_at"`
}

// NewScan holds the fields accepted when creating a scan. user_id is never
// taken from the client — it's always the authenticated request's user_id.
type NewScan struct {
	Date              string
	Weight            *float64
	BodyFat           *float64
	LeanBodyMass      *float64
	BodyFatMass       *float64
	SMM               *float64
	VisceralFat       *float64
	BMR               *float64
	TEE               *float64
	LeanLeftArm       *float64
	LeanRightArm      *float64
	LeanTrunk         *float64
	LeanLeftLeg       *float64
	LeanRightLeg      *float64
	FatLeftArm        *float64
	FatRightArm       *float64
	FatTrunk          *float64
	FatLeftLeg        *float64
	FatRightLeg       *float64
	Notes             *string
	ReportPhotoPath   *string
	ProgressPhotoPath *string
}

const scanColumns = `id, date::text, weight, body_fat, lean_body_mass, body_fat_mass, smm,
	visceral_fat, bmr, tee,
	lean_left_arm, lean_right_arm, lean_trunk, lean_left_leg, lean_right_leg,
	fat_left_arm, fat_right_arm, fat_trunk, fat_left_leg, fat_right_leg,
	notes, report_photo_path, progress_photo_path, created_at`

func scanRow(row pgx.Row) (Scan, error) {
	var s Scan
	err := row.Scan(
		&s.ID, &s.Date, &s.Weight, &s.BodyFat, &s.LeanBodyMass, &s.BodyFatMass, &s.SMM,
		&s.VisceralFat, &s.BMR, &s.TEE,
		&s.LeanLeftArm, &s.LeanRightArm, &s.LeanTrunk, &s.LeanLeftLeg, &s.LeanRightLeg,
		&s.FatLeftArm, &s.FatRightArm, &s.FatTrunk, &s.FatLeftLeg, &s.FatRightLeg,
		&s.Notes, &s.ReportPhotoPath, &s.ProgressPhotoPath, &s.CreatedAt,
	)
	return s, err
}

// ListScans returns all scans for userID, most recent first.
func ListScans(ctx context.Context, db *pgxpool.Pool, userID string) ([]Scan, error) {
	rows, err := db.Query(ctx, `select `+scanColumns+` from scans where user_id = $1 order by date desc`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying scans: %w", err)
	}
	defer rows.Close()

	scans := []Scan{}
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning scan row: %w", err)
		}
		scans = append(scans, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating scan rows: %w", err)
	}
	return scans, nil
}

// CreateScan inserts a new scan for userID and returns the stored row.
func CreateScan(ctx context.Context, db *pgxpool.Pool, userID string, s NewScan) (Scan, error) {
	row := db.QueryRow(ctx, `
		insert into scans (
			user_id, date, weight, body_fat, lean_body_mass, body_fat_mass, smm,
			visceral_fat, bmr, tee,
			lean_left_arm, lean_right_arm, lean_trunk, lean_left_leg, lean_right_leg,
			fat_left_arm, fat_right_arm, fat_trunk, fat_left_leg, fat_right_leg,
			notes, report_photo_path, progress_photo_path
		) values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) returning `+scanColumns,
		userID, s.Date, s.Weight, s.BodyFat, s.LeanBodyMass, s.BodyFatMass, s.SMM,
		s.VisceralFat, s.BMR, s.TEE,
		s.LeanLeftArm, s.LeanRightArm, s.LeanTrunk, s.LeanLeftLeg, s.LeanRightLeg,
		s.FatLeftArm, s.FatRightArm, s.FatTrunk, s.FatLeftLeg, s.FatRightLeg,
		s.Notes, s.ReportPhotoPath, s.ProgressPhotoPath,
	)

	scan, err := scanRow(row)
	if err != nil {
		return Scan{}, fmt.Errorf("inserting scan: %w", err)
	}
	return scan, nil
}

// DeleteScan deletes the scan with the given id, but only if it belongs to
// userID. Returns false if no matching row was found (either it doesn't
// exist, or it belongs to someone else) — the handler treats both as 404,
// never revealing which.
func DeleteScan(ctx context.Context, db *pgxpool.Pool, userID, scanID string) (bool, error) {
	tag, err := db.Exec(ctx, `delete from scans where id = $1 and user_id = $2`, scanID, userID)
	if err != nil {
		return false, fmt.Errorf("deleting scan: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
