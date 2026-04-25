package database

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// database/database.go
func Conn(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("could not get database url")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	// Prevent "prepared statement already exists" errors when the pool
	// reuses a connection that already has a cached statement from sqlc.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	return pgxpool.NewWithConfig(ctx, config)
}
