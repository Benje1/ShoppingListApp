package pantry

import (
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterPantryRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/pantry")

	// GET /pantry — list all pantry items for the user/household
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, householdID, err := userAndHousehold(r)
				if err != nil {
					return nil, err
				}
				return getPantry(r.Context(), db, userID, householdID)
			}
		},
	})

	// POST /pantry/add — add or top-up an item in the pantry
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddToPantryInput]{
		Path: "/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddToPantryInput) (any, error) {
			return func(r *http.Request, input AddToPantryInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return addToPantry(r.Context(), db, userID, input)
			}
		},
	})

	// DELETE /pantry/remove?id= — remove a pantry entry entirely
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/remove", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				var id int32
				if _, err := fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id); err != nil || id == 0 {
					return nil, fmt.Errorf("valid id required")
				}
				return map[string]string{"status": "removed"}, removePantryItem(r.Context(), db, id)
			}
		},
	})

	// POST /pantry/cook — cook a meal, decrement pantry portions
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[CookMealInput]{
		Path: "/cook", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, CookMealInput) (any, error) {
			return func(r *http.Request, input CookMealInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				if input.Portions <= 0 {
					input.Portions = 1
				}
				if err := cookMeal(r.Context(), db, userID, input); err != nil {
					return nil, err
				}
				return map[string]string{"status": "portions decremented"}, nil
			}
		},
	})

	// POST /pantry/shelf-life?item_id= — set shelf life for a shopping item
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[SetShelfLifeInput]{
		Path: "/shelf-life", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, SetShelfLifeInput) (any, error) {
			return func(r *http.Request, input SetShelfLifeInput) (any, error) {
				var itemID int32
				if _, err := fmt.Sscanf(r.URL.Query().Get("item_id"), "%d", &itemID); err != nil || itemID == 0 {
					return nil, fmt.Errorf("valid item_id required")
				}
				return setShelfLife(r.Context(), db, itemID, input)
			}
		},
	})
}

func userAndHousehold(r *http.Request) (int32, int32, error) {
	userID, err := authentication.GetUserID(r)
	if err != nil {
		return 0, 0, err
	}
	var householdID int32
	fmt.Sscanf(r.URL.Query().Get("household_id"), "%d", &householdID)
	return userID, householdID, nil
}
