package user

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterUserRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	mux.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		CreateUser(w, r, db)
	})
	mux.HandleFunc("/users/update", func(w http.ResponseWriter, r *http.Request) {
		UpdateUser(w, r, db)
	})
}
