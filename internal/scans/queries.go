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
	ID        string    `json:"id"`
	Date      string    `json:"date"`
	Weight    *float64  `json:"weight"`
	BodyFat   *float64  `json:"body_fat"`
	SMM       *float64  `json:"smm"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// NewScan holds the fields accepted when creating a scan. user_id is never
// taken from the client — it's always the authenticated request's user_id.
type NewScan struct {
	Date    string
	Weight  *float64
	BodyFat *float64
	SMM     *float64
	Notes   *string
}

const scanColumns = `id, date::text, weight, body_fat, smm, notes, created_at`

func scanRow(row pgx.Row) (Scan, error) {
	var s Scan
	err := row.Scan(
		&s.ID, &s.Date, &s.Weight, &s.BodyFat, &s.SMM, &s.Notes, &s.CreatedAt,
	)
	return s, err
}

// ListScans returns all scans for userID in environment, most recent first.
func ListScans(ctx context.Context, db *pgxpool.Pool, userID, environment string) ([]Scan, error) {
	rows, err := db.Query(ctx, `select `+scanColumns+` from scans where user_id = $1 and environment = $2 order by date desc`, userID, environment)
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

// CreateScan inserts a new scan for userID in environment and returns it.
func CreateScan(ctx context.Context, db *pgxpool.Pool, userID, environment string, s NewScan) (Scan, error) {
	row := db.QueryRow(ctx, `
		insert into scans (
		user_id, environment, date, weight, body_fat, smm, notes
		) values (
			$1, $2, $3, $4, $5, $6, $7
		) returning `+scanColumns,
		userID, environment, s.Date, s.Weight, s.BodyFat, s.SMM, s.Notes,
	)

	scan, err := scanRow(row)
	if err != nil {
		return Scan{}, fmt.Errorf("inserting scan: %w", err)
	}
	return scan, nil
}

// DeleteScan deletes the scan with the given id, but only if it belongs to
// userID in environment. Returns false if no matching row was found — the
// handler treats missing, wrong-user, and wrong-environment rows as 404.
func DeleteScan(ctx context.Context, db *pgxpool.Pool, userID, environment, scanID string) (bool, error) {
	tag, err := db.Exec(ctx, `delete from scans where id = $1 and user_id = $2 and environment = $3`, scanID, userID, environment)
	if err != nil {
		return false, fmt.Errorf("deleting scan: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
