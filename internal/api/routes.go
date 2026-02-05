package api

import (
	"net/http"

	"weekly-shopping-app/authentication"
	user "weekly-shopping-app/internal/api/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	authentication.RegisterRoutes(mux, db)
	user.RegisterUserRoutes(mux, db)
}
