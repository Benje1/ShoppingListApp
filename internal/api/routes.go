package api

import (
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/households"
	"weekly-shopping-app/internal/api/httpx"
	shoppinglist "weekly-shopping-app/shoppingList"
	"weekly-shopping-app/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	apiWrap := httpx.Wrap
	authentication.RegisterRoutes(mux, db, apiWrap)
	user.RegisterUserRoutes(mux, db, apiWrap)
	households.RegisterHouseholdRoutes(mux, db, apiWrap)
	shoppinglist.RegisterShoppingListRoutes(mux, db, apiWrap)
}
