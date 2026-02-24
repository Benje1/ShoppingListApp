package households

import (
	"net/http"

	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterHouseholdRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	// mux.HandleFunc("/households/create", func(w http.ResponseWriter, r *http.Request) {
	// 	CreateHousehold(w, r, db)
	// })
}
