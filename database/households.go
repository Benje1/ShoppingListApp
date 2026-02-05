package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Household struct {
	Id uint
}

func InsertHousehold(ctx context.Context, db *pgxpool.Pool, householdId uint) error {
	sql := `
		INSERT INTO households (household_id)
		VALUES ($1)
	`

	_, err := db.Exec(ctx, sql, householdId)
	if err != nil {
		return fmt.Errorf("falied to insert household: %w", err)
	}
	return nil
}

func DeleteHouseholdById(ctx context.Context, db *pgxpool.Pool, householdId uint) error {
	sql := `
		DELETE FROM households
		WHERE household_id = $1
	`

	_, err := db.Exec(ctx, sql, householdId)
	if err != nil {
		return fmt.Errorf("failed to delete household: %w", err)
	}
	return nil
}
