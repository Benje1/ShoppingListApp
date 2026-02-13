package main

import (
	"context"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api"
	"weekly-shopping-app/internal/api/middleware"

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

	handler := middleware.MiddlewareWrapper(mux)

	authentication.StartSessionCleanup()

	fmt.Println("Server listening on :8080")
	fmt.Println(http.ListenAndServe(":8080", handler))
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
