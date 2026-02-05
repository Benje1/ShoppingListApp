package households

import (
	"context"
	"net/http"

	"weekly-shopping-app/database"
	httpapi "weekly-shopping-app/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HouseholdInput struct {
	Id uint `json:"id"`
}

func CreateHousehold(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input HouseholdInput
	ok := httpapi.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return
	}

	err := createHousehold(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Household created"))
}

func createHousehold(ctx context.Context, db *pgxpool.Pool, input HouseholdInput) error {
	return database.InsertHousehold(ctx, db, input.Id)
}
