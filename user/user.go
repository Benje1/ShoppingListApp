package user

import (
	"context"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	httpapi "weekly-shopping-app/internal/api"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInput struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Household uint   `json:"household"`
}

func CreateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input UserInput
	ok := httpapi.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return
	}

	err := createUser(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created"))
}

func UpdateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input UserInput
	ok := httpapi.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return
	}

	err := updateUser(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User updated"))
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
