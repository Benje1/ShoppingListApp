package user

import (
	"net/http"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterUserRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	mux.Handle("/users/create", wrap(httpx.Post(createUserPost(db))))
	mux.Handle("/users/update", wrap(httpx.Post(updateUserPost(db))))
}

func createUserPost(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
	return func(r *http.Request, input UserInput) (any, error) {
		user, err := createUser(r.Context(), db, input)
		if err != nil {
			return nil, err
		}

		return user, nil
	}
}

func updateUserPost(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
	return func(r *http.Request, input UserInput) (any, error) {
		// err := updateUser(r.Context(), db, input)
		// if err != nil {
		// 	return nil, err
		// }

		return map[string]string{
			"status": "user updated",
		}, nil
	}
}
