package database

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepo struct {
	DB *pgxpool.Pool
}

func (p *PostgresUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string, household uint) (*sqlc.User, error) {
	q := sqlc.New(p.DB)

	user, err := q.InsertUser(ctx, sqlc.InsertUserParams{
		Name:         name,
		Username:     username,
		PasswordHash: passwordHash,
		Household: pgtype.Int4{
			Int32: int32(household),
			Valid: true,
		},
	})
	return &user, err
}

func (p *PostgresUserRepo) UpdateUser(ctx context.Context, username, name, passwordHash string) (*sqlc.User, error) {
	q := sqlc.New(p.DB)

	user, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		Username:     username,
		Name:         name,
		PasswordHash: passwordHash,
	})
	return &user, err
}

func (p *PostgresUserRepo) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	q := sqlc.New(p.DB)

	u, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &u, nil
}
