package authntication_test

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"weekly-shopping-app/database/sqlc"
)

type FakeUserRepo struct {
	User *database.User
}

func (f *FakeUserRepo) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	if f.User == nil || f.User.Username != username {
		return nil, errors.New("not found")
	}
	return f.User, nil
}

func (f *FakeUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string, household uint) (*database.User, error) {
	f.User = &database.User{
		Name: name, Username: username, PasswordHash: passwordHash, Household: pgtype.Int4{Int32: int32(household)},
	}
	return f.User, nil
}

func (f *FakeUserRepo) UpdateUser(ctx context.Context, username, name, passwordHash string) (*database.User, error) {
	f.User.Name = name
	f.User.PasswordHash = passwordHash
	return f.User, nil
}
