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

	ctx := context.Background()

	// Run database migrations before opening the connection pool.
	// Already-applied migrations are skipped automatically.
	if err := database.RunMigrations(ctx); err != nil {
		panic(fmt.Sprintf("migrations failed: %v", err))
	}
	fmt.Println("Migrations applied successfully")

	pool, err := database.Conn(ctx)
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
