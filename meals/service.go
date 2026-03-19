package meals

import (
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterMealRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/meals")

	// GET /meals — list all meals with ingredient counts
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				return listMeals(r.Context(), db)
			}
		},
	})

	// GET /meals/get?id= — get a single meal with its ingredients
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/get", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return getMeal(r.Context(), db, id)
			}
		},
	})

	// POST /meals/create — create a meal (optionally with ingredients)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[CreateMealInput]{
		Path: "/create", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, CreateMealInput) (any, error) {
			return func(r *http.Request, input CreateMealInput) (any, error) {
				return createMeal(r.Context(), db, input)
			}
		},
	})

	// POST /meals/update?id= — update name/description/portions
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UpdateMealInput]{
		Path: "/update", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, UpdateMealInput) (any, error) {
			return func(r *http.Request, input UpdateMealInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return updateMeal(r.Context(), db, id, input)
			}
		},
	})

	// DELETE /meals/delete?id=
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/delete", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				if err := deleteMeal(r.Context(), db, id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "deleted"}, nil
			}
		},
	})

	// POST /meals/ingredient/add?id=<meal_id>
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddIngredientInput]{
		Path: "/ingredient/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddIngredientInput) (any, error) {
			return func(r *http.Request, input AddIngredientInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return addIngredient(r.Context(), db, id, input)
			}
		},
	})

	// POST /meals/ingredient/remove?id=<meal_id>
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[RemoveIngredientInput]{
		Path: "/ingredient/remove", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, RemoveIngredientInput) (any, error) {
			return func(r *http.Request, input RemoveIngredientInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return removeIngredient(r.Context(), db, id, input)
			}
		},
	})
}

func queryID(r *http.Request) (int32, error) {
	var id int32
	if _, err := fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id); err != nil || id == 0 {
		return 0, fmt.Errorf("valid id query parameter is required")
	}
	return id, nil
}
