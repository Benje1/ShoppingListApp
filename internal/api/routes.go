package api

import (
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	authentication.RegisterRoutes(mux, db)
	user.RegisterUserRoutes(mux, db)
}
