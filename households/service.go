package households

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterHouseholdRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	mux.HandleFunc("/households/create", func(w http.ResponseWriter, r *http.Request) {
		CreateHousehold(w, r, db)
	})
}
