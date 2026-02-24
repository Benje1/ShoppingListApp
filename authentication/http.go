package authentication

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	mux.Handle("/login", wrap(LoginHandler(db)))
	mux.Handle("/logout", wrap(LogoutHandler()))
	mux.Handle("/profile", RequireAuth(wrap(ProfileHandler())))
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginHandler(db *pgxpool.Pool) httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Println("There is a decoding error")
			return nil, err
		}

		repo := &database.PostgresUserRepo{DB: db}

		return login(r.Context(), req, repo)
	}
}

func login(ctx context.Context, user LoginRequest, repo database.UserRepository) (any, error) {
	err := LoginService(
		ctx,
		repo,
		user.Username,
		user.Password,
	)

	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "ok"}, nil
}

func ProfileHandler() httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		user := r.Header.Get("X-User")
		return map[string]string{
			"message": "Welcome " + user,
		}, nil
	}
}

func LogoutHandler() httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		DestroySession(nil, r)
		return map[string]string{
			"status": "logged out",
		}, nil
	}
}
