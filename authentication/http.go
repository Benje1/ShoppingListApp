package authentication

import (
	"context"
	"net/http"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"
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

// LoginResponse is what the frontend receives on successful login.
// Households is the full list of households the user belongs to, each with
// their ID and display name, so the frontend can populate the selector
// without any extra API calls.
type LoginResponse struct {
	ID         int32                `json:"id"`
	Name       string               `json:"name"`
	Username   string               `json:"username"`
	Households []sqlc.UserHousehold `json:"households"`
}

func loginHandlerFn(db *pgxpool.Pool) func(http.ResponseWriter, *http.Request, LoginRequest) (any, error) {
	return func(w http.ResponseWriter, r *http.Request, req LoginRequest) (any, error) {
		repo := &database.PostgresUserRepo{DB: db}
		user, err := login(r.Context(), req, repo)
		if err != nil {
			return nil, err
		}
		// Extract household IDs to store in the session
		householdIds := make([]int32, len(user.Households))
		for i, h := range user.Households {
			householdIds[i] = h.HouseholdID
		}
		CreateSession(w, user.Username, user.ID, householdIds)
		return LoginResponse{
			ID:         user.ID,
			Name:       user.Name,
			Username:   user.Username,
			Households: user.Households,
		}, nil
	}
}

func login(ctx context.Context, user LoginRequest, repo database.UserRepository) (*SafeUser, error) {
	return LoginService(ctx, repo, user.Username, user.Password)
}
