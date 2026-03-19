package user

import (
	"context"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserInput no longer accepts a household_id — users join households via invite codes.
type UserInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateUserInput struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func createUser(ctx context.Context, db *pgxpool.Pool, input UserInput) (*sqlc.User, error) {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	repo := &database.PostgresUserRepo{DB: db}
	return repo.InsertUser(ctx, input.Name, input.Username, hash)
}

func updateUserName(ctx context.Context, db *pgxpool.Pool, input UpdateUserInput) (*sqlc.User, error) {
	repo := &database.PostgresUserRepo{DB: db}
	return repo.UpdateUserName(ctx, input.Username, input.Name)
}

func updateUserPassword(ctx context.Context, db *pgxpool.Pool, input UpdateUserInput) (*sqlc.User, error) {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	repo := &database.PostgresUserRepo{DB: db}
	return repo.UpdateUserPassword(ctx, input.Username, hash)
}
