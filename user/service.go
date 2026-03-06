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

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UpdateUserInput]{
		Path:   "/update/name",
		Method: "POST",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, UpdateUserInput) (any, error) {
			return updateUserNamePost(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UpdateUserInput]{
		Path:   "/update/password",
		Method: "POST",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, UpdateUserInput) (any, error) {
			return updateUserPasswordPost(db)
		},
	})
}

func createUserPost(db *pgxpool.Pool) func(*http.Request, UserInput) (any, error) {
	return func(r *http.Request, input UserInput) (any, error) {
		return createUser(r.Context(), db, input)
	}
}

func updateUserNamePost(db *pgxpool.Pool) func(*http.Request, UpdateUserInput) (any, error) {
	return func(r *http.Request, input UpdateUserInput) (any, error) {
		return updateUserName(r.Context(), db, input)
	}
}

func updateUserPasswordPost(db *pgxpool.Pool) func(*http.Request, UpdateUserInput) (any, error) {
	return func(r *http.Request, input UpdateUserInput) (any, error) {
		return updateUserPassword(r.Context(), db, input)
	}
}
