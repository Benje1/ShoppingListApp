package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	fmt.Println("Server listening on :8080")
	fmt.Println(http.ListenAndServe(":8080", nil))

	http.HandleFunc("/setup", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		pool := createPool(ctx)
		defer pool.Close()
		w.Header().Set("Content-Type", "application/json")

		items, err := database.GetItemsFromList(pool, ctx)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(items)
		}

	})
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
