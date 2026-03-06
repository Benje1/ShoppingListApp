package database

import (
	"context"
	sqlc "weekly-shopping-app/database/sqlc"
)

type UserRepository interface {
	InsertUser(ctx context.Context, name, username, passwordHash string, household uint) (*sqlc.User, error)
	UpdateUser(ctx context.Context, username, name, passwordHash string) (*sqlc.User, error)
	GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error)
}
