package database

import "context"

type UserRepository interface {
	InsertUser(ctx context.Context, name, username, passwordHash string, household uint) error
	UpdateUser(ctx context.Context, username, name, passwordHash string) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
}
