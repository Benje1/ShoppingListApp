package authentication

import (
	"context"
	"net/http"

	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, RequireAuth, "")

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[LoginRequest]{
		Path:   "/login",
		Method: "POST",
		Public: true,
		HandlerWithWriter: func(db *pgxpool.Pool) func(http.ResponseWriter, *http.Request, LoginRequest) (any, error) {
			return loginHandlerFn(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/logout",
		Public: true,
		HandlerWithWriter: func(_ *pgxpool.Pool) func(http.ResponseWriter, *http.Request, struct{}) (any, error) {
			return func(w http.ResponseWriter, r *http.Request, _ struct{}) (any, error) {
				DestroySession(w, r)
				return map[string]string{"message": "logged out"}, nil
			}
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/profile",
		Method: "GET",
		Handler: func(_ *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				user := r.Header.Get("X-User")
				return map[string]string{"message": "Welcome " + user}, nil
			}
		},
	})
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID           int32       `json:"id"`
	Name         string      `json:"name"`
	Username     string      `json:"username"`
	HouseholdIds interface{} `json:"household_ids"`
}

func loginHandlerFn(db *pgxpool.Pool) func(http.ResponseWriter, *http.Request, LoginRequest) (any, error) {
	return func(w http.ResponseWriter, r *http.Request, req LoginRequest) (any, error) {
		repo := &database.PostgresUserRepo{DB: db}
		user, err := login(r.Context(), req, repo)
		if err != nil {
			return nil, err
		}
		CreateSession(w, user.Username)
		return UserResponse{
			ID:           user.ID,
			Name:         user.Name,
			Username:     user.Username,
			HouseholdIds: user.HouseholdIds,
		}, nil
	}
}

func login(ctx context.Context, user LoginRequest, repo database.UserRepository) (*SafeUser, error) {
	return LoginService(ctx, repo, user.Username, user.Password)
}
