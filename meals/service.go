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

	// GET /meals/get?id= — get a single meal with ingredients, cooks, and components
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/get", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return getMealFull(r.Context(), db, id)
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

	registerPlanAndCookRoutes(r, db)

	// POST /meals/component/add?id=<parent_meal_id>
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddComponentInput]{
		Path: "/component/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddComponentInput) (any, error) {
			return func(r *http.Request, input AddComponentInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return addComponent(r.Context(), db, id, input)
			}
		},
	})

	// POST /meals/component/remove?id=<parent_meal_id>
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[RemoveComponentInput]{
		Path: "/component/remove", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, RemoveComponentInput) (any, error) {
			return func(r *http.Request, input RemoveComponentInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return removeComponent(r.Context(), db, id, input)
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

// RegisterMealPlanAndCookRoutes adds the meal plan and cook routes.
// Called from RegisterMealRoutes after the base meal routes are registered.
func registerPlanAndCookRoutes(r *Router, db *pgxpool.Pool) {
	// GET /meals/plan — full week plan with meal details
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/plan", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
					sess, err := authentication.SessionFromContext(r)
					if err != nil {
						return nil, err
					}
				return getMealPlanFull(r.Context(), db, sess.UserID, sess.FirstHouseholdID())
			}
		},
	})

	// POST /meals/plan/set — assign a meal to a day
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[SetMealPlanInput]{
		Path: "/plan/set", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, SetMealPlanInput) (any, error) {
			return func(r *http.Request, input SetMealPlanInput) (any, error) {
					sess, err := authentication.SessionFromContext(r)
					if err != nil {
						return nil, err
					}
				return setMealPlanDay(r.Context(), db, sess.UserID, input)
			}
		},
	})

	// POST /meals/plan/clear — remove meal from a day
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[ClearMealPlanInput]{
		Path: "/plan/clear", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, ClearMealPlanInput) (any, error) {
			return func(r *http.Request, input ClearMealPlanInput) (any, error) {
					sess, err := authentication.SessionFromContext(r)
					if err != nil {
						return nil, err
					}
				if err := clearMealPlanDay(r.Context(), db, sess.UserID, input); err != nil {
					return nil, err
				}
				return map[string]string{"status": "cleared"}, nil
			}
		},
	})

	// GET /meals/cooks?id=<meal_id> — list cooks for a meal
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/cooks", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return getMealCooks(r.Context(), db, id)
			}
		},
	})

	// POST /meals/cooks/add?id=<meal_id> — add a cook to a meal
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddCookInput]{
		Path: "/cooks/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddCookInput) (any, error) {
			return func(r *http.Request, input AddCookInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return addMealCook(r.Context(), db, id, input.UserID)
			}
		},
	})

	// POST /meals/cooks/remove?id=<meal_id> — remove a cook from a meal
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddCookInput]{
		Path: "/cooks/remove", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddCookInput) (any, error) {
			return func(r *http.Request, input AddCookInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return removeMealCook(r.Context(), db, id, input.UserID)
			}
		},
	})

	// GET /meals/for-cook?user_id= — meals a specific user can cook
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/for-cook", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getMealsForCook(r.Context(), db, sess.UserID)
			}
		},
	})
}

type Router = httpx.Router
