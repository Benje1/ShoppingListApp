package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		login(w, r, db)
	})
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/profile", RequireAuth(profile))
}

func login(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	repo := &database.PostgresUserRepo{DB: db}

	err := LoginService(r.Context(), repo, username, password)
	if err != nil {
		http.Error(w, fmt.Sprintf("username or password incorrect: %s", err.Error()), http.StatusUnauthorized)
		return
	}

	CreateSession(w, username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func profile(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User")
	fmt.Fprintf(w, "Welcome %s\n", user)
}

func logout(w http.ResponseWriter, r *http.Request) {
	DestroySession(w, r)
	fmt.Fprintln(w, "Logged out")
}
