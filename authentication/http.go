package authentication

import (
	"context"
	"net/http"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	mux.Handle("/login", wrap(httpx.Post[LoginRequest](loginHandler(db))))
	mux.Handle("/logout", wrap(httpx.Get(logoutHandler())))
	mux.Handle("/profile", RequireAuth(wrap(ProfileHandler())))
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(db *pgxpool.Pool) func(*http.Request, LoginRequest) (any, error) {
	return func(r *http.Request, req LoginRequest) (any, error) {
		repo := &database.PostgresUserRepo{DB: db}
		return login(r.Context(), req, repo)
	}
}

func logoutHandler() func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		DestroySession(nil, r)

		return map[string]string{
			"status": "logged out",
		}, nil
	}
}

func login(ctx context.Context, user LoginRequest, repo database.UserRepository) (*database.User, error) {
	reUser, err := LoginService(
		ctx,
		repo,
		user.Username,
		user.Password,
	)

	if err != nil {
		return nil, err
	}

	return reUser, nil
}

func ProfileHandler() httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		user := r.Header.Get("X-User")
		return map[string]string{
			"message": "Welcome " + user,
		}, nil
	}
}
