package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	Id           int
	Name         string
	Household    int
	Username     string
	PasswordHash string
	CreatedAt    string
}

func InsertUser(ctx context.Context, db *pgxpool.Pool, name, username, passwordHash string, household uint) error {
	sql := `
		INSERT INTO users (name, username, password_hash, household)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.Exec(ctx, sql, name, username, passwordHash, household)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func UpdateUser(ctx context.Context, db *pgxpool.Pool, username, newName, newPasswordHash string) error {
	sql := `
        UPDATE users
        SET name = $1, password_hash = $2
        WHERE username = $3
    `
	_, err := db.Exec(ctx, sql, newName, newPasswordHash, username)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func GetUserByUsername(ctx context.Context, db *pgxpool.Pool, username string) (*User, error) {
	sql := `
        SELECT id, name, household, username, password_hash, created_at
        FROM users
        WHERE username = $1
    `
	row := db.QueryRow(ctx, sql, username)
	u := &User{}
	err := row.Scan(&u.Id, &u.Name, &u.Household, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return u, nil
}
