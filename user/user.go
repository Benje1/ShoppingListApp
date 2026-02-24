package user

import (
	"context"
	"errors"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInput struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Household uint   `json:"household"`
}

func CreateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) error {
	var input UserInput
	ok := httpx.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return errors.New("could not decode json")
	}

	err := createUser(r.Context(), db, input)
	if err != nil {
		return err
	}

	return nil
}

func UpdateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) error {
	var input UserInput
	ok := httpx.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return errors.New("could not decode json")
	}

	err := updateUser(r.Context(), db, input)
	if err != nil {
		return err
	}

	return nil
}

func createUser(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}

	return database.InsertUser(ctx, db, input.Name, input.Username, hash, input.Household)
}

func updateUser(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
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
