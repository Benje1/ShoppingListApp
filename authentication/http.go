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
	mux.Handle("/login", wrap(loginHandler(db)))
	mux.Handle("/logout", wrap(logoutHandler()))
	mux.Handle("/profile", RequireAuth(wrap(ProfileHandler())))
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Household int32  `json:"household"`
}

func loginHandler(db *pgxpool.Pool) httpx.AppHandler {
	return httpx.PostWithWriter(func(w http.ResponseWriter, r *http.Request, req LoginRequest) (any, error) {
		repo := &database.PostgresUserRepo{DB: db}
		user, err := login(r.Context(), req, repo)
		if err != nil {
			return nil, err
		}

		CreateSession(w, user.Username)

		return UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Username:  user.Username,
			Household: user.Household.Int32,
		}, nil
	})
}

func logoutHandler() httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		DestroySession(w, r)
		return map[string]string{"message": "logged out"}, nil
	}
}

func login(ctx context.Context, user LoginRequest, repo database.UserRepository) (*sqlc.User, error) {
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
