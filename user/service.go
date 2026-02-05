package user

import (
	"context"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUser(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}

	return database.InsertUser(ctx, db, input.Name, input.Username, hash, input.Household)
}

func UpdateUser(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
	hashed, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}

	return database.UpdateUser(
		ctx,
		db,
		input.Username,
		input.Name,
		hashed,
	)
}
