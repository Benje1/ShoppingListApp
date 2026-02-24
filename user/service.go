package user

import (
	"net/http"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterUserRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	mux.Handle("/users/create", wrap(CreateUserHandler(db)))
	mux.HandleFunc("/users/update", wrap(UpdateUserHandler(db)))
}

func CreateUserHandler(db *pgxpool.Pool) httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		err := CreateUser(w, r, db)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "created"}, nil
	}
}

func UpdateUserHandler(db *pgxpool.Pool) httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		err := UpdateUser(w, r, db)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "created"}, nil
	}
}
