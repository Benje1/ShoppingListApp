package user

import (
	"net/http"
	httpapi "weekly-shopping-app/http"

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
	var input UserInput
	ok := httpapi.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return
	}

	err := CreateUser(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created"))
}

func updateUser(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var input UserInput
	ok := httpapi.DecodeJSON(w, r, http.MethodPost, input)
	if !ok {
		return
	}

	err := UpdateUser(r.Context(), db, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User updated"))
}
