package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepo struct {
	DB *pgxpool.Pool
}

func (p *PostgresUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string, household uint) error {
	return InsertUser(ctx, p.DB, name, username, passwordHash, household)
}

func (p *PostgresUserRepo) UpdateUser(ctx context.Context, username, name, passwordHash string) error {
	return UpdateUser(ctx, p.DB, username, name, passwordHash)
}

func (p *PostgresUserRepo) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return GetUserByUsername(ctx, p.DB, username)
}
