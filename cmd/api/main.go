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
	"weekly-shopping-app/internal/logger"
	"weekly-shopping-app/pantry"

	"github.com/joho/godotenv"
)

func main() {
	forceMigration := flag.Int("force-migration", -1, "Force a specific migration version (clears dirty state). Example: -force-migration=1")
	flag.Parse()

	loadEnv()

	// Initialise structured logging to stdout + logs/app-YYYY-MM-DD.log.
	// Must happen before any other code that uses the logger.
	if err := logger.Init("logs"); err != nil {
		panic(fmt.Sprintf("logger init failed: %v", err))
	}

	ctx := context.Background()

	// Recovery mode: clear a dirty migration version and exit.
	if *forceMigration >= 0 {
		if err := database.ForceVersion(*forceMigration); err != nil {
			logger.Error("force migration failed", "err", err)
			panic(fmt.Sprintf("force migration failed: %v", err))
		}
		logger.Info("dirty state cleared, restart the server normally")
		return
	}

	// Normal startup: apply any pending migrations.
	if err := database.RunMigrations(ctx); err != nil {
		logger.Error("migrations failed", "err", err)
		panic(fmt.Sprintf("migrations failed: %v", err))
	}
	logger.Info("migrations up to date")

	pool, err := database.Conn(ctx)
	if err != nil {
		logger.Error("database connection failed", "err", err)
		panic(err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, pool)

	handler := middleware.MiddlewareWrapper(mux)

	// Background jobs
	authentication.StartSessionCleanup()
	pantry.StartExpiryScheduler(pool)

	logger.Info("server listening", "addr", ":8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Error("server stopped", "err", err)
	}
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
