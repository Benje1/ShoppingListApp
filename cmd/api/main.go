package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api"
	"weekly-shopping-app/internal/api/middleware"
	"weekly-shopping-app/pantry"

	"github.com/joho/godotenv"
)

func main() {
	forceMigration := flag.Int("force-migration", -1, "Force a specific migration version (clears dirty state). Example: -force-migration=1")
	flag.Parse()

	loadEnv()

	ctx := context.Background()

	// Recovery mode: clear a dirty migration version and exit.
	if *forceMigration >= 0 {
		if err := database.ForceVersion(*forceMigration); err != nil {
			panic(fmt.Sprintf("force migration failed: %v", err))
		}
		fmt.Println("Dirty state cleared. You can now run the server normally.")
		return
	}

	// Normal startup: apply any pending migrations.
	if err := database.RunMigrations(ctx); err != nil {
		panic(fmt.Sprintf("migrations failed: %v", err))
	}
	fmt.Println("Migrations up to date")

	pool, err := database.Conn(ctx)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, pool)

	handler := middleware.MiddlewareWrapper(mux)

	// Background jobs
	authentication.StartSessionCleanup()
	pantry.StartExpiryScheduler(pool) // marks perishables as expiring_soon / expired every hour

	fmt.Println("Server listening on :8080")
	fmt.Println(http.ListenAndServe(":8080", handler))
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
