package profile

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Profile struct {
	HeightCM float64 `json:"height_cm"`
	Gender   string  `json:"gender"`
	Age      int     `json:"age"`
}

var ErrNotFound = pgx.ErrNoRows

func Get(ctx context.Context, db *pgxpool.Pool, userID, environment string) (Profile, error) {
	var p Profile
	err := db.QueryRow(ctx, `select height_cm, gender, age from profiles where user_id = $1 and environment = $2`, userID, environment).
		Scan(&p.HeightCM, &p.Gender, &p.Age)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("querying profile: %w", err)
	}
	return p, nil
}

func Upsert(ctx context.Context, db *pgxpool.Pool, userID, environment string, p Profile) (Profile, error) {
	var out Profile
	err := db.QueryRow(ctx, `
		insert into profiles (user_id, environment, height_cm, gender, age)
		values ($1, $2, $3, $4, $5)
		on conflict (user_id, environment) do update set
			height_cm = excluded.height_cm, gender = excluded.gender, age = excluded.age
		returning height_cm, gender, age`, userID, environment, p.HeightCM, p.Gender, p.Age).
		Scan(&out.HeightCM, &out.Gender, &out.Age)
	if err != nil {
		return Profile{}, fmt.Errorf("upserting profile: %w", err)
	}
	return out, nil
}
