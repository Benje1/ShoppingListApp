package user

import (
	"context"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInput struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	HouseholdID int32  `json:"household_id"`
}

type UpdateUserInput struct {
	Username    string `json:"username"`
	Name        string `json:"name"`
	Password    string `json:"password"`
	HouseholdID int32  `json:"household_id"`
}

func createUser(ctx context.Context, db *pgxpool.Pool, input UserInput) (*sqlc.User, error) {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	repo := &database.PostgresUserRepo{DB: db}

	user, err := repo.InsertUser(ctx, input.Name, input.Username, hash)
	if err != nil {
		return nil, err
	}

	if input.HouseholdID != 0 {
		if err := repo.AddUserToHousehold(ctx, user.ID, input.HouseholdID); err != nil {
			return nil, err
		}
	}

	return user, nil
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
