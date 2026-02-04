package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserInput struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Household uint   `json:"household"`
}

func RegisterUserRoutes(mux *http.ServeMux, db *pgxpool.Pool) {
	mux.HandleFunc("/users/create", func(w http.ResponseWriter, r *http.Request) {
		createUser(w, r, db)
	})
	mux.HandleFunc("/users/update", func(w http.ResponseWriter, r *http.Request) {
		updateUser(w, r, db)
	})
}

func createUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := CreateUserService(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created"))
}

func updateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := UpdateUserService(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User updated"))
}
