package scans

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Goals is one user's target metrics. Any field may be unset (nil) if the
// user hasn't set a target for it yet.
type Goals struct {
	Weight  *float64 `json:"weight"`
	BodyFat *float64 `json:"body_fat"`
	SMM     *float64 `json:"smm"`
}

// ErrGoalsNotFound means the user hasn't set any goals yet.
var ErrGoalsNotFound = errors.New("goals not found")

// GetGoals returns userID's goals, or ErrGoalsNotFound if they haven't set
// any yet.
func GetGoals(ctx context.Context, db *pgxpool.Pool, userID string) (Goals, error) {
	var g Goals
	err := db.QueryRow(ctx, `select weight, body_fat, smm from goals where user_id = $1`, userID).
		Scan(&g.Weight, &g.BodyFat, &g.SMM)
	if errors.Is(err, pgx.ErrNoRows) {
		return Goals{}, ErrGoalsNotFound
	}
	if err != nil {
		return Goals{}, fmt.Errorf("querying goals: %w", err)
	}
	return g, nil
}

// UpsertGoals creates or replaces userID's goals.
func UpsertGoals(ctx context.Context, db *pgxpool.Pool, userID string, g Goals) (Goals, error) {
	var out Goals
	err := db.QueryRow(ctx, `
		insert into goals (user_id, weight, body_fat, smm)
		values ($1, $2, $3, $4)
		on conflict (user_id) do update set
			weight = excluded.weight,
			body_fat = excluded.body_fat,
			smm = excluded.smm
		returning weight, body_fat, smm`,
		userID, g.Weight, g.BodyFat, g.SMM,
	).Scan(&out.Weight, &out.BodyFat, &out.SMM)
	if err != nil {
		return Goals{}, fmt.Errorf("upserting goals: %w", err)
	}
	return out, nil
}
