package authentication

import (
	"fmt"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/profile", RequireAuth(profile))
}

func login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != "admin" || password != "password" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	CreateSession(w, username)
	fmt.Fprintln(w, "Logged in")
}

func profile(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User")
	fmt.Fprintf(w, "Welcome %s\n", user)
}

func logout(w http.ResponseWriter, r *http.Request) {
	DestroySession(w, r)
	fmt.Fprintln(w, "Logged out")
}
