package database

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresHouseholdRepo struct {
	DB *pgxpool.Pool
}

func (p *PostgresHouseholdRepo) InsertHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	h, err := q.InsertHousehold(ctx, householdID)
	return &h, err
}

func (p *PostgresHouseholdRepo) GetHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	h, err := q.GetHousehold(ctx, householdID)
	return &h, err
}

func (p *PostgresHouseholdRepo) DeleteHousehold(ctx context.Context, householdID int32) error {
	q := sqlc.New(p.DB)
	return q.DeleteHousehold(ctx, householdID)
}
