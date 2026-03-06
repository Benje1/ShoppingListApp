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
	id, err := q.InsertHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	return &sqlc.Household{HouseholdID: id}, nil
}

func (p *PostgresHouseholdRepo) GetHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	id, err := q.GetHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	return &sqlc.Household{HouseholdID: id}, nil
}

func (p *PostgresHouseholdRepo) DeleteHousehold(ctx context.Context, householdID int32) error {
	q := sqlc.New(p.DB)
	return q.DeleteHousehold(ctx, householdID)
}
