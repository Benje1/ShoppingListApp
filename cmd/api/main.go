package main

import (
	"context"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api"

	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	pool, err := database.Conn(context.Background())
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, pool)

	authentication.StartSessionCleanup()

	fmt.Println("Server listening on :8080")
	fmt.Println(http.ListenAndServe(":8080", mux))
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
