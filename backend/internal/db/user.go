package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetOrCreateUser returns the user's id for the given email.
// If no user exists with that email, it creates a new one.
func GetOrCreateUser(ctx context.Context, pool *pgxpool.Pool, email string) (string, error) {
	var userID string

	err := pool.QueryRow(ctx, `
		INSERT INTO users (email)
		VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, email).Scan(&userID)

	if err != nil {
		return "", fmt.Errorf("failed to get or create user: %w", err)
	}

	return userID, nil
}
