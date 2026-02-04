package api

import (
	"context"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUserService(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
	hashed, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}

	return database.InsertUser(
		ctx,
		db,
		input.Name,
		input.Username,
		hashed,
		input.Household,
	)
}

func UpdateUserService(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
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
