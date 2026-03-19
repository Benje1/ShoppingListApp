package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func newMigrator() (*migrate.Migrate, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating migrator: %w", err)
	}

	return m, nil
}

// RunMigrations applies all pending up-migrations.
// It is safe to call on every startup — already-applied migrations are skipped.
func RunMigrations(_ context.Context) error {
	m, err := newMigrator()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

// ForceVersion marks a specific migration version as applied and not dirty.
// Use this to recover from a dirty database state after fixing the underlying issue:
//
//	go run ./cmd/api -force-migration=1
func ForceVersion(version int) error {
	m, err := newMigrator()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("forcing version %d: %w", version, err)
	}

	fmt.Printf("Forced migration version to %d (dirty flag cleared)\n", version)
	return nil
}
