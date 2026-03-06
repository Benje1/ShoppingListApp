package user

import (
	"net/http"
	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterUserRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/users")

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UserInput]{
		Path:   "/create",
		Method: "POST",
		Public: true, 
		Handler: func(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
			return createUserPost(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UserInput]{
		Path:   "/update",
		Method: "POST",
		Public: false, 
		Handler: func(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
			return updateUserPost(db)
		},
	})
}

func createUserPost(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
	return func(r *http.Request, input UserInput) (any, error) {
		user, err := createUser(r.Context(), db, input)
		if err != nil {
			return nil, err
		}

		return user, nil
	}
}

func updateUserPost(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
	return func(r *http.Request, input UserInput) (any, error) {
		// err := updateUser(r.Context(), db, input)
		// if err != nil {
		// 	return nil, err
		// }

		return map[string]string{
			"status": "user updated",
		}, nil
	}
}
