package main

import (
	"context"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// "encoding/json"
// "time"

func main() {
	loadEnv()
	http.HandleFunc("/login", login)
	http.HandleFunc("/profile", authentication.RequireAuth(profile))
	http.HandleFunc("/logout", logout)
	authentication.StartSessionCleanup()

	fmt.Println("Server listening on :8080")
	fmt.Println(http.ListenAndServe(":8080", nil))
}

func createPool(ctx context.Context) *pgxpool.Pool {
	pool, err := database.Conn(ctx)
	if err != nil {
		panic(err)
	}
	return pool
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != "admin" || password != "password" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	authentication.CreateSession(w, username)
	fmt.Fprintln(w, "Logged in")
}

func profile(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User")
	fmt.Fprintf(w, "Welcome %s\n", user)
}

func logout(w http.ResponseWriter, r *http.Request) {
	authentication.DestroySession(w, r)
	fmt.Fprintln(w, "Logged out")
}
