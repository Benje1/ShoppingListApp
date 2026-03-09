package database

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepo struct {
	DB *pgxpool.Pool
}

func (p *PostgresUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string) (*sqlc.User, error) {
	q := sqlc.New(p.DB)
	user, err := q.InsertUser(ctx, sqlc.InsertUserParams{
		Name:         name,
		Username:     username,
		PasswordHash: passwordHash,
	})
	return &user, err
}

func (p *PostgresUserRepo) AddUserToHousehold(ctx context.Context, userID, householdID int32) error {
	q := sqlc.New(p.DB)
	return q.AddUserToHousehold(ctx, sqlc.AddUserToHouseholdParams{
		UserID:      userID,
		HouseholdID: householdID,
	})
}

func (p *PostgresUserRepo) UpdateUserName(ctx context.Context, username, name string) (*sqlc.User, error) {
	q := sqlc.New(p.DB)
	user, err := q.UpdateUserName(ctx, sqlc.UpdateUserNameParams{
		Username: username,
		Name:     name,
	})
	return &user, err
}

func (p *PostgresUserRepo) UpdateUserPassword(ctx context.Context, username, passwordHash string) (*sqlc.User, error) {
	q := sqlc.New(p.DB)
	user, err := q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		Username:     username,
		PasswordHash: passwordHash,
	})
	return &user, err
}

func (p *PostgresUserRepo) UpdateUserHouseholdMemberships(ctx context.Context, userID, householdID int32) error {
	q := sqlc.New(p.DB)
	args := sqlc.UpdateUserHouseholdMembershipsParams{
		UserID:      userID,
		HouseholdID: householdID,
	}
	return q.UpdateUserHouseholdMemberships(ctx, args)
}

func (p *PostgresUserRepo) GetUserByUsername(ctx context.Context, username string) (*sqlc.GetUserByUsernameRow, error) {
	q := sqlc.New(p.DB)
	row, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &row, nil
}
