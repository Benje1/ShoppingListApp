package households

import (
	"context"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HouseholdInput struct {
	ID int32 `json:"id"`
	NumPeople int32 `json:"num_people"`
}

func createHousehold(ctx context.Context, db *pgxpool.Pool, input HouseholdInput) (*sqlc.Household, error) {
	repo := &database.PostgresHouseholdRepo{DB: db}
	return repo.InsertHousehold(ctx, input.ID, input.NumPeople)
}

func getHousehold(ctx context.Context, db *pgxpool.Pool, id int32) (*sqlc.Household, error) {
	repo := &database.PostgresHouseholdRepo{DB: db}
	return repo.GetHousehold(ctx, id)
}

func deleteHousehold(ctx context.Context, db *pgxpool.Pool, id int32) error {
	repo := &database.PostgresHouseholdRepo{DB: db}
	return repo.DeleteHousehold(ctx, id)
}
