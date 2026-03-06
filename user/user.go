package user

import (
	"context"

	"weekly-shopping-app/authentication"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInput struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Household uint   `json:"household"`
}

// func CreateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) error {
// 	var input UserInput
// 	ok := httpx.DecodeJSON(w, r, http.MethodPost, input)
// 	if !ok {
// 		return errors.New("could not decode json")
// 	}

// 	err := createUser(r.Context(), db, input)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func UpdateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) error {
// 	var input UserInput
// 	ok := httpx.DecodeJSON(w, r, http.MethodPost, input)
// 	if !ok {
// 		return errors.New("could not decode json")
// 	}

// 	err := updateUser(r.Context(), db, input)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func createUser(ctx context.Context, db *pgxpool.Pool, input UserInput) (sqlc.User, error) {
	hash, err := authentication.HashPassword(input.Password)
	if err != nil {
		return sqlc.User{}, err
	}
	q := sqlc.New(db)
	args := sqlc.InsertUserParams{
		Name:         input.Name,
		Username:     input.Username,
		PasswordHash: hash,
		Household:    pgtype.Int4{Int32: int32(input.Household), Valid: true}}
	return q.InsertUser(ctx, args)
}

// func updateUser(ctx context.Context, db *pgxpool.Pool, input UserInput) error {
// 	hashed, err := authentication.HashPassword(input.Password)
// 	if err != nil {
// 		return err
// 	}

// 	return database.UpdateUser(
// 		ctx,
// 		db,
// 		input.Username,
// 		input.Name,
// 		hashed,
// 	)
// }
